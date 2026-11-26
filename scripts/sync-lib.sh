#!/usr/bin/env bash
# 同步 lib/ 下的地基库到上游最新代码：
#   - .gitmodules 声明的库走 git submodule update --init --remote --recursive
#   - 未声明的普通 git 仓库走 fetch + 默认分支快进同步
# 由项目根目录的 Makefile 调用：make lib
set -uo pipefail

cd "$(dirname "$0")/.." || exit 1

LIB_DIR=${LIB_DIR:-lib}
GIT_PREFIX=${GIT_PREFIX:-github:iotames}
GIT_PREFIX=${GIT_PREFIX%/}

declare -a FAILED=()
TOTAL=0

info() { printf '%s\n' "$*"; }
fail_repo() { printf '[失败] %s: %s\n' "$1" "$2" >&2; FAILED+=("$1"); }

submodule_name_for() {
	git config -f .gitmodules --get-regexp '^submodule\..*\.path$' 2>/dev/null |
		awk -v path="$1" '$2 == path { name = $1; sub(/^submodule\./, "", name); sub(/\.path$/, "", name); print name; exit }'
}

lib_names() {
	{
		git config -f .gitmodules --get-regexp '^submodule\..*\.path$' 2>/dev/null | awk '{ print $2 }'
		find "$LIB_DIR" -mindepth 1 -maxdepth 1 -type d 2>/dev/null
	} | sed -n "s#^${LIB_DIR}/##p" | grep -v / | sort -u
}

is_dirty() {
	[ -n "$(git -C "$1" status --porcelain --untracked-files=no 2>/dev/null)" ]
}

default_branch() {
	local dir=$1 branch candidate
	branch=$(git -C "$dir" symbolic-ref -q --short refs/remotes/origin/HEAD 2>/dev/null | sed 's#^origin/##')
	if [ -z "$branch" ]; then
		for candidate in main master; do
			if git -C "$dir" show-ref --verify --quiet "refs/remotes/origin/$candidate"; then
				branch=$candidate
				break
			fi
		done
	fi
	if [ -z "$branch" ]; then
		branch=$(git -C "$dir" ls-remote --symref origin HEAD 2>/dev/null |
			awk '/^ref:/ { sub("refs/heads/", "", $2); print $2; exit }')
	fi
	printf '%s' "$branch"
}

sync_submodule() {
	local sub=$1 dir=$2 url=$3
	git config "submodule.$sub.url" "$url" || { fail_repo "$dir" "写入子模块 URL 失败"; return; }
	info "[同步] $dir"
	git submodule update --init --remote --recursive "$dir" ||
		fail_repo "$dir" "子模块同步失败（检查网络、SSH 别名或未提交改动）"
}

sync_plain_repo() {
	local dir=$1 url=$2 remote branch current
	remote=$(git -C "$dir" remote get-url origin 2>/dev/null || true)
	if [ -z "$remote" ]; then
		git -C "$dir" remote add origin "$url" || { fail_repo "$dir" "无法添加 origin"; return; }
	elif [ "$remote" != "$url" ]; then
		info "[注意] $dir: origin 为 $remote，按现有远程同步"
	fi

	branch=$(default_branch "$dir")
	if [ -z "$branch" ]; then
		fail_repo "$dir" "无法确定上游默认分支"
		return
	fi

	current=$(git -C "$dir" symbolic-ref -q --short HEAD 2>/dev/null || true)
	if [ "$current" != "$branch" ]; then
		if is_dirty "$dir"; then
			fail_repo "$dir" "当前分支 ${current:-detached HEAD} 有未提交改动，无法切换到 $branch"
			return
		fi
		if git -C "$dir" show-ref --verify --quiet "refs/heads/$branch"; then
			git -C "$dir" checkout --quiet "$branch" || { fail_repo "$dir" "切换到 $branch 失败"; return; }
		else
			git -C "$dir" checkout --quiet -b "$branch" "origin/$branch" || { fail_repo "$dir" "创建分支 $branch 失败"; return; }
		fi
		info "[切换] $dir: ${current:-detached HEAD} -> $branch"
	fi

	info "[更新] $dir ($branch)"
	git -C "$dir" pull --quiet --ff-only origin "$branch" ||
		fail_repo "$dir" "拉取失败（本地分叉提交、未提交改动或网络问题）"
}

main() {
	local name dir url sub
	mkdir -p "$LIB_DIR"

	names=()
	while IFS= read -r name; do
		names+=("$name")
	done < <(lib_names)
	if [ ${#names[@]} -eq 0 ]; then
		info "未发现 $LIB_DIR/ 下的依赖库，跳过"
		return 0
	fi

	for name in "${names[@]}"; do
		TOTAL=$((TOTAL + 1))
		dir="$LIB_DIR/$name"
		url="$GIT_PREFIX/$name.git"

		if [ ! -e "$dir/.git" ] && [ -d "$dir" ] && [ -n "$(ls -A "$dir" 2>/dev/null)" ]; then
			fail_repo "$name" "$dir 已存在但不是 git 仓库，请手动处理"
			continue
		fi
		if [ -e "$dir/.git" ] && is_dirty "$dir"; then
			fail_repo "$name" "有未提交改动，跳过（请先提交或还原）"
			continue
		fi

		sub=$(submodule_name_for "$dir")
		if [ -n "$sub" ]; then
			sync_submodule "$sub" "$dir" "$url"
		elif [ ! -e "$dir/.git" ]; then
			info "[克隆] $name <- $url"
			git clone --quiet "$url" "$dir" || fail_repo "$name" "克隆失败（检查网络、SSH 别名或 LIB_GIT_PREFIX）"
		else
			sync_plain_repo "$dir" "$url"
		fi
	done

	local ok=$((TOTAL - ${#FAILED[@]}))
	if [ ${#FAILED[@]} -gt 0 ]; then
		printf '完成 %d/%d，失败: %s\n' "$ok" "$TOTAL" "${FAILED[*]}" >&2
		return 1
	fi
	printf '完成 %d/%d 个依赖库同步\n' "$ok" "$TOTAL"
}

main "$@"