/** 页面共享的小工具：只做展示与转义，不访问 API。 */
import { esc } from '../core/dom.js';
import { badgeHTML } from '../components/badge.js';

export function errText(err) {
  return err && err.message ? err.message : String(err);
}

export function userStatusBadge(status) {
  return status === 'active'
    ? badgeHTML({ text: '启用', tone: 'green' })
    : badgeHTML({ text: '禁用', tone: 'gray' });
}

export function rolesInline(roles) {
  if (!roles || !roles.length) return '<span class="muted">—</span>';
  return roles.map((role) => badgeHTML({ text: role.name || role.code, tone: 'blue' })).join(' ');
}

export function roleOptionsHTML(roles, selected) {
  const picked = selected || [];
  if (!roles || !roles.length) return '<span class="muted">暂无角色</span>';
  return roles.map((role) =>
    '<label class="form-check"><input type="checkbox" name="roles[]" value="' + esc(role.code) + '"' +
    (picked.indexOf(role.code) >= 0 ? ' checked' : '') + '> ' + esc(role.name) + '</label>').join('');
}

export function permissionOptionsHTML(items, selected) {
  const picked = selected || [];
  const groups = {};
  const order = [];
  (items || []).forEach((item) => {
    const key = item.module || 'general';
    if (!groups[key]) {
      groups[key] = [];
      order.push(key);
    }
    groups[key].push(item);
  });
  return order.map((key) =>
    '<div class="card-sub" style="margin:6px 0 2px">' + esc(key) + '</div>' +
    groups[key].map((item) =>
      '<label class="form-check"><input type="checkbox" name="permissions[]" value="' + esc(item.code) + '"' +
      (picked.indexOf(item.code) >= 0 ? ' checked' : '') + '> ' + esc(item.name) +
      ' <span class="cfg-key">' + esc(item.code) + '</span></label>').join('')).join('');
}
