/** 用户管理：筛选、创建、编辑角色、启停与重置密码。 */
import { esc, bindActions } from '../../../core/dom.js';
import { fmtDateTime } from '../../../core/format.js';
import { createDataTable } from '../../../components/data-table.js';
import { createFilterBar } from '../../../components/filter-bar.js';
import { form } from '../../../components/form.js';
import { openModal, confirmDialog } from '../../../components/modal.js';
import { errText, userStatusBadge, rolesInline, roleOptionsHTML } from '../../_shared.js';

let cleanup = null;

export async function mount(ctx) {
  ctx.setTitle('用户管理');
  ctx.setDescription('账号、状态与角色分配');
  ctx.setActions([{ act: 'user-create', label: '新建用户', className: 'btn-primary' }]);
  ctx.el.innerHTML =
    '<div class="card"><div id="users-filter"></div><div id="users-table"></div></div>';

  let alive = true;
  let roles = [];
  let rows = [];
  let off = () => {};

  const filter = createFilterBar({
    ns: 'users',
    live: true,
    fields: [
      { key: 'keyword', placeholder: '用户名 / 显示名', width: '220px' },
      { key: 'status', type: 'select', options: [['', '全部状态'], ['active', '启用'], ['disabled', '禁用']] },
    ],
    onQuery: () => loadUsers(),
  });
  const filterHost = ctx.el.querySelector('#users-filter');
  filterHost.innerHTML = filter.html();
  filter.bind(filterHost);

  const table = createDataTable({
    ns: 'users',
    rowKey: (row) => row.id,
    empty: '暂无用户',
    columns: [
      { key: 'username', title: '用户名' },
      { key: 'display_name', title: '显示名' },
      { key: 'roles', title: '角色', render: (row) => rolesInline(row.roles) },
      { key: 'status', title: '状态', render: (row) => userStatusBadge(row.status) },
      { key: 'last_login_at', title: '最近登录', render: (row) => esc(fmtDateTime(row.last_login_at)) },
    ],
    rowActions: (row) =>
      '<button class="btn btn-sm" data-act="user-edit" data-id="' + esc(row.id) + '">编辑</button> ' +
      '<button class="btn btn-sm" data-act="user-status" data-id="' + esc(row.id) + '">' +
      (row.status === 'active' ? '禁用' : '启用') + '</button> ' +
      '<button class="btn btn-sm" data-act="user-reset" data-id="' + esc(row.id) + '">重置密码</button>',
  });
  const tableHost = ctx.el.querySelector('#users-table');
  table.bind(tableHost);

  function findRow(id) {
    return rows.filter((row) => String(row.id) === String(id))[0] || null;
  }

  async function loadUsers() {
    const query = filter.collect();
    try {
      const data = await ctx.api.get('/api/system/users', { query: { keyword: query.keyword, status: query.status } });
      if (!alive) return;
      rows = data.items || [];
      table.render(rows);
    } catch (err) {
      if (!alive) return;
      tableHost.innerHTML = '<div class="empty">' + esc(errText(err)) + '</div>';
      ctx.notify.error(errText(err));
    }
  }

  function overlayFooter() {
    return '<button class="btn" data-user-act="cancel">取消</button>' +
      '<button class="btn btn-primary" data-user-act="save">保存</button>';
  }

  function openEditor(row) {
    const editing = !!row;
    const selected = row && row.roles ? row.roles.map((role) => role.code) : [];
    const body =
      '<form id="user-form">' +
      form.field({
        label: '用户名', field: 'username', required: !editing,
        control: form.input({ attrs: { name: 'username', value: row ? row.username : '' }, disabled: editing }),
        hint: editing ? '用户名创建后不可修改' : '登录账号，创建后不可修改',
      }) +
      form.field({
        label: '显示名', field: 'display_name', required: true,
        control: form.input({ attrs: { name: 'display_name', value: row ? row.display_name : '' } }),
      }) +
      (editing ? '' :
        form.field({ label: '初始密码', field: 'password', required: true, control: form.input({ type: 'password', attrs: { name: 'password', autocomplete: 'new-password' }, placeholder: '至少 8 位' }) }) +
        form.field({ label: '确认密码', field: 'confirm', required: true, control: form.input({ type: 'password', attrs: { name: 'confirm', autocomplete: 'new-password' } }) })) +
      form.field({ label: '角色', field: 'roles', control: form.checkList(roleOptionsHTML(roles, selected)) }) +
      '</form>';
    const overlay = openModal({ title: editing ? '编辑用户' : '新建用户', width: 560, body, footer: overlayFooter() });
    const formEl = overlay.querySelector('#user-form');
    const saveBtn = overlay.querySelector('[data-user-act="save"]');
    overlay.addEventListener('click', async (event) => {
      const action = event.target.closest('[data-user-act]');
      if (!action) return;
      if (action.getAttribute('data-user-act') === 'cancel') {
        overlay.__close();
        return;
      }
      const values = form.readForm(formEl);
      if (!values.display_name) {
        ctx.notify.error('显示名不能为空');
        return;
      }
      if (!editing && (!values.username || !values.password)) {
        ctx.notify.error('用户名与初始密码不能为空');
        return;
      }
      if (!editing && values.password !== values.confirm) {
        ctx.notify.error('两次输入的密码不一致');
        return;
      }
      saveBtn.disabled = true;
      saveBtn.textContent = '保存中…';
      try {
        const payload = { display_name: values.display_name, roles: values.roles || [] };
        if (editing) {
          await ctx.api.put('/api/system/users/' + row.id, payload);
        } else {
          payload.username = values.username;
          payload.password = values.password;
          await ctx.api.post('/api/system/users', payload);
        }
        overlay.__close();
        ctx.notify.success(editing ? '用户已更新' : '用户已创建');
        await loadUsers();
      } catch (err) {
        const fields = err && err.details ? err.details.fields : null;
        if (fields) form.applyFieldErrors(formEl, fields);
        ctx.notify.error(errText(err));
      } finally {
        if (saveBtn.isConnected) {
          saveBtn.disabled = false;
          saveBtn.textContent = '保存';
        }
      }
    });
  }

  function openReset(row) {
    const body = '<form id="reset-form">' +
      form.field({ label: '新密码', field: 'new_password', required: true, control: form.input({ type: 'password', attrs: { name: 'new_password', autocomplete: 'new-password' }, placeholder: '至少 8 位' }) }) +
      form.field({ label: '确认密码', field: 'confirm', required: true, control: form.input({ type: 'password', attrs: { name: 'confirm', autocomplete: 'new-password' } }) }) +
      '</form>';
    const overlay = openModal({ title: '重置密码 · ' + row.username, width: 460, body, footer: overlayFooter() });
    const formEl = overlay.querySelector('#reset-form');
    const saveBtn = overlay.querySelector('[data-user-act="save"]');
    overlay.addEventListener('click', async (event) => {
      const action = event.target.closest('[data-user-act]');
      if (!action) return;
      if (action.getAttribute('data-user-act') === 'cancel') {
        overlay.__close();
        return;
      }
      const values = form.readForm(formEl);
      if (!values.new_password) {
        ctx.notify.error('新密码不能为空');
        return;
      }
      if (values.new_password !== values.confirm) {
        ctx.notify.error('两次输入的密码不一致');
        return;
      }
      saveBtn.disabled = true;
      try {
        await ctx.api.put('/api/system/users/' + row.id + '/password', { new_password: values.new_password });
        overlay.__close();
        ctx.notify.success('密码已重置，该用户的会话已失效');
      } catch (err) {
        const fields = err && err.details ? err.details.fields : null;
        if (fields) form.applyFieldErrors(formEl, fields);
        ctx.notify.error(errText(err));
      } finally {
        if (saveBtn.isConnected) saveBtn.disabled = false;
      }
    });
  }

  async function toggleStatus(row) {
    const next = row.status === 'active' ? 'disabled' : 'active';
    const ok = await confirmDialog({
      title: next === 'disabled' ? '禁用用户' : '启用用户',
      message: '确认' + (next === 'disabled' ? '禁用' : '启用') + '用户“' + esc(row.username) + '”？',
      danger: next === 'disabled',
      confirmText: next === 'disabled' ? '禁用' : '启用',
    });
    if (!ok || !alive) return;
    try {
      await ctx.api.put('/api/system/users/' + row.id + '/status', { status: next });
      ctx.notify.success(next === 'disabled' ? '用户已禁用' : '用户已启用');
      await loadUsers();
    } catch (err) {
      ctx.notify.error(errText(err));
    }
  }

  off = bindActions(ctx.el, {
    'user-create': () => openEditor(null),
    'user-edit': (el) => { const row = findRow(el.getAttribute('data-id')); if (row) openEditor(row); },
    'user-status': (el) => { const row = findRow(el.getAttribute('data-id')); if (row) toggleStatus(row); },
    'user-reset': (el) => { const row = findRow(el.getAttribute('data-id')); if (row) openReset(row); },
  });

  cleanup = () => {
    alive = false;
    off();
    table.destroy();
    filter.destroy();
  };
  ctx.onCleanup(cleanup);

  try {
    const data = await ctx.api.get('/api/system/roles');
    if (!alive) return;
    roles = data.items || [];
  } catch (err) {
    if (alive) ctx.notify.error('读取角色失败：' + errText(err));
  }
  await loadUsers();
}

export function unmount() {
  if (cleanup) {
    cleanup();
    cleanup = null;
  }
}

export default { mount, unmount };
