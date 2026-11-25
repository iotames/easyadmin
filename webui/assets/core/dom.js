/** HTML 转义：所有不可信文本进入字符串模板前必须经过本函数。 */
const ESC_MAP = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' };

export function esc(value) {
  return String(value == null ? '' : value).replace(/[&<>"']/g, (c) => ESC_MAP[c]);
}

/** 查询单个元素。 */
export function $(selector, root) {
  return (root || document).querySelector(selector);
}

/** 查询多个元素并返回数组。 */
export function $$(selector, root) {
  return Array.prototype.slice.call((root || document).querySelectorAll(selector));
}

/** 组装 HTML 属性：undefined/null/false 跳过，true 输出裸属性。 */
export function attrsHTML(attrs) {
  if (!attrs) return '';
  return Object.keys(attrs).map((key) => {
    const value = attrs[key];
    if (value === undefined || value === null || value === false) return '';
    if (value === true) return ' ' + key;
    return ' ' + key + '="' + esc(String(value)) + '"';
  }).join('');
}

/** 防抖。 */
export function debounce(fn, wait) {
  let timer = null;
  return function debounced(...args) {
    clearTimeout(timer);
    timer = setTimeout(() => fn.apply(this, args), wait);
  };
}

/** 拼装查询串：空值跳过，避免服务端把空串当过滤条件。 */
export function queryString(params) {
  const usp = new URLSearchParams();
  Object.keys(params || {}).forEach((key) => {
    const value = params[key];
    if (value === undefined || value === null || value === '') return;
    usp.set(key, String(value));
  });
  const text = usp.toString();
  return text ? '?' + text : '';
}

/** 在光标处插入文本并触发 input。 */
export function insertAtCursor(input, text) {
  const start = input.selectionStart;
  const end = input.selectionEnd;
  const value = input.value;
  input.value = value.slice(0, start) + text + value.slice(end);
  input.selectionStart = input.selectionEnd = start + text.length;
  input.dispatchEvent(new Event('input'));
  input.focus();
}

function focusSelector(el) {
  if (el.id) return '#' + (window.CSS && CSS.escape ? CSS.escape(el.id) : el.id);
  const keys = ['data-field', 'data-fb', 'data-combo-input', 'data-cfg-input'];
  for (let i = 0; i < keys.length; i += 1) {
    const value = el.getAttribute(keys[i]);
    if (value === null) continue;
    return '[' + keys[i] + '="' + String(value).replace(/\\/g, '\\\\').replace(/"/g, '\\"') + '"]';
  }
  return '';
}

/** 重绘前抓取焦点位置；整块 innerHTML 重绘后用于恢复。 */
export function captureFocus(root) {
  const el = document.activeElement;
  if (!root || !el || el === document.body || !root.contains(el)) return null;
  const selector = focusSelector(el);
  if (!selector) return null;
  return {
    selector,
    value: el.value === undefined || el.value === null ? null : String(el.value),
    start: typeof el.selectionStart === 'number' ? el.selectionStart : null,
    end: typeof el.selectionEnd === 'number' ? el.selectionEnd : null,
  };
}

/** 重绘后恢复焦点与光标；值已被用户改动时把光标放到末尾。 */
export function restoreFocus(root, snap) {
  if (!root || !snap) return;
  const el = root.querySelector(snap.selector);
  if (!el) return;
  el.focus();
  if (snap.start === null || typeof el.setSelectionRange !== 'function') return;
  const value = String(el.value === undefined || el.value === null ? '' : el.value);
  const changed = snap.value !== value;
  const start = changed ? value.length : Math.min(snap.start, value.length);
  const end = changed ? value.length : Math.min(snap.end, value.length);
  try { el.setSelectionRange(start, end); } catch (e) { /* 元素已不可编辑 */ }
}

/** 事件委托：root 内 [data-act] 点击分派到 handlers。返回 dispose。 */
export function bindActions(root, handlers) {
  if (!root) return () => {};
  const listener = (event) => {
    const el = event.target.closest('[data-act]');
    if (!el || !root.contains(el)) return;
    const handler = handlers && handlers[el.getAttribute('data-act')];
    if (typeof handler === 'function') handler(el, event);
  };
  root.addEventListener('click', listener);
  return () => root.removeEventListener('click', listener);
}

/** 安全 DOM 构造：默认 textContent，禁止直接塞入不可信 HTML。 */
export function h(tag, props = {}, ...children) {
  const node = document.createElement(tag);
  Object.keys(props || {}).forEach((key) => {
    const value = props[key];
    if (value === undefined || value === null || value === false) return;
    if (key === 'class') node.className = String(value);
    else if (key === 'text') node.textContent = String(value);
    else if (key === 'dataset') Object.assign(node.dataset, value);
    else if (key === 'attrs') {
      Object.keys(value).forEach((name) => node.setAttribute(name, String(value[name])));
    } else if (key.startsWith('on') && typeof value === 'function') {
      node.addEventListener(key.slice(2).toLowerCase(), value);
    } else {
      node.setAttribute(key, value === true ? '' : String(value));
    }
  });
  children.flat().forEach((child) => {
    if (child === undefined || child === null || child === false) return;
    node.appendChild(child instanceof Node ? child : document.createTextNode(String(child)));
  });
  return node;
}

/** 清空节点。 */
export function clear(node) {
  if (node) node.textContent = '';
}
