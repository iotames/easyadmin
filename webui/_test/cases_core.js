import { suite, test, ok, eq, deepEq, contains, throws } from './runner.js';
import { esc, attrsHTML, queryString } from '../assets/core/dom.js';
import { fmtInt, fmtBytes, fmtDuration, fmtDateTime, toInputDateTime } from '../assets/core/format.js';
import { createStore } from '../assets/core/store.js';
import { createPerm } from '../assets/core/perm.js';
import { isValidTheme, getTheme } from '../assets/core/theme.js';
import * as data from '../assets/core/data.js';
import * as api from '../assets/core/api.js';
import { flattenMenus, findByPath, firstPagePath } from '../assets/core/menu.js';
import { parsePath } from '../assets/core/router.js';

async function withFetch(stub, fn) {
  const original = window.fetch;
  window.fetch = stub;
  try {
    return await fn();
  } finally {
    window.fetch = original;
  }
}

suite('core', () => {
  test('dom.esc 转义全部 HTML 特殊字符', () => {
    eq(esc('<a href="x">&\'</a>'), '&lt;a href=&quot;x&quot;&gt;&amp;&#39;&lt;/a&gt;');
    eq(esc(null), '');
  });

  test('dom.attrsHTML 跳过空值并支持布尔属性', () => {
    eq(attrsHTML({ id: 'a', hidden: true, x: false, y: null }), ' id="a" hidden');
    eq(attrsHTML({ title: '"q"' }), ' title="&quot;q&quot;"');
  });

  test('dom.queryString 跳过空值', () => {
    eq(queryString({ a: 1, b: '', c: null, d: 'x y' }), '?a=1&d=x+y');
    eq(queryString({}), '');
  });

  test('format 数字 / 字节 / 时长 / 时间', () => {
    eq(fmtInt(1234567), '1,234,567');
    eq(fmtBytes(1536), '1.5 KB');
    eq(fmtDuration(0.424), '424 毫秒');
    eq(fmtDuration(3661), '1 小时 1 分');
    ok(fmtDateTime(null) === '—');
    ok(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/.test(toInputDateTime('2026-01-02T03:04:05Z')));
  });

  test('store 显式读写与订阅取消', () => {
    const s = createStore({ a: 1 });
    let seen = 0;
    const off = s.subscribe('a', (value) => { seen = value; });
    s.set('a', 2);
    eq(s.get('a'), 2);
    eq(seen, 2);
    off();
    s.set('a', 3);
    eq(seen, 2);
    eq(s.snapshot().a, 3);
  });

  test('perm 只做 UI 辅助，通配符生效', () => {
    const p = createPerm(['system.user.read']);
    ok(p.can('system.user.read'));
    ok(!p.can('system.user.write'));
    ok(p.can(''));
    ok(createPerm(['*']).can('anything'));
  });

  test('theme 只接受已登记主题', () => {
    ok(isValidTheme('dark'));
    ok(!isValidTheme('purple'));
    ok(['light', 'dark', 'green'].indexOf(getTheme()) >= 0);
  });

  test('data 供应商：SWR / TTL / 单飞 / 失败兜底', async () => {
    data.reset();
    let calls = 0;
    data.provide('num', { fetch: async () => { calls += 1; return 42; }, ttl: 0, fallback: 1 });
    eq(data.peek('num'), 1);
    const value = await data.load('num');
    eq(value, 42);
    eq(calls, 1);
    data.provide('boom', { fetch: async () => { throw new Error('x'); }, fallback: 'fb' });
    eq(await data.load('boom'), 'fb');
    eq(data.get('boom'), 'fb');
  });

  test('api 成功响应解包 data 并附加 CSRF / 分页归一化', async () => {
    const captured = [];
    await withFetch(async (url, options) => {
      captured.push({ url, options });
      return new Response(JSON.stringify({
        ok: true,
        data: { items: [1], x: 2 },
        meta: { request_id: 'r1', page: 2, page_size: 20, total: 41 },
      }), { status: 200 });
    }, async () => {
      api.configure({ baseUrl: '', onUnauthorized: null, onWarning: null, onUnreachable: null });
      api.setCsrfToken('csrf-1');
      const paged = await api.get('/api/x', { query: { a: 1 } });
      eq(paged.page, 2);
      eq(paged.pageSize, 20);
      eq(paged.total, 41);
      deepEq(paged.items, [1]);
      await api.post('/api/x', { a: 1 });
    });
    contains(captured[0].url, '/api/x?a=1');
    eq(captured[1].options.headers['X-CSRF-Token'], 'csrf-1');
  });

  test('api 失败响应抛出带 code/status/requestId 的 ApiError', async () => {
    const err = await withFetch(async () => new Response(JSON.stringify({
      ok: false,
      error: { code: 'COMMON_VALIDATION_FAILED', message: '请求字段校验失败', details: { fields: { email: ['格式错误'] } } },
      meta: { request_id: 'req-9' },
    }), { status: 422 }), () => throws(() => api.get('/api/x')));
    eq(err.name, 'ApiError');
    eq(err.code, 'COMMON_VALIDATION_FAILED');
    eq(err.status, 422);
    eq(err.requestId, 'req-9');
    deepEq(err.details.fields.email, ['格式错误']);
  });

  test('api 401 触发入口失效回调但仍抛错', async () => {
    let unauthorized = 0;
    api.configure({ onUnauthorized: () => { unauthorized += 1; } });
    const err = await withFetch(async () => new Response('', { status: 401 }),
      () => throws(() => api.get('/api/x')));
    eq(err.status, 401);
    eq(unauthorized, 1);
    api.configure({ onUnauthorized: null });
  });

  test('menu 只从服务端结构派生，不认硬编码路径', () => {
    const menus = [
      { key: 'dashboard', title: '仪表盘', path: '/dashboard' },
      { key: 'system', title: '系统', children: [
        { key: 'system.users', title: '用户', path: '/system/users' },
      ] },
    ];
    deepEq(flattenMenus(menus).map((item) => item.key), ['dashboard', 'system.users']);
    eq(findByPath(menus, '/system/users').title, '用户');
    eq(firstPagePath(menus), '/dashboard');
    eq(findByPath(menus, '/nope'), null);
  });

  test('router.parsePath 规范化路径与查询', () => {
    const route = parsePath('/system/users/', '?page=2&q=x');
    eq(route.path, '/system/users');
    deepEq(route.query, { page: '2', q: 'x' });
    eq(parsePath('/', '').path, '/');
  });
});
