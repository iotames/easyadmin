/** 行详情弹层：字段清单 → 键值网格；复制反馈由组件本地完成，不弹全局 toast。 */
import { esc } from '../core/dom.js';
import { openModal } from './modal.js';

function fieldHTML(field, row) {
  const raw = row[field.key];
  const copy = field.copy && raw != null && raw !== ''
    ? '<button class="btn btn-sm btn-text detail-copy" data-copy-value="' + esc(String(raw)) + '" title="复制">复制</button>'
    : '';
  if (typeof field.render === 'function') {
    return '<div class="detail-item detail-item-block"><span class="k">' + esc(field.label) + '：</span>' +
      '<span class="v">' + field.render(row) + '</span>' + copy + '</div>';
  }
  const text = raw == null || raw === '' ? '—' : String(raw);
  if (field.pre) {
    return '<div class="detail-item detail-item-block"><span class="k">' + esc(field.label) + '：</span>' +
      copy + '<span class="detail-pre">' + esc(text) + '</span></div>';
  }
  return '<div class="detail-item"><span class="k">' + esc(field.label) + '：</span>' +
    '<span class="v">' + esc(text) + '</span>' + copy + '</div>';
}

export function showDetail(options) {
  const opts = options || {};
  if (!opts.fields || !opts.fields.length) return null;
  const row = opts.row || {};
  const body = '<div class="detail-grid detail-grid-modal">' +
    opts.fields.map((field) => fieldHTML(field, row)).join('') + '</div>';
  let footer = '';
  if (opts.actions && opts.actions.length) {
    footer = opts.actions.map((action, index) =>
      '<button class="btn ' + (action.className || 'btn-primary') + '" data-detail-act="' + index + '">' +
      esc(action.label || '') + '</button>').join('');
  }
  const overlay = openModal({
    title: (typeof opts.title === 'function' ? opts.title(row) : opts.title) || '详情',
    body,
    footer,
    width: opts.width,
  });
  overlay.addEventListener('click', (event) => {
    const copyBtn = event.target.closest('[data-copy-value]');
    if (copyBtn) {
      const text = copyBtn.getAttribute('data-copy-value');
      const done = (ok) => {
        copyBtn.textContent = ok ? '已复制' : '复制失败';
        setTimeout(() => { if (copyBtn.isConnected) copyBtn.textContent = '复制'; }, 1500);
        const callback = ok ? opts.onCopied : opts.onCopyError;
        if (typeof callback === 'function') callback(text);
      };
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(() => done(true), () => done(false));
      } else {
        done(false);
      }
      return;
    }
    const actBtn = event.target.closest('[data-detail-act]');
    if (actBtn) {
      const action = opts.actions[Number(actBtn.getAttribute('data-detail-act'))];
      if (action && typeof action.onClick === 'function') action.onClick(row, overlay);
    }
  });
  return overlay;
}

export const detailModal = { show: showDetail };
