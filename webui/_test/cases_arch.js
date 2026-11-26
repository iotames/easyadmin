import { suite, test, ok, eq, contains, notContains } from './runner.js';

const CORE = ['api.js', 'boot.js', 'data.js', 'dom.js', 'format.js', 'layout.js', 'menu.js', 'notify.js', 'perm.js', 'router.js', 'store.js', 'theme.js'];
const COMP = ['action-panel.js', 'badge.js', 'data-table.js', 'date-range.js', 'detail-modal.js', 'dsn-form.js', 'empty.js', 'filter-bar.js', 'form.js', 'log-view.js', 'modal.js', 'page-header.js', 'tabs.js', 'tooltip.js'];

async function text(path) {
  const url = new URL(path, import.meta.url);
  const response = await fetch(url, { cache: 'no-store' });
  if (!response.ok) throw new Error('无法读取 ' + path + '（HTTP ' + response.status + '）');
  return response.text();
}

async function allSources() {
  const entries = [];
  for (const name of CORE) entries.push(['core/' + name, await text('../assets/core/' + name)]);
  for (const name of COMP) entries.push(['components/' + name, await text('../assets/components/' + name)]);
  return entries;
}

suite('architecture', () => {
  test('只有 core/api.js 允许裸 fetch，组件与页面不得绕行', async () => {
    const sources = await allSources();
    const offenders = sources.filter(([name, source]) => /\bfetch\s*\(/.test(source) && name !== 'core/api.js').map(([name]) => name);
    eq(offenders.length, 0, '裸 fetch 文件：' + offenders.join(', '));
  });

  test('组件不得 import api / notify', async () => {
    const sources = await allSources();
    const offenders = sources.filter(([name, source]) =>
      name.startsWith('components/') && /from\s+'\.\.\/core\/(api|notify)\.js'/.test(source)).map(([name]) => name);
    eq(offenders.length, 0, '组件越界依赖：' + offenders.join(', '));
  });

  test('禁止 window.Rock 全局命名空间与原生 alert/confirm', async () => {
    const sources = await allSources();
    sources.forEach(([name, source]) => {
      notContains(source, 'window.Rock', name + ' 仍使用全局命名空间');
      ok(!/\balert\s*\(/.test(source), name + ' 使用了原生 alert');
      ok(!/\b(?:window\.)?confirm\s*\(/.test(source), name + ' 使用了原生 confirm');
    });
  });

  test('提示只经 notify：组件内不出现 toast 拼接', async () => {
    const sources = await allSources();
    const offenders = sources.filter(([name, source]) =>
      name.startsWith('components/') && /toastRaw|class="toast/.test(source)).map(([name]) => name);
    eq(offenders.length, 0, '组件自造提示：' + offenders.join(', '));
  });

  test('刷新按钮全站只有一处定义', async () => {
    const sources = await allSources();
    const hits = sources.reduce((sum, [, source]) => sum + (source.match(/<button[^>]*data-act="page-refresh"/g) || []).length, 0);
    eq(hits, 1);
  });

  test('菜单不硬编码页面标题或路径', async () => {
    const source = await text('../assets/core/menu.js');
    notContains(source, '仪表盘');
    notContains(source, '/system/');
    notContains(source, '/dashboard');
    contains(source, 'data-route');
  });

  test('index.html 极薄：只加载 boot.js，主题首屏前置', async () => {
    const html = await text('../index.html');
    eq((html.match(/type="module"/g) || []).length, 1);
    contains(html, '/assets/core/boot.js');
    contains(html, 'easyadmin.theme');
    contains(html, 'tokens.css');
    contains(html, 'components.css');
    notContains(html, 'nav.js');
  });

  test('生产 embed 不含 _test，dev 构建读磁盘', async () => {
    const prod = await text('../embed.go');
    const dev = await text('../embed_dev.go');
    contains(prod, '//go:embed assets');
    notContains(prod, '//go:embed _test');
    notContains(prod, 'assets/_test');
    contains(dev, 'os.DirFS');
  });

  test('样式令牌覆盖 light/dark/green 三套主题', async () => {
    const tokens = await text('../assets/styles/tokens.css');
    contains(tokens, ':root');
    contains(tokens, '[data-theme="dark"]');
    contains(tokens, '[data-theme="green"]');
    contains(tokens, '--primary');
    const components = await text('../assets/styles/components.css');
    notContains(components, 'rock-api-loading');
    contains(components, 'ea-api-loading');
  });

  test('每个 core 模块都是 ESM 导出，不残留 IIFE', async () => {
    const sources = await allSources();
    sources.forEach(([name, source]) => {
      ok(/\bexport\s/.test(source), name + ' 未导出任何符号');
      notContains(source, "'use strict';\n\n  window.Rock", name + ' 仍残留 IIFE 头');
    });
  });

  test('框架资源不夹带业务名词', async () => {
    const sources = await allSources();
    const banned = ['邮件', '邮箱', 'ERP', '邮集'];
    sources.forEach(([name, source]) => {
      banned.forEach((word) => ok(source.indexOf(word) < 0, name + ' 含业务词：' + word));
    });
  });

  test('API 客户端守住 CSRF 与 request_id 契约', async () => {
    const source = await text('../assets/core/api.js');
    contains(source, 'X-CSRF-Token');
    contains(source, 'request_id');
    contains(source, 'credentials: \'same-origin\'');
    notContains(source, 'localStorage');
  });

  test('启动流程读取 /api/meta 并根据 initialized/user 分流', async () => {
    const source = await text('../assets/core/boot.js');
    contains(source, '/api/meta');
    contains(source, 'initialized === false');
    contains(source, 'meta.user');
    contains(source, '/api/auth/login');
    contains(source, '/api/auth/setup');
  });

  test('页面契约：router 只接受带 mount 的页面并负责 unmount', async () => {
    const source = await text('../assets/core/router.js');
    contains(source, 'typeof page.mount');
    contains(source, 'currentPage.unmount');
    contains(source, 'onCleanup');
    contains(source, 'registerEntries');
  });

  test('六个页面入口存在且导出 mount/unmount', async () => {
    const ids = ['dashboard', 'profile', 'system.audit', 'system.config', 'system.roles', 'system.users'];
    for (const id of ids) {
      const mod = await import('../assets/pages/' + id.replace(/\./g, '/') + '/index.js');
      ok(typeof mod.mount === 'function', id + ' 未导出 mount');
      ok(typeof mod.unmount === 'function', id + ' 未导出 unmount');
      ok(mod.default && typeof mod.default.mount === 'function', id + ' default 未导出 mount');
    }
  });

  test('core/pages.js 注册路径与页面文件一一对应', async () => {
    const source = await text('../assets/core/pages.js');
    const ids = ['dashboard', 'profile', 'system.audit', 'system.config', 'system.roles', 'system.users'];
    ids.forEach((id) => contains(source, "'" + id + "': () => import('../pages/" + id.replace(/\./g, '/') + "/index.js')"));
  });
});
