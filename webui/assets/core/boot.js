/** 启动流程：主题 → /api/meta → 初始化 / 登录 / 主布局 → 路由。 */
import { esc } from './dom.js';
import { get, post, configure, setCsrfToken } from './api.js';
import { notify, clearToasts } from './notify.js';
import { store } from './store.js';
import { createPerm } from './perm.js';
import { applyTheme, getTheme, bindTheme } from './theme.js';
import { mountLayout } from './layout.js';
import { mountMenu } from './menu.js';
import { createRouter } from './router.js';
import { initTooltip } from '../components/tooltip.js';
import { form } from '../components/form.js';

let authViewActive = false;
let unreachable = false;
let layoutHandle = null;
let routerHandle = null;
let menuDispose = null;

function appEl() {
  return document.getElementById('app');
}

function renderBrand(name) {
  return '<div class="login-brand"><span class="brand-name">' + esc(name || 'EasyAdmin') + '</span></div>';
}

function renderLogin(message) {
  authViewActive = true;
  clearToasts();
  const app = appEl();
  app.innerHTML =
    '<div class="login-body"><div class="login-card">' +
    renderBrand('EasyAdmin') +
    '<div class="login-title">登录</div>' +
    '<div class="login-hint">请输入管理员账号与密码</div>' +
    (message ? '<div class="form-error" style="margin-bottom:12px">' + esc(message) + '</div>' : '') +
    '<form id="login-form" autocomplete="off">' +
    form.field({
      label: '用户名', field: 'username',
      control: form.input({ id: 'login-username', attrs: { name: 'username', autocomplete: 'username' }, placeholder: '用户名' }),
    }) +
    form.field({
      label: '密码', field: 'password',
      control: form.input({ id: 'login-password', type: 'password', attrs: { name: 'password', autocomplete: 'current-password' }, placeholder: '密码' }),
    }) +
    '<button class="btn btn-primary btn-block" type="submit">登录</button>' +
    '</form>' +
    '<div class="form-hint" style="margin-top:16px">账号或密码错误时不会提示具体是哪一项，以避免枚举账号。</div>' +
    '</div></div>';
  bindTheme();
  const formEl = document.getElementById('login-form');
  if (formEl) formEl.addEventListener('submit', submitLogin);
  const user = document.getElementById('login-username');
  if (user) user.focus();
}

async function submitLogin(event) {
  event.preventDefault();
  const formEl = event.currentTarget;
  const button = formEl.querySelector('button[type="submit"]');
  const values = form.readForm(formEl);
  if (!values.username || !values.password) {
    notify.error('请输入用户名和密码后重试');
    return;
  }
  button.disabled = true;
  button.textContent = '登录中…';
  try {
    await post('/api/auth/login', { username: values.username, password: values.password });
    authViewActive = false;
    await boot();
  } catch (err) {
    notify.error(err && err.message ? err.message : String(err));
    button.disabled = false;
    button.textContent = '登录';
  }
}

function renderSetup(meta) {
  authViewActive = true;
  clearToasts();
  const app = appEl();
  const appName = (meta.app && meta.app.name) || 'EasyAdmin';
  app.innerHTML =
    '<div class="login-body"><div class="login-card">' +
    renderBrand(appName) +
    '<div class="login-title">首次初始化</div>' +
    '<div class="login-hint">系统尚未初始化。创建第一个管理员账号后即可进入后台。</div>' +
    '<form id="setup-form" autocomplete="off">' +
    form.field({ label: '用户名', field: 'username', control: form.input({ id: 'setup-username', attrs: { name: 'username' }, placeholder: 'admin' }) }) +
    form.field({ label: '显示名称', field: 'display_name', control: form.input({ id: 'setup-display', attrs: { name: 'display_name' }, placeholder: '管理员' }) }) +
    form.field({ label: '密码', field: 'password', control: form.input({ id: 'setup-password', type: 'password', attrs: { name: 'password' }, placeholder: '至少 8 位' }) }) +
    form.field({ label: '确认密码', field: 'confirm', control: form.input({ id: 'setup-confirm', type: 'password', attrs: { name: 'confirm' }, placeholder: '再次输入密码' }) }) +
    '<button class="btn btn-primary btn-block" type="submit">创建管理员</button>' +
    '</form>' +
    '<div class="form-hint" style="margin-top:16px">非回环地址初始化需要服务端已配置初始口令；若无法提交，请查看服务端日志。</div>' +
    '</div></div>';
  const formEl = document.getElementById('setup-form');
  if (formEl) formEl.addEventListener('submit', submitSetup);
  const input = document.getElementById('setup-username');
  if (input) input.focus();
}

async function submitSetup(event) {
  event.preventDefault();
  const formEl = event.currentTarget;
  const button = formEl.querySelector('button[type="submit"]');
  const values = form.readForm(formEl);
  if (!values.username || !values.password) {
    notify.error('用户名和密码不能为空');
    return;
  }
  if (values.password !== values.confirm) {
    notify.error('两次输入的密码不一致，请重新确认');
    return;
  }
  button.disabled = true;
  button.textContent = '创建中…';
  try {
    await post('/api/auth/setup', {
      username: values.username,
      display_name: values.display_name || values.username,
      password: values.password,
    });
    authViewActive = false;
    await boot();
  } catch (err) {
    notify.error(err && err.message ? err.message : String(err));
    button.disabled = false;
    button.textContent = '创建管理员';
  }
}

function renderFatal(err) {
  const app = appEl();
  const requestId = err && err.requestId ? '请求编号：' + err.requestId : '';
  app.innerHTML =
    '<div class="login-body"><div class="login-card">' +
    renderBrand('EasyAdmin') +
    '<div class="login-title">无法连接管理服务</div>' +
    '<div class="empty">' + esc(err && err.message ? err.message : String(err)) +
    (requestId ? '<br><span class="muted">' + esc(requestId) + '</span>' : '') + '</div>' +
    '<button class="btn btn-primary btn-block" id="boot-retry">重试</button>' +
    '</div></div>';
  const retry = document.getElementById('boot-retry');
  if (retry) retry.addEventListener('click', () => boot());
}

async function logout() {
  try {
    await post('/api/auth/logout', {});
  } catch (err) {
    if (err && err.status !== 401) notify.error('登出失败：' + (err.message || '未知错误'));
  }
  await boot();
}

function handleUnauthorized() {
  if (authViewActive) return;
  store.set('meta', null);
  renderLogin('会话已失效，请重新登录');
}

function mountApp(meta) {
  const app = appEl();
  app.innerHTML = '';
  authViewActive = false;
  store.set('meta', meta);
  setCsrfToken(meta.csrf_token || '');
  const permissions = createPerm(meta.permissions || []);

  layoutHandle = mountLayout(app, {
    app: meta.app || {},
    user: meta.user || {},
    env: (meta.app && meta.app.env) || '',
    onLogout: logout,
  });
  layoutHandle.setConnection(unreachable ? 'bad' : 'ok');
  menuDispose = mountMenu(layoutHandle.menuHost, meta.menus || [], {
    onNavigate: (path) => { if (routerHandle) routerHandle.navigate(path); },
  });

  routerHandle = createRouter({
    layout: layoutHandle,
    store,
    notify,
    api: { get, post },
    perm: permissions,
  });
  initTooltip();
  routerHandle.start();
}

export async function boot() {
  if (layoutHandle) {
    // 重新登录/重新加载 meta 时重建布局，避免残留上一个用户的状态。
    try { routerHandle && (await routerHandle.stop()); } catch (e) { /* 忽略 */ }
    if (typeof menuDispose === 'function') menuDispose();
    layoutHandle = null;
    routerHandle = null;
    menuDispose = null;
  }
  applyTheme(getTheme());
  configure({
    onUnauthorized: handleUnauthorized,
    onWarning: (message) => notify.warn(message),
    onUnreachable: (bad) => {
      unreachable = !!bad;
      if (layoutHandle) layoutHandle.setConnection(bad ? 'bad' : 'ok');
    },
  });

  let meta;
  try {
    meta = await get('/api/meta', { silent: true });
  } catch (err) {
    if (err && err.status === 401) {
      renderLogin('请先登录');
      return;
    }
    renderFatal(err);
    return;
  }

  if (!meta) {
    renderFatal(new Error('服务端 /api/meta 返回空数据'));
    return;
  }
  if (meta.initialized === false) {
    renderSetup(meta);
    return;
  }
  if (!meta.user) {
    renderLogin('');
    return;
  }
  mountApp(meta);
}

function start() {
  boot().catch((err) => {
    console.error('[easyadmin] 启动失败', err);
    renderFatal(err);
  });
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', start);
} else {
  start();
}

export default { boot };
