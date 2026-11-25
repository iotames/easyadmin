/** 服务端菜单渲染：菜单树来自 /api/meta，前端不硬编码任何菜单项。 */
import { esc } from './dom.js';

const ICONS = {
  grid: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><rect x="3" y="3" width="7" height="7" rx="1"></rect><rect x="14" y="3" width="7" height="7" rx="1"></rect><rect x="3" y="14" width="7" height="7" rx="1"></rect><rect x="14" y="14" width="7" height="7" rx="1"></rect></svg>',
  home: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M3 11l9-8 9 8"></path><path d="M5 10v10h14V10"></path></svg>',
  users: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="9" cy="8" r="3.2"></circle><path d="M3 20c0-3.3 2.7-5.4 6-5.4s6 2.1 6 5.4"></path><path d="M16 11h5v7h-5z"></path><path d="M16 13l2.5 1.8L21 13"></path></svg>',
  settings: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.9.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.9 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.9l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.9.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.9-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.9V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"></path></svg>',
  file: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M6 3h8l4 4v14H6z"></path><path d="M14 3v4h4"></path></svg>',
  task: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M9 5h11"></path><path d="M9 12h11"></path><path d="M9 19h11"></path><path d="M4 5l1.5 1.5L7 5"></path><path d="M4 12l1.5 1.5L7 12"></path><path d="M4 19l1.5 1.5L7 19"></path></svg>',
  database: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><ellipse cx="12" cy="5" rx="8" ry="3"></ellipse><path d="M4 5v14c0 1.7 3.6 3 8 3s8-1.3 8-3V5"></path><path d="M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3"></path></svg>',
  shield: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M12 3l8 3v6c0 4.5-3.2 7.8-8 9-4.8-1.2-8-4.5-8-9V6z"></path><path d="M9 12l2 2 4-4"></path></svg>',
  activity: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M3 12h4l3-8 4 16 3-8h4"></path></svg>',
  list: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M8 6h13"></path><path d="M8 12h13"></path><path d="M8 18h13"></path><circle cx="4" cy="6" r="1"></circle><circle cx="4" cy="12" r="1"></circle><circle cx="4" cy="18" r="1"></circle></svg>',
};

export function flattenMenus(menus) {
  const out = [];
  (menus || []).forEach((node) => {
    if (node.children && node.children.length) {
      node.children.forEach((child) => {
        if (child.path) out.push({ key: child.key, title: child.title, path: child.path, icon: child.icon, group: node.key });
      });
    } else if (node.path) {
      out.push({ key: node.key, title: node.title, path: node.path, icon: node.icon, group: '' });
    }
  });
  return out;
}

export function findByPath(menus, path) {
  const list = flattenMenus(menus);
  return list.filter((item) => item.path === path)[0] || null;
}

export function firstPagePath(menus) {
  const list = flattenMenus(menus);
  return list.length ? list[0].path : '';
}

function iconHTML(name) {
  return ICONS[name] || ICONS.grid;
}

function leafHTML(item, depth) {
  return '<a class="menu-item' + (depth ? ' menu-sub-item' : '') + '" href="' + esc(item.path) + '"' +
    ' data-route="' + esc(item.path) + '" data-key="' + esc(item.key) + '">' +
    (depth ? '' : iconHTML(item.icon)) + ' ' + esc(item.title) + '</a>';
}

export function menuHTML(menus) {
  return (menus || []).map((node) => {
    if (!node.children || !node.children.length) return leafHTML(node, false);
    return '<div class="menu-group" id="menu-group-' + esc(node.key) + '">' +
      '<div class="menu-item menu-parent" data-group="' + esc(node.key) + '">' +
      '<span class="menu-parent-label">' + iconHTML(node.icon) + '<span>' + esc(node.title) + '</span></span>' +
      '<span class="menu-arrow">▾</span></div>' +
      '<div class="menu-sub">' + node.children.map((child) => leafHTML(child, true)).join('') + '</div></div>';
  }).join('');
}

export function mountMenu(host, menus, handlers) {
  if (!host) return () => {};
  const onNavigate = (handlers && handlers.onNavigate) || function noop() {};
  host.innerHTML = menuHTML(menus);

  const onClick = (event) => {
    const parent = event.target.closest('.menu-parent');
    if (parent) {
      const group = parent.closest('.menu-group');
      if (group) group.classList.toggle('open');
      return;
    }
    const link = event.target.closest('[data-route]');
    if (!link) return;
    event.preventDefault();
    onNavigate(link.getAttribute('data-route'));
  };
  host.addEventListener('click', onClick);
  return () => host.removeEventListener('click', onClick);
}

export function setActive(host, path) {
  if (!host) return;
  host.querySelectorAll('[data-route]').forEach((link) => {
    link.classList.toggle('active', link.getAttribute('data-route') === path);
  });
  host.querySelectorAll('.menu-group').forEach((group) => {
    group.classList.toggle('open', !!group.querySelector('[data-route].active'));
  });
}

export const menu = { flattenMenus, findByPath, firstPagePath, menuHTML, mountMenu, setActive };
