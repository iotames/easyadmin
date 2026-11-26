/**
 * 路由：路径匹配 → 动态 import 页面入口 → mount/unmount。
 * 页面只能操作 ctx.el；切换页面先 unmount 旧页面并清理监听器/计时器/请求。
 */
import { esc, bindActions as bindActionsDOM } from './dom.js';
import { pageHeaderHTML, refreshButtonHTML } from '../components/page-header.js';
import { emptyHTML } from '../components/empty.js';
import { findByPath, firstPagePath, setActive } from './menu.js';

const entries = Object.create(null);

/** 注册/合并 Page ID → loader 映射。后续业务页面由项目 webui/pages.js 注入。 */
export function registerEntries(map) {
  Object.keys(map || {}).forEach((key) => {
    if (entries[key]) throw new Error('Page ID 重复注册：' + key);
    entries[key] = map[key];
  });
}

export function registeredEntries() {
  return Object.keys(entries);
}

export function parsePath(pathname, search) {
  const path = String(pathname || '/').replace(/\/+$/, '') || '/';
  const query = {};
  new URLSearchParams(search || '').forEach((value, key) => { query[key] = value; });
  return { path, query };
}

export function createRouter(options) {
  const opts = options || {};
  const layout = opts.layout;
  const store = opts.store;
  const notify = opts.notify;
  const api = opts.api;
  const perm = opts.perm;

  let current = null;
  let currentPage = null;
  let disposers = [];
  let token = 0;
  let started = false;

  function meta() { return store.get('meta') || {}; }
  function menus() { return meta().menus || []; }

  function headerState() {
    const menu = current && current.menu;
    return {
      title: (menu && menu.title) || '',
      description: '',
      actions: '',
      refresh: true,
      titleHTML: null,
    };
  }

  function setActions(actions) {
    const state = headerState();
    if (typeof actions === 'string') {
      renderHeaderWith({ actions });
      return;
    }
    const html = (actions || []).map((action) =>
      '<button class="btn btn-sm ' + esc(action.className || '') + '" data-act="' + esc(action.act || '') + '"' +
      (action.disabled ? ' disabled' : '') + (action.title ? ' title="' + esc(action.title) + '"' : '') + '>' +
      esc(action.label || '') + '</button>').join('');
    renderHeaderWith({ actions: html });
  }

  let headerOverrides = {};
  function renderHeaderWith(patch) {
    headerOverrides = Object.assign(headerOverrides, patch || {});
    const base = headerState();
    const state = Object.assign(base, headerOverrides);
    const actions = state.actions + (state.refresh ? refreshButtonHTML() : '');
    layout.pageHead.innerHTML = pageHeaderHTML({
      title: state.title,
      titleHTML: state.titleHTML,
      description: state.description,
      actions,
      refresh: false,
    });
    const title = state.title || (current && current.menu && current.menu.title) || '';
    const appName = (meta().app && meta().app.name) || 'EasyAdmin';
    document.title = title ? title + ' · ' + appName : appName;
  }

  function cleanup() {
    const list = disposers.slice();
    disposers = [];
    list.reverse().forEach((fn) => {
      try { fn(); } catch (e) { console.error('[easyadmin] 页面清理失败', e); }
    });
  }

  async function unmountCurrent() {
    if (currentPage && typeof currentPage.unmount === 'function') {
      try { await currentPage.unmount(); } catch (e) { console.error('[easyadmin] 页面卸载失败', e); }
    }
    cleanup();
    currentPage = null;
    current = null;
  }

  function renderNotice(title, description, actionHTML) {
    layout.pageHead.innerHTML = '';
    layout.pageBody.innerHTML = emptyHTML({ title, description, action: actionHTML || '' });
  }

  async function render() {
    const seq = ++token;
    const route = parsePath(location.pathname, location.search);
    const metaInfo = meta();
    const list = metaInfo.menus || [];

    if (!list.length) {
      await unmountCurrent();
      renderNotice('暂无可用页面', '当前账号没有可见菜单，请联系管理员分配权限后刷新。');
      return;
    }

    const menuItem = findByPath(list, route.path);
    if (!menuItem) {
      await unmountCurrent();
      const first = firstPagePath(list);
      renderNotice('页面不存在', '地址 ' + route.path + ' 没有对应的页面入口。',
        first ? '<a class="btn btn-sm btn-primary" href="' + esc(first) + '" data-act="goto-first">返回首页</a>' : '');
      return;
    }

    setActive(layout.menuHost, route.path);
    const loader = entries[menuItem.key];
    if (typeof loader !== 'function') {
      await unmountCurrent();
      renderNotice('页面入口未注册', '模块 ' + menuItem.key + ' 的前端入口未在 router.entries 中登记，请联系开发者补齐。');
      return;
    }

    let page;
    try {
      const module = await loader();
      if (seq !== token) return;
      page = module && module.default ? module.default : module;
    } catch (err) {
      if (seq !== token) return;
      console.error('[easyadmin] 页面加载失败', err);
      renderNotice('页面加载失败', '模块 ' + menuItem.key + ' 加载失败：' + (err && err.message ? err.message : String(err)) + '。请刷新后重试。');
      return;
    }

    if (!page || typeof page.mount !== 'function') {
      renderNotice('页面契约错误', '模块 ' + menuItem.key + ' 必须导出带 mount(ctx) 的页面对象。');
      return;
    }

    await unmountCurrent();
    if (seq !== token) return;

    current = { path: route.path, query: route.query, menu: menuItem };
    currentPage = page;
    headerOverrides = {};
    renderHeaderWith({});
    layout.pageBody.innerHTML = '';

    const ctx = {
      el: layout.pageBody,
      route: { path: route.path, query: route.query, moduleId: menuItem.key },
      api,
      notify,
      perm,
      store,
      setTitle(title) { renderHeaderWith({ title }); },
      setDescription(description) { renderHeaderWith({ description }); },
      setActions,
      setRefresh(enabled) { renderHeaderWith({ refresh: !!enabled }); },
      // 页面级动作同时覆盖页头（setActions 渲染区）与页面主体，统一经 data-act 分发。
      bindActions(handlers) {
        const offHead = bindActionsDOM(layout.pageHead, handlers);
        const offBody = bindActionsDOM(layout.pageBody, handlers);
        const off = () => { offHead(); offBody(); };
        disposers.push(off);
        return off;
      },
      onCleanup(fn) { if (typeof fn === 'function') disposers.push(fn); },
    };

    try {
      const maybeCleanup = await page.mount(ctx);
      if (typeof maybeCleanup === 'function') disposers.push(maybeCleanup);
    } catch (err) {
      console.error('[easyadmin] 页面挂载失败', err);
      renderNotice('页面初始化失败', (err && err.message ? err.message : String(err)) + '。请刷新后重试。');
    }
  }

  function navigate(path, options) {
    const optsNav = options || {};
    const url = path + (optsNav.query ? '?' + new URLSearchParams(optsNav.query).toString() : '');
    if (optsNav.replace) history.replaceState({}, '', url);
    else history.pushState({}, '', url);
    render();
  }

  async function refresh() {
    if (currentPage && typeof currentPage.refresh === 'function') {
      await currentPage.refresh();
      return;
    }
    await render();
  }

  function start() {
    if (started) return;
    started = true;
    const first = firstPagePath(menus());
    if ((location.pathname === '/' || location.pathname === '') && first) {
      history.replaceState({}, '', first);
    }
    window.addEventListener('popstate', render);
    layout.pageHead.addEventListener('click', (event) => {
      const btn = event.target.closest('[data-act="page-refresh"]');
      if (btn) refresh();
    });
    render();
  }

  async function stop() {
    started = false;
    window.removeEventListener('popstate', render);
    await unmountCurrent();
    layout.pageHead.innerHTML = '';
    layout.pageBody.innerHTML = '';
  }

  return {
    start,
    stop,
    navigate,
    refresh,
    render,
    get current() { return current; },
  };
}

export const router = { registerEntries, registeredEntries, parsePath, createRouter };
