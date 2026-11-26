/** 个人资料：查看账号信息并修改密码。 */
import { esc } from '../../core/dom.js';
import { fmtDateTime } from '../../core/format.js';
import { form } from '../../components/form.js';
import { badgeHTML } from '../../components/badge.js';
import { errText, rolesInline } from '../_shared.js';

let cleanup = null;

function infoHTML(user) {
  return '<div class="card"><div class="card-title">账号信息</div><div class="info-grid">' +
    '<div class="info-item"><span class="k">用户名</span><span class="v">' + esc(user.username || '—') + '</span></div>' +
    '<div class="info-item"><span class="k">显示名</span><span class="v">' + esc(user.display_name || '—') + '</span></div>' +
    '<div class="info-item"><span class="k">状态</span><span class="v">' + (user.status === 'active' ? badgeHTML({ text: '启用', tone: 'green' }) : badgeHTML({ text: '禁用', tone: 'gray' })) + '</span></div>' +
    '<div class="info-item"><span class="k">角色</span><span class="v">' + rolesInline(user.roles) + '</span></div>' +
    '<div class="info-item"><span class="k">最近登录</span><span class="v">' + esc(fmtDateTime(user.last_login_at)) + '</span></div>' +
    '<div class="info-item"><span class="k">创建时间</span><span class="v">' + esc(fmtDateTime(user.created_at)) + '</span></div>' +
    '</div></div>';
}

function passwordFormHTML() {
  return '<div class="card"><div class="card-title">修改密码<span class="card-sub">修改后其他会话失效</span></div>' +
    '<form id="profile-password">' +
    form.field({ label: '原密码', field: 'old_password', required: true, control: form.input({ type: 'password', attrs: { name: 'old_password', autocomplete: 'current-password' } }) }) +
    form.field({ label: '新密码', field: 'new_password', required: true, control: form.input({ type: 'password', attrs: { name: 'new_password', autocomplete: 'new-password' }, placeholder: '至少 8 位' }) }) +
    form.field({ label: '确认新密码', field: 'confirm', required: true, control: form.input({ type: 'password', attrs: { name: 'confirm', autocomplete: 'new-password' } }) }) +
    '<button class="btn btn-primary" type="submit">保存新密码</button>' +
    '</form></div>';
}

export async function mount(ctx) {
  ctx.setTitle('个人资料');
  ctx.setDescription('当前账号信息与密码');
  ctx.el.innerHTML = '<div class="grid-main-side">' +
    '<div id="profile-info"><div class="card"><div class="empty">加载中…</div></div></div>' +
    '<div id="profile-password"></div></div>';

  let alive = true;
  let off = () => {};

  const infoHost = ctx.el.querySelector('#profile-info');
  const passwordHost = ctx.el.querySelector('#profile-password');

  async function submitPassword(event) {
    event.preventDefault();
    const formEl = event.currentTarget;
    const button = formEl.querySelector('button[type="submit"]');
    const values = form.readForm(formEl);
    if (!values.old_password || !values.new_password) {
      ctx.notify.error('请填写原密码与新密码');
      return;
    }
    if (values.new_password !== values.confirm) {
      ctx.notify.error('两次输入的新密码不一致');
      return;
    }
    button.disabled = true;
    button.textContent = '保存中…';
    try {
      await ctx.api.put('/api/auth/password', { old_password: values.old_password, new_password: values.new_password });
      if (!alive) return;
      ctx.notify.success('密码已更新，其他会话已失效');
      formEl.reset();
    } catch (err) {
      if (!alive) return;
      const fields = err && err.details ? err.details.fields : null;
      if (fields) form.applyFieldErrors(formEl, fields);
      ctx.notify.error(errText(err));
    } finally {
      if (button.isConnected) {
        button.disabled = false;
        button.textContent = '保存新密码';
      }
    }
  }

  try {
    const data = await ctx.api.get('/api/auth/me');
    if (!alive) return;
    infoHost.innerHTML = infoHTML(data.user || {});
    passwordHost.innerHTML = passwordFormHTML();
    const formEl = passwordHost.querySelector('#profile-password');
    if (formEl) formEl.addEventListener('submit', submitPassword);
  } catch (err) {
    if (alive) infoHost.innerHTML = '<div class="card"><div class="empty">' + esc(errText(err)) + '</div></div>';
  }

  cleanup = () => {
    alive = false;
    off();
  };
  ctx.onCleanup(cleanup);
}

export function unmount() {
  if (cleanup) {
    cleanup();
    cleanup = null;
  }
}

export default { mount, unmount };
