/**
 * 布局：顶栏、侧栏容器、页头容器、内容区。
 * 全局区域唯一所有者；页面只能通过 router 暴露的 ctx 使用内容区。
 */
import { esc } from './dom.js';
import { bindTheme } from './theme.js';

const SIDEBAR_KEY = 'easyadmin.sidebar';

const BRAND_SVG = '<svg class="brand-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">' +
  '<rect x="2.5" y="4.5" width="19" height="15" rx="2"></rect><path d="M3 6.5l9 6 9-6"></path></svg>';

export function mountLayout(app, options) {
  const opts = options || {};
  const appInfo = opts.app || {};
  const user = opts.user || {};
  const brand = appInfo.name || 'EasyAdmin';
  const version = appInfo.version || '';

  app.innerHTML =
    '<header class="topbar">' +
    '<button class="btn sidebar-toggle" id="btn-sidebar" title="收起或展开侧边栏" data-tip="收起 / 展开侧边栏">☰</button>' +
    '<div class="brand">' + BRAND_SVG +
    '<span class="brand-name">' + esc(brand) + '</span>' +
    '<span class="brand-version" id="brand-version">' + esc(version) + '</span></div>' +
    '<div class="topbar-mid">' +
    '<span class="status-inline"><span class="dot dot-ok" id="conn-dot"></span><span id="conn-text">在线</span></span>' +
    '<span class="topbar-hint" id="viewer-text"></span>' +
    (opts.env ? '<span class="tag tag-blue" id="env-badge">' + esc(opts.env) + '</span>' : '') +
    '</div>' +
    '<div class="topbar-actions">' +
    '<label class="topbar-label">主题<select class="select select-sm" id="theme-select" title="切换界面主题">' +
    '<option value="light">浅色</option><option value="dark">深色</option><option value="green">护眼绿</option>' +
    '</select></label>' +
    '<span class="topbar-user" id="user-name">' + esc(user.display_name || user.username || '') + '</span>' +
    '<button class="btn btn-sm" id="btn-logout">登出</button>' +
    '</div></header>' +
    '<div class="banner hidden" id="unreachable-banner">' +
    '<span>⚠</span><span class="banner-text"><span id="unreachable-text">管理接口不可达：页面不会自动重试，请确认服务在运行后点页面上的刷新按钮。</span>' +
    '<span>最后更新：<span id="banner-last-updated">—</span></span></span></div>' +
    '<div class="layout" id="layout">' +
    '<aside class="sidebar" id="sidebar"><nav class="menu" id="menu"></nav>' +
    '<div class="sidebar-foot">' + esc(appInfo.footer || 'EasyAdmin') + '</div></aside>' +
    '<main class="main" id="main"><div id="page-head"></div><div id="page-body"></div></main>' +
    '</div>';

  const layout = app.querySelector('#layout');
  const menuHost = app.querySelector('#menu');
  const pageHead = app.querySelector('#page-head');
  const pageBody = app.querySelector('#page-body');
  const banner = app.querySelector('#unreachable-banner');
  const bannerText = app.querySelector('#unreachable-text');
  const bannerUpdated = app.querySelector('#banner-last-updated');
  const connDot = app.querySelector('#conn-dot');
  const connText = app.querySelector('#conn-text');
  const viewerText = app.querySelector('#viewer-text');
  const versionEl = app.querySelector('#brand-version');
  const userName = app.querySelector('#user-name');
  const layoutEl = app.querySelector('#layout');
  const sidebarBtn = app.querySelector('#btn-sidebar');

  try {
    if (localStorage.getItem(SIDEBAR_KEY) === 'hidden') layout.classList.add('sidebar-hidden');
  } catch (e) { /* 隐私模式忽略 */ }

  if (sidebarBtn) {
    sidebarBtn.addEventListener('click', () => {
      layout.classList.toggle('sidebar-hidden');
      try {
        localStorage.setItem(SIDEBAR_KEY, layout.classList.contains('sidebar-hidden') ? 'hidden' : 'open');
      } catch (e) { /* 忽略存储异常 */ }
    });
  }
  const logoutBtn = app.querySelector('#btn-logout');
  if (logoutBtn && typeof opts.onLogout === 'function') logoutBtn.addEventListener('click', opts.onLogout);

  bindTheme();

  function setConnection(state) {
    const bad = state === 'bad';
    if (connDot) {
      connDot.classList.remove('dot-ok', 'dot-bad');
      connDot.classList.add(bad ? 'dot-bad' : 'dot-ok');
    }
    if (connText) connText.textContent = bad ? '接口不可达' : '在线';
    if (banner) banner.classList.toggle('hidden', !bad);
    if (bannerText && opts.unreachableText && bad) bannerText.textContent = opts.unreachableText;
    if (bannerUpdated && bad) bannerUpdated.textContent = new Date().toLocaleString();
  }

  return {
    menuHost,
    pageHead,
    pageBody,
    setConnection,
    setViewer(text) { if (viewerText) viewerText.textContent = text || ''; },
    setVersion(text) { if (versionEl) versionEl.textContent = text || ''; },
    setUser(info) {
      if (userName) userName.textContent = (info && (info.display_name || info.username)) || '';
    },
    toggleSidebar() { if (sidebarBtn) sidebarBtn.click(); },
    get layoutEl() { return layoutEl; },
  };
}
