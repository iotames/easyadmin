/** 角色管理：创建、编辑权限与删除非系统角色。 */
import { esc, bindActions } from '../../../core/dom.js';
import { createDataTable } from '../../../components/data-table.js';
import { form } from '../../../components/form.js';
import { openModal, confirmDialog } from '../../../components/modal.js';
import { badgeHTML } from '../../../components/badge.js';
import { errText, permissionOptionsHTML } from '../../_shared.js';

let cleanup = null;

export async function mount(ctx) {
  ctx.setTitle('角色管理');
  ctx.setDescription('角色编码、权限授予与系统角色保护');
  ctx.setActions([{ act: 'role-create', label: '新建角色', className: 'btn-primary' }]);
  ctx.el.innerHTML = '<div class="card"><div id="roles-table"></div></div>';

  let alive = true;
  let roles = [];
  let permissions = [];
  let off = () => {};

  const table = createDataTable({
    ns: 'roles',
    rowKey: (row) => row.id,
    empty: '暂无角色',
    columns: [
      { key: 'code', title: '编码', render: (row) => '<span class="mono">' + esc(row.code) + '</span>' },
      { key: 'name', title: '名称' },
      { key: 'description', title: '说明' },
      { key: 'is_system', title: '类型', render: (row) => row.is_system ? badgeHTML({ text: '系统角色', tone: 'orange' }) : badgeHTML({ text: '自定义', tone: 'blue' }) },
    ],
    rowActions: (row) =>
      '<button class="btn btn-sm" data-act="role-edit" data-id="' + esc(row.id) + '">编辑权限</button> ' +
      (row.is_system ? '' : '<button class="btn btn-sm btn-danger" data-act="role-delete" data-id="' + esc(row.id) + '">删除</button>'),
  });
  table.bind(ctx.el.querySelector('#roles-table'));

  function findRole(id) {
    return roles.filter((row) => String(row.id) === String(id))[0] || null;
  }

  async function load() {
    try {
      const [roleData, permData] = await Promise.all([
        ctx.api.get('/api/system/roles'),
        ctx.api.get('/api/system/permissions'),
      ]);
      if (!alive) return;
      roles = roleData.items || [];
      permissions = (permData.items || []).filter((item) => item.status === 'active');
      table.render(roles);
    } catch (err) {
      if (!alive) return;
      ctx.notify.error(errText(err));
    }
  }

  function openEditor(row) {
    const editing = !!row;
    const detail = editing ? ctx.api.get('/api/system/roles/' + row.id) : Promise.resolve({ permissions: [] });
    detail.then((data) => {
      if (!alive) return;
      const selected = (data && data.permissions) || [];
      const body = '<form id="role-form">' +
        form.field({
          label: '角色编码', field: 'code', required: !editing,
          control: form.input({ attrs: { name: 'code', value: row ? row.code : '' }, disabled: editing }),
          hint: editing ? '编码创建后不可修改' : '小写字母、数字与下划线，如 operator',
        }) +
        form.field({ label: '角色名称', field: 'name', required: true, control: form.input({ attrs: { name: 'name', value: row ? row.name : '' } }) }) +
        form.field({ label: '说明', field: 'description', control: form.textarea({ attrs: { name: 'description' }, rows: 2, value: row ? row.description : '' }) }) +
        form.field({ label: '权限', field: 'permissions', control: form.checkList(permissionOptionsHTML(permissions, selected)), hint: editing ? '系统角色权限由代码固定，不可修改' : '按模块勾选权限码' }) +
        '</form>';
      const overlay = openModal({
        title: editing ? '编辑角色 · ' + row.name : '新建角色',
        width: 620,
        body,
        footer: '<button class="btn" data-role-act="cancel">取消</button>' +
          '<button class="btn btn-primary" data-role-act="save"' + (editing && row.is_system ? ' disabled' : '') + '>保存</button>',
      });
      const formEl = overlay.querySelector('#role-form');
      const saveBtn = overlay.querySelector('[data-role-act="save"]');
      overlay.addEventListener('click', async (event) => {
        const action = event.target.closest('[data-role-act]');
        if (!action) return;
        if (action.getAttribute('data-role-act') === 'cancel') {
          overlay.__close();
          return;
        }
        const values = form.readForm(formEl);
        if (!values.name) {
          ctx.notify.error('角色名称不能为空');
          return;
        }
        if (!editing && !values.code) {
          ctx.notify.error('角色编码不能为空');
          return;
        }
        saveBtn.disabled = true;
        try {
          const payload = { name: values.name, description: values.description || '', permissions: values.permissions || [] };
          if (editing) {
            await ctx.api.put('/api/system/roles/' + row.id, payload);
          } else {
            payload.code = values.code;
            await ctx.api.post('/api/system/roles', payload);
          }
          overlay.__close();
          ctx.notify.success(editing ? '角色已更新' : '角色已创建');
          await load();
        } catch (err) {
          const fields = err && err.details ? err.details.fields : null;
          if (fields) form.applyFieldErrors(formEl, fields);
          ctx.notify.error(errText(err));
        } finally {
          if (saveBtn.isConnected) saveBtn.disabled = false;
        }
      });
    }).catch((err) => {
      ctx.notify.error('读取角色详情失败：' + errText(err));
    });
  }

  async function removeRole(row) {
    const ok = await confirmDialog({
      title: '删除角色',
      message: '确认删除角色“' + esc(row.name) + '”？删除前请确认没有用户仍引用该角色。',
      danger: true,
      confirmText: '删除',
    });
    if (!ok || !alive) return;
    try {
      await ctx.api.del('/api/system/roles/' + row.id);
      ctx.notify.success('角色已删除');
      await load();
    } catch (err) {
      ctx.notify.error(errText(err));
    }
  }

  off = bindActions(ctx.el, {
    'role-create': () => openEditor(null),
    'role-edit': (el) => { const row = findRole(el.getAttribute('data-id')); if (row) openEditor(row); },
    'role-delete': (el) => { const row = findRole(el.getAttribute('data-id')); if (row) removeRole(row); },
  });

  cleanup = () => {
    alive = false;
    off();
    table.destroy();
  };
  ctx.onCleanup(cleanup);
  await load();
}

export function unmount() {
  if (cleanup) {
    cleanup();
    cleanup = null;
  }
}

export default { mount, unmount };
