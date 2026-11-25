/**
 * 通用数据表格：列声明 + 行渲染 + 客户端/服务端分页 + 行详情 + 多选。
 * 列 render / rowActions 返回 HTML 属调用方显式信任边界，其余值默认转义。
 */
import { esc } from '../core/dom.js';
import { fmtInt } from '../core/format.js';
import { emptyHTML } from './empty.js';
import { detailModal } from './detail-modal.js';

function titleOf(column) {
  return column.title != null ? column.title : column.label;
}

export function createDataTable(config) {
  const cfg = config || {};
  const ns = cfg.ns || 'dt';
  const pagingCfg = cfg.paging || {};
  const mode = pagingCfg.mode || 'client';
  const sizeOptions = pagingCfg.pageSizeOptions || [20, 50, 100];
  const selectable = !!cfg.selectable;
  let page = 1;
  let pageSize = pagingCfg.pageSize || 20;
  let total = 0;
  let picked = {};
  let host = null;
  let lastRows = [];
  const disposers = [];

  const pages = () => Math.max(1, Math.ceil(total / pageSize));
  const rowKeyOf = (row) => (cfg.rowKey ? String(cfg.rowKey(row)) : String(row == null ? '' : row.id));
  const selection = () => Object.keys(picked);
  const clearSelection = () => { picked = {}; };
  const state = () => ({ page, pageSize, offset: (page - 1) * pageSize });

  function notifySelect() {
    if (typeof cfg.onSelect === 'function') cfg.onSelect(selection());
  }

  function go(n) {
    const wanted = Math.floor(Number(n) || 0);
    page = Math.min(Math.max(1, wanted), pages());
  }

  function setPageSize(n) {
    pageSize = Number(n) || 20;
    page = 1;
  }

  function reset() {
    page = 1;
    return state();
  }

  function cellHTML(column, row) {
    if (typeof column.render === 'function') return column.render(row);
    const value = row[column.key];
    return esc(value == null ? '' : String(value));
  }

  function cellCls(column) {
    const cls = [];
    if (column.cls) cls.push(column.cls);
    if (column.nowrap) cls.push('nowrap');
    return cls.length ? ' class="' + esc(cls.join(' ')) + '"' : '';
  }

  function rowHTML(row) {
    const cls = 'dt-row' + (cfg.rowClass ? ' ' + cfg.rowClass(row) : '');
    const key = rowKeyOf(row);
    let html = '<tr class="' + cls + (cfg.detail ? ' dt-row-clickable' : '') + '"' +
      (cfg.detail ? ' data-act="' + esc(ns) + '-detail" data-key="' + esc(key) + '"' : '') + '>';
    if (selectable) {
      html += '<td class="dt-pick"><input type="checkbox" data-dt-pick="' + esc(key) + '"' +
        (picked[key] ? ' checked' : '') + ' aria-label="选择该行"></td>';
    }
    html += cfg.columns.map((column) => '<td' + cellCls(column) + '>' + cellHTML(column, row) + '</td>').join('');
    if (cfg.rowActions) html += '<td class="dt-row-actions">' + cfg.rowActions(row) + '</td>';
    return html + '</tr>';
  }

  function pickHeadHTML(rows) {
    if (!selectable) return '';
    const pickedOnPage = rows.filter((row) => picked[rowKeyOf(row)]).length;
    const all = rows.length > 0 && pickedOnPage === rows.length;
    return '<th class="dt-pick"><input type="checkbox" data-dt-pick-all' + (all ? ' checked' : '') +
      ' aria-label="全选本页"></th>';
  }

  function pagingHTML() {
    const left = '<div class="dt-paging-left">共 ' + fmtInt(total) + ' 条 · 每页 ' +
      '<select class="select select-sm" data-dt-size>' +
      sizeOptions.map((n) => '<option value="' + n + '"' + (n === pageSize ? ' selected' : '') + '>' + n + '</option>').join('') +
      '</select></div>';
    if (pages() <= 1) return '<div class="dt-paging">' + left + '</div>';
    const current = Math.min(page, pages());
    return '<div class="dt-paging">' + left + '<div class="dt-paging-right">' +
      '<button class="btn btn-sm" data-dt-page="' + (current - 1) + '"' + (current <= 1 ? ' disabled' : '') + '>‹ 上一页</button>' +
      '<span class="muted">第</span>' +
      '<input type="number" class="input input-sm dt-page-input" data-dt-page-input min="1" max="' + pages() + '" value="' + current + '">' +
      '<span class="muted">/ ' + pages() + ' 页</span>' +
      '<button class="btn btn-sm" data-dt-page="' + (current + 1) + '"' + (current >= pages() ? ' disabled' : '') + '>下一页 ›</button>' +
      '</div></div>';
  }

  function html(rows, options) {
    const data = rows || [];
    lastRows = data;
    const opts = options || {};
    if (mode === 'server') total = Number(opts.total) || 0;
    else total = data.length;
    if (page > pages()) page = pages();

    if (selectable) {
      const current = {};
      data.forEach((row) => { current[rowKeyOf(row)] = true; });
      Object.keys(picked).forEach((key) => { if (!current[key]) delete picked[key]; });
    }

    let body;
    if (!data.length) {
      body = cfg.empty && typeof cfg.empty === 'object'
        ? emptyHTML(cfg.empty)
        : emptyHTML({ text: cfg.emptyText || cfg.empty || '暂无数据' });
    } else {
      const shown = mode === 'client' ? data.slice((page - 1) * pageSize, page * pageSize) : data;
      const head = pickHeadHTML(shown) + cfg.columns.map((column) =>
        '<th' + (column.width ? ' style="width:' + esc(column.width) + '"' : '') + '>' +
        esc(titleOf(column)) + '</th>').join('') +
        (cfg.rowActions ? '<th style="width:' + esc(cfg.rowActionsWidth || '140px') + '">操作</th>' : '');
      body = '<div class="table-wrap"' + (opts.maxHeight ? ' style="max-height:' + esc(opts.maxHeight) + '"' : '') + '>' +
        '<table class="table"><thead><tr>' + head + '</tr></thead><tbody>' +
        shown.map(rowHTML).join('') + '</tbody></table></div>';
    }
    return body + pagingHTML();
  }

  function syncPickAll() {
    if (!host) return;
    const all = host.querySelector('[data-dt-pick-all]');
    if (!all) return;
    const boxes = host.querySelectorAll('[data-dt-pick]');
    let checked = 0;
    boxes.forEach((box) => { if (box.checked) checked += 1; });
    all.checked = boxes.length > 0 && checked === boxes.length;
    all.indeterminate = checked > 0 && checked < boxes.length;
  }

  function bind(target) {
    if (!target || host) return;
    host = target;
    const onClick = (event) => {
      const detailRow = event.target.closest('[data-act="' + ns + '-detail"]');
      if (detailRow && typeof instance.onDetail === 'function') {
        const key = detailRow.getAttribute('data-key');
        const row = lastRows.filter((item) => rowKeyOf(item) === key)[0];
        if (row) instance.onDetail(row);
        return;
      }
      const btn = event.target.closest('[data-dt-page]');
      if (!btn || btn.disabled) return;
      go(btn.getAttribute('data-dt-page'));
      if (cfg.onPaging) cfg.onPaging(state());
    };
    const onChange = (event) => {
      const pick = event.target.closest('[data-dt-pick]');
      if (pick) {
        const key = pick.getAttribute('data-dt-pick');
        if (pick.checked) picked[key] = true;
        else delete picked[key];
        syncPickAll();
        notifySelect();
        return;
      }
      const pickAll = event.target.closest('[data-dt-pick-all]');
      if (pickAll) {
        host.querySelectorAll('[data-dt-pick]').forEach((box) => {
          box.checked = pickAll.checked;
          const key = box.getAttribute('data-dt-pick');
          if (pickAll.checked) picked[key] = true;
          else delete picked[key];
        });
        notifySelect();
        return;
      }
      const sizeSel = event.target.closest('[data-dt-size]');
      if (sizeSel) {
        setPageSize(sizeSel.value);
        if (cfg.onPaging) cfg.onPaging(state());
        return;
      }
      const input = event.target.closest('[data-dt-page-input]');
      if (input) {
        go(input.value);
        if (cfg.onPaging) cfg.onPaging(state());
      }
    };
    const onKeydown = (event) => {
      const input = event.target.closest('[data-dt-page-input]');
      if (input && event.key === 'Enter') {
        go(input.value);
        if (cfg.onPaging) cfg.onPaging(state());
      }
    };
    host.addEventListener('click', onClick);
    host.addEventListener('change', onChange);
    host.addEventListener('keydown', onKeydown);
    disposers.push(() => host.removeEventListener('click', onClick));
    disposers.push(() => host.removeEventListener('change', onChange));
    disposers.push(() => host.removeEventListener('keydown', onKeydown));
    return instance;
  }

  function mount(target) { bind(target); return instance; }
  function render(rows, options) {
    if (host) host.innerHTML = html(rows, options);
  }

  function defaultDetail(row) {
    const d = cfg.detail;
    detailModal.show({
      title: typeof d.title === 'function' ? d.title(row) : (d.title || ns),
      fields: d.fields,
      row,
      width: d.width,
      actions: d.actions,
    });
  }

  const instance = {
    html, bind, mount, render, update: render, state, go, setPageSize, reset, selection, clearSelection,
    get dataHost() { return host; },
    destroy() {
      disposers.splice(0).forEach((fn) => fn());
      host = null;
    },
    onDetail: cfg.onDetail || (cfg.detail ? defaultDetail : null),
  };
  return instance;
}

export const dataTable = { create: createDataTable };
