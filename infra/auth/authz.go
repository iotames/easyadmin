package auth

// Wildcard 表示超级管理员的全量权限。
const Wildcard = "*"

// Has 判断权限集合是否满足所需权限；required 为空表示登录即可。
func Has(perms []string, required string) bool {
	if required == "" {
		return true
	}
	for _, p := range perms {
		if p == Wildcard || p == required {
			return true
		}
	}
	return false
}

// HasAny 判断是否满足任一权限。
func HasAny(perms []string, required ...string) bool {
	for _, r := range required {
		if Has(perms, r) {
			return true
		}
	}
	return false
}
