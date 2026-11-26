/** 仪表盘：当前用户、权限与页面概览，附服务连通性探测。 */
import { esc, bindActions } from '../../core/dom.js';
import { fmtInt } from '../../core/format.js';
import { badgeHTML } from '../../components/badge.js';
import { errText } from '../_shared.js';

let cleanup = null;

export function mount(ctx) {
  const meta = ctx.store.get('meta') || {};
  const user = meta.user || {};
  const app = meta.app || {};
  const permissions = meta.permissions || [];
  const menus = meta.menus || [];
  const pageCount = menus.reduce((sum, node) => sum + (node.children && node.children.length ? node.children.length : (node.path ? 1 : 0)), 0);

  ctx.setTitle('仪表盘');
  ctx.setDescription('运行概览与当前账号可见范围');
  ctx.el.innerHTML =
    '<div class="metric-grid">' +
    '<div class="metric-tile"><div class="metric-label">当前用户</div>' +
    '<div class="metric-value">' + esc(user.display_name || user.username || '—') + '</div>' +
    '<div class="metric-hint">' + esc(user.username || '') + '</div></div>' +
    '<div class="metric-tile"><div class="metric-label">权限码</div>' +
    '<div class="metric-value">' + fmtInt(permissions.indexOf('*') >= 0 ? permissions.length - 1 : permissions.length) + '</div>' +
    '<div class="metric-hint">' + (permissions.indexOf('*') >= 0 ? '超级管理员（通配）' : '已授予的权限') + '</div></div>' +
    '<div class="metric-tile"><div class="metric-label">可见页面</div>' +
    '<div class="metric-value">' + fmtInt(pageCount) + '</div>' +
    '<div class="metric-hint">来自服务端菜单</div></div>' +
    '<div class="metric-tile"><div class="metric-label">运行环境</div>' +
    '<div class="metric-value">' + esc(app.env || '—') + '</div>' +
    '<div class="metric-hint">' + esc(app.version || '') + '</div></div>' +
    '</div>' +
    '<div class="grid-2">' +
    '<div class="card"><div class="card-title">服务状态<span class="card-sub">控制台接口探测</span></div>' +
    '<div id="dash-ping">正在检测…</div>' +
    '<div class="form-inline" style="margin-top:12px"><button class="btn btn-sm" data-act="dash-ping">重新检测</button></div></div>' +
    '<div class="card"><div class="card-title">账号信息</div><div class="info-grid">' +
    '<div class="info-item"><span class="k">用户名</span><span class="v">' + esc(user.username || '—') + '</span></div>' +
    '<div class="info-item"><span class="k">显示名</span><span class="v">' + esc(user.display_name || '—') + '</span></div>' +
    '<div class="info-item"><span class="k">超级管理员</span><span class="v">' + (user.is_super_admin ? badgeHTML({ text: '是', tone: 'green' }) : badgeHTML({ text: '否', tone: 'gray' })) + '</span></div>' +
    '<div class="info-item"><span class="k">应用</span><span class="v">' + esc(app.name || 'EasyAdmin') + '</span></div>' +
    '</div></div></div>';

  let alive = true;
  const pingHost = ctx.el.querySelector('#dash-ping');

  async function ping() {
    if (pingHost) pingHost.innerHTML = '<span class="muted">正在检测…</span>';
    try {
      const data = await ctx.api.get('/api/console/ping', { silent: true });
      if (!alive || !pingHost) return;
      pingHost.innerHTML = '<span class="dot dot-ok"></span> 控制台接口正常' +
        (data && data.pong ? '（pong）' : '');
    } catch (err) {
      if (!alive || !pingHost) return;
      pingHost.innerHTML = '<span class="dot dot-bad"></span> ' + esc(errText(err));
    }
  }

  const off = bindActions(ctx.el, { 'dash-ping': ping });
  cleanup = () => {
    alive = false;
    off();
  };
  ctx.onCleanup(cleanup);
  ping();
}

export function unmount() {
  if (cleanup) {
    cleanup();
    cleanup = null;
  }
}

export default { mount, unmount };
