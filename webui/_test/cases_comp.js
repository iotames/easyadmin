import { suite, test, ok, eq, deepEq, contains, notContains } from './runner.js';
import { form } from '../assets/components/form.js';
import { badgeHTML } from '../assets/components/badge.js';
import { tabsHTML } from '../assets/components/tabs.js';
import { pageHeaderHTML, REFRESH_BUTTON } from '../assets/components/page-header.js';
import { emptyHTML } from '../assets/components/empty.js';
import { createDataTable } from '../assets/components/data-table.js';
import { createFilterBar } from '../assets/components/filter-bar.js';
import { at, recentDays, DR_PARTS } from '../assets/components/date-range.js';
import { detailModal } from '../assets/components/detail-modal.js';
import { openModal, confirmDialog } from '../assets/components/modal.js';
import { logViewHTML } from '../assets/components/log-view.js';
import { actionPanelHTML } from '../assets/components/action-panel.js';
import { mountMenu, setActive } from '../assets/core/menu.js';

function host(html) {
  const node = document.createElement('div');
  node.innerHTML = html || '';
  document.body.appendChild(node);
  return node;
}

suite('components', () => {
  test('form.input：转义、宽度语义类、禁用与密码类型', () => {
    const html = form.input({ value: '"><script>', width: 'sm', disabled: true, placeholder: 'a"b' });
    contains(html, 'form-w-sm');
    contains(html, 'disabled');
    contains(html, '&quot;&gt;&lt;script&gt;');
    contains(html, 'placeholder="a&quot;b"');
    contains(form.input({ type: 'password' }), 'type="password"');
  });

  test('form.select / switch / checkbox / radio 保持选中态', () => {
    const sel = form.select({ options: [['a', '<A>'], ['b', 'B']], selected: 'b' });
    contains(sel, 'value="b" selected');
    contains(sel, '&lt;A&gt;');
    contains(form.switch({ checked: true, title: '开' }), 'checked');
    contains(form.checkbox({ label: '同意', checked: true }), 'form-check');
    contains(form.radio({ name: 'r', options: [['1', '一']], selected: '1' }), 'checked');
  });

  test('form.applyFieldErrors 映射 422 字段并返回未映射项', () => {
    const box = host(form.field({ label: '邮箱', field: 'profile.email', control: form.input({}) }));
    const unmapped = form.applyFieldErrors(box, { 'profile.email': ['格式错误'], other: ['无控件'] });
    deepEq(unmapped, ['other']);
    const err = box.querySelector('[data-field="profile.email"] .form-error');
    ok(err && !err.hidden);
    eq(err.textContent, '格式错误');
    box.remove();
  });

  test('badge：未知 tone 退回灰标签并转义', () => {
    contains(badgeHTML({ text: '<x>', tone: 'purple' }), 'tag-gray');
    contains(badgeHTML({ text: '<x>' }), '&lt;x&gt;');
  });

  test('tabs：选中态与数量角标', () => {
    const html = tabsHTML([{ name: 'a', label: 'A', count: 3 }, { name: 'b', label: 'B' }], 'a', { act: 'tab' });
    contains(html, 'tab active');
    contains(html, 'tab-count');
    contains(html, 'data-act="tab"');
  });

  test('page-header：刷新按钮全站唯一且可关闭', () => {
    const html = pageHeaderHTML({ title: '用户', description: '列表' });
    eq((html.match(/data-act="page-refresh"/g) || []).length, 1);
    contains(html, '⟳ 刷新');
    notContains(pageHeaderHTML({ title: 'x', refresh: false }), 'page-refresh');
    contains(REFRESH_BUTTON, 'page-refresh');
  });

  test('empty：支持 title/description 与纯文本两种口径', () => {
    contains(emptyHTML({ title: '暂无用户', description: '新建一个' }), 'empty-title');
    contains(emptyHTML({ text: '<x>' }), '&lt;x&gt;');
  });

  test('dataTable 客户端分页：切片、总条数与翻页', () => {
    const table = createDataTable({
      ns: 'users',
      columns: [{ key: 'name', title: '名称' }],
      paging: { pageSize: 2 },
    });
    const rows = [{ id: 1, name: 'a' }, { id: 2, name: 'b' }, { id: 3, name: 'c' }];
    const page1 = table.html(rows);
    contains(page1, '共 3 条');
    contains(page1, '>a<');
    notContains(page1, '>c<');
    table.go(2);
    const page2 = table.html(rows);
    contains(page2, '>c<');
    notContains(page2, '>a<');
    deepEq(table.state(), { page: 2, pageSize: 2, offset: 2 });
  });

  test('dataTable 服务端分页：total 来自入参并回调 onPaging', () => {
    const seen = [];
    const table = createDataTable({
      ns: 'sys',
      columns: [{ key: 'name', title: '名称' }],
      paging: { mode: 'server', pageSize: 20 },
      onPaging: (state) => seen.push(state.page),
    });
    const box = host(table.html([{ id: 1, name: 'a' }], { total: 100 }));
    table.bind(box);
    box.querySelector('[data-dt-page="2"]').click();
    eq(table.state().page, 2);
    deepEq(seen, [2]);
    table.destroy();
    box.remove();
  });

  test('dataTable 多选：单行勾选、换渲染自动清理不可见选中', () => {
    const table = createDataTable({
      ns: 'pick',
      columns: [{ key: 'name', title: '名称' }],
      rowKey: (row) => row.id,
      selectable: true,
    });
    const box = host();
    table.bind(box);
    box.innerHTML = table.html([{ id: 1, name: 'a' }, { id: 2, name: 'b' }]);
    const first = box.querySelector('[data-dt-pick="1"]');
    first.checked = true;
    first.dispatchEvent(new Event('change', { bubbles: true }));
    deepEq(table.selection(), ['1']);
    box.innerHTML = table.html([{ id: 2, name: 'b' }]);
    deepEq(table.selection(), []);
    box.querySelector('[data-dt-pick="2"]').click();
    deepEq(table.selection(), ['2']);
    table.destroy();
    box.remove();
  });

  test('dataTable 空态支持对象描述', () => {
    const table = createDataTable({
      columns: [{ key: 'name', title: '名称' }],
      empty: { title: '暂无数据', description: '换一个筛选条件' },
    });
    const html = table.html([]);
    contains(html, '暂无数据');
    contains(html, '换一个筛选条件');
  });

  test('filterBar：默认值、collect 与 dateRange 键名', () => {
    const bar = createFilterBar({
      ns: 'fb1',
      fields: [
        { type: 'text', key: 'keyword', default: 'abc' },
        { type: 'select', key: 'status', options: [['', '全部'], ['on', '启用']], default: '' },
        { type: 'check', key: 'only', label: '只看异常', default: true },
        { type: 'dateRange', key: 'ts', default: { fromDate: '2026-01-01', fromTime: '00:00', toDate: '2026-01-02', toTime: '23:59' } },
      ],
    });
    const box = host(bar.html());
    const state = bar.state();
    eq(state.keyword, 'abc');
    eq(state.tsfromDate, '2026-01-01');
    eq(state.tstoTime, '23:59');
    box.querySelector('[data-fb="keyword"]').value = 'xyz';
    box.querySelector('[data-fb="only"]').checked = false;
    eq(bar.collect().keyword, 'xyz');
    eq(bar.collect().only, false);
    deepEq(DR_PARTS, ['fromDate', 'fromTime', 'toDate', 'toTime']);
    bar.destroy();
    box.remove();
  });

  test('filterBar：reset 回到声明默认值', () => {
    const bar = createFilterBar({ ns: 'fb2', fields: [{ type: 'text', key: 'q', default: 'd' }] });
    const box = host(bar.html());
    box.querySelector('[data-fb="q"]').value = 'changed';
    bar.collect();
    bar.reset();
    eq(box.querySelector('[data-fb="q"]').value, 'd');
    bar.destroy();
    box.remove();
  });

  test('menu：激活子项时自动展开父分组', () => {
    const box = host();
    mountMenu(box, [
      { key: 'dashboard', title: '仪表盘', path: '/dashboard' },
      { key: 'system', title: '系统', children: [{ key: 'system.users', title: '用户', path: '/system/users' }] },
    ], { onNavigate: () => {} });
    setActive(box, '/system/users');
    const group = box.querySelector('.menu-group');
    ok(group.classList.contains('open'));
    ok(box.querySelector('[data-route="/system/users"]').classList.contains('active'));
    box.remove();
  });

  test('dateRange：严格校验，拒绝 2026-02-31 静默进位', () => {
    eq(at('2026-02-31', '00:00'), null);
    eq(at('2026-13-01', '00:00'), null);
    ok(String(at('2026-02-28', '10:30')).endsWith('Z'));
    const range = recentDays(7);
    ok(range.fromDate && range.toDate);
    ok(range.fromDate <= range.toDate);
  });

  test('detailModal：字段渲染、转义、复制按钮与关闭', () => {
    const overlay = detailModal.show({
      title: '详情',
      row: { name: '<script>x</script>', raw: 'v', long: 'line' },
      fields: [
        { key: 'name', label: '名称' },
        { key: 'raw', label: '富文本', render: () => '<b>ok</b>' },
        { key: 'long', label: '长文本', pre: true, copy: true },
      ],
    });
    ok(overlay);
    contains(overlay.textContent, '<script>x</script>');
    contains(overlay.innerHTML, '&lt;script&gt;');
    contains(overlay.innerHTML, '<b>ok</b>');
    contains(overlay.innerHTML, 'data-copy-value="line"');
    overlay.__close();
    ok(!overlay.isConnected);
  });

  test('openModal 默认宽度 800，confirmDialog 返回布尔', async () => {
    const overlay = openModal({ title: 'T', body: 'B' });
    contains(overlay.innerHTML, 'width:800px');
    overlay.__close();
    const pending = confirmDialog({ message: '确定吗', confirmText: '删除', danger: true });
    const okBtn = document.querySelector('[data-modal-act="ok"]');
    ok(okBtn);
    okBtn.click();
    eq(await pending, true);
  });

  test('dataTable 行详情：点击行打开 detailModal', () => {
    const table = createDataTable({
      ns: 'detail',
      columns: [{ key: 'name', title: '名称' }],
      rowKey: (row) => row.id,
      detail: { title: '详情', fields: [{ key: 'name', label: '名称' }] },
    });
    const box = host();
    table.bind(box);
    box.innerHTML = table.html([{ id: 1, name: 'alpha' }]);
    box.querySelector('tr[data-act="detail-detail"]').click();
    const overlay = document.querySelector('#modal-root .modal-overlay');
    ok(overlay, '未打开详情弹层');
    contains(overlay.textContent, 'alpha');
    overlay.__close();
    table.destroy();
    box.remove();
  });

  test('logView：级别过滤、时间与转义', () => {
    const entries = [
      { seq: 1, level: 'info', message: '<a>', logged_at: '2026-01-01T00:00:00Z' },
      { seq: 2, level: 'error', message: 'boom', attrs: { k: 'v' } },
    ];
    const html = logViewHTML(entries, 'error');
    notContains(html, '&lt;a&gt;');
    contains(html, '[error]');
    contains(html, 'boom');
    contains(html, '&quot;k&quot;');
  });

  test('actionPanel：默认标题与主色按钮', () => {
    const html = actionPanelHTML({ buttons: [{ label: '重建', act: 'rebuild', disabled: true }], hint: '注意' });
    contains(html, '维护');
    contains(html, 'btn-primary');
    contains(html, 'data-act="rebuild"');
    contains(html, 'disabled');
  });
});

