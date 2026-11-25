/** 统一表单控件：页面不得手写裸控件或内联宽度。 */
import { esc, attrsHTML } from '../core/dom.js';

export { attrsHTML };

function commonAttrs(opts) {
  let html = '';
  if (opts.id) html += ' id="' + esc(opts.id) + '"';
  html += attrsHTML(opts.attrs);
  return html;
}

export function input(options) {
  const opts = options || {};
  const cls = ['input'];
  if (opts.sm) cls.push('input-sm');
  if (opts.width) cls.push('form-w-' + opts.width);
  if (opts.mono) cls.push('mono');
  if (opts.cls) cls.push(opts.cls);
  return '<input type="' + esc(opts.type || 'text') + '" class="' + cls.join(' ').trim() + '"' +
    commonAttrs(opts) + (opts.disabled ? ' disabled' : '') +
    (opts.placeholder ? ' placeholder="' + esc(opts.placeholder) + '"' : '') +
    ' value="' + esc(opts.value == null ? '' : String(opts.value)) + '"' +
    ' autocomplete="' + esc(opts.autocomplete || 'off') + '" spellcheck="false">';
}

export function select(options) {
  const opts = options || {};
  const cls = ['select'];
  if (opts.sm) cls.push('select-sm');
  if (opts.width) cls.push('form-w-' + opts.width);
  if (opts.cls) cls.push(opts.cls);
  return '<select class="' + cls.join(' ').trim() + '"' + commonAttrs(opts) + (opts.disabled ? ' disabled' : '') + '>' +
    (opts.options || []).map((item) => {
      const value = String(item[0] == null ? '' : item[0]);
      return '<option value="' + esc(value) + '"' + (String(opts.selected) === value ? ' selected' : '') + '>' +
        esc(item[1]) + '</option>';
    }).join('') + '</select>';
}

export function checkbox(options) {
  const opts = options || {};
  const cls = opts.cls ? ' class="' + esc(opts.cls) + '"' : '';
  const box = '<input type="checkbox"' + cls + commonAttrs(opts) + (opts.checked ? ' checked' : '') +
    (opts.disabled ? ' disabled' : '') + '>';
  return opts.label ? '<label class="form-check">' + box + ' ' + esc(opts.label) + '</label>' : box;
}

export function radio(options) {
  const opts = options || {};
  const name = opts.name ? ' name="' + esc(opts.name) + '"' : '';
  const cls = opts.cls ? ' class="' + esc(opts.cls) + '"' : '';
  return (opts.options || []).map((item) => {
    const value = String(item[0] == null ? '' : item[0]);
    return '<label class="form-check"><input type="radio"' + name + ' value="' + esc(value) + '"' + cls +
      (String(opts.selected) === value ? ' checked' : '') + (opts.disabled ? ' disabled' : '') + '> ' +
      esc(item[1]) + '</label>';
  }).join('');
}

export function textarea(options) {
  const opts = options || {};
  const cls = ['input'];
  if (opts.mono) cls.push('mono');
  if (opts.cls) cls.push(opts.cls);
  return '<textarea class="' + cls.join(' ').trim() + '"' + commonAttrs(opts) +
    ' rows="' + (opts.rows == null ? 4 : opts.rows) + '"' +
    (opts.placeholder ? ' placeholder="' + esc(opts.placeholder) + '"' : '') +
    (opts.disabled ? ' disabled' : '') + '>' +
    esc(opts.value == null ? '' : String(opts.value)) + '</textarea>';
}

export function switchControl(options) {
  const opts = options || {};
  const cls = opts.cls ? ' class="' + esc(opts.cls) + '"' : '';
  return '<label class="el-switch"' + (opts.title ? ' title="' + esc(opts.title) + '"' : '') + '>' +
    '<input type="checkbox"' + cls + commonAttrs(opts) + (opts.checked ? ' checked' : '') +
    (opts.disabled ? ' disabled' : '') + '><span class="el-switch-core"></span></label>';
}

export function search(options) {
  const opts = options || {};
  return input({
    id: opts.id,
    value: opts.value,
    placeholder: opts.placeholder,
    attrs: opts.attrs,
    cls: 'input-search',
    width: opts.width || 'full',
  });
}

export function inline(controlsHTML) {
  return '<div class="form-inline">' + (controlsHTML || '') + '</div>';
}

export function checkList(itemsHTML) {
  return '<div class="check-list">' + (itemsHTML || '') + '</div>';
}

export function label(text, controlHTML) {
  return '<div class="form-row"><label class="form-label">' + esc(text) + '</label>' + (controlHTML || '') + '</div>';
}

export function hint(text) {
  return '<div class="form-hint">' + esc(text) + '</div>';
}

/**
 * 字段行：包装 data-field，便于把 422 的 details.fields 映射回控件。
 * @param {{label:string,field?:string,control:string,required?:boolean,hint?:string,error?:string}} options
 */
export function field(options) {
  const opts = options || {};
  const required = opts.required ? '<span class="form-required">*</span>' : '';
  return '<div class="form-row"' + (opts.field ? ' data-field="' + esc(opts.field) + '"' : '') + '>' +
    '<label class="form-label">' + required + esc(opts.label) + '</label>' +
    (opts.control || '') +
    (opts.error ? '<div class="form-error">' + esc(opts.error) + '</div>' : '<div class="form-error" hidden></div>') +
    (opts.hint ? hint(opts.hint) : '') + '</div>';
}

/** 读取 [name] 控件值：checkbox 返回布尔，多选按同名聚合为数组。 */
export function readForm(host) {
  const values = {};
  if (!host) return values;
  host.querySelectorAll('[name]').forEach((el) => {
    const name = el.getAttribute('name');
    if (el.type === 'checkbox') {
      if (el.name.endsWith('[]')) {
        const key = name.slice(0, -2);
        values[key] = values[key] || [];
        if (el.checked) values[key].push(el.value);
      } else {
        values[name] = el.checked;
      }
    } else if (el.type === 'radio') {
      if (el.checked) values[name] = el.value;
    } else {
      values[name] = el.value;
    }
  });
  return values;
}

/**
 * 把 422 的 error.details.fields 落到对应 data-field 容器。
 * 无法映射的字段返回给调用方进入表单级错误摘要。
 */
export function applyFieldErrors(host, fields) {
  const unmapped = [];
  if (!host) return Object.keys(fields || {});
  host.querySelectorAll('[data-field]').forEach((row) => {
    const box = row.querySelector('.form-error');
    if (!box) return;
    box.hidden = true;
    box.textContent = '';
  });
  Object.keys(fields || {}).forEach((path) => {
    const row = host.querySelector('[data-field="' + path.replace(/"/g, '\\"') + '"]');
    const messages = Array.isArray(fields[path]) ? fields[path] : [String(fields[path])];
    if (!row) { unmapped.push(path); return; }
    let box = row.querySelector('.form-error');
    if (!box) {
      box = document.createElement('div');
      box.className = 'form-error';
      row.appendChild(box);
    }
    box.hidden = false;
    box.textContent = messages.join('；');
  });
  return unmapped;
}

export const form = {
  input, select, checkbox, radio, textarea, field, readForm, applyFieldErrors,
  switch: switchControl, search, inline, checkList, label, hint,
};
