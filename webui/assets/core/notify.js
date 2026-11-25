/** 全站唯一提示入口。error/warning 常驻，success/info 自动消失。 */
const ICONS = { success: '✓', error: '✕', warning: '⚠', info: 'ℹ' };

function toastRoot() {
  let root = document.getElementById('toast-root');
  if (!root) {
    root = document.createElement('div');
    root.id = 'toast-root';
    document.body.appendChild(root);
  }
  return root;
}

export function toastRaw(message, type, duration) {
  const kind = type || 'success';
  const sticky = kind === 'error' || kind === 'warning';
  const root = toastRoot();
  const el = document.createElement('div');
  el.className = 'toast toast-' + kind + (sticky ? ' toast-sticky' : '');
  el.innerHTML =
    '<div class="toast-head"><span class="toast-icon">' + (ICONS[kind] || ICONS.info) + '</span>' +
    '<span class="toast-msg"></span>' +
    (sticky ? '<button class="toast-x" data-toast-act="close" title="关闭">✕</button>' : '') +
    '</div>' +
    (sticky ? '<div class="toast-foot"><button class="btn btn-sm" data-toast-act="close">知道了</button></div>' : '');
  el.querySelector('.toast-msg').textContent = message;
  root.appendChild(el);
  requestAnimationFrame(() => el.classList.add('show'));

  let closed = false;
  const close = () => {
    if (closed || !el.isConnected) return;
    closed = true;
    el.classList.remove('show');
    setTimeout(() => el.remove(), 260);
  };
  el.addEventListener('click', (event) => {
    if (!sticky || event.target.closest('[data-toast-act="close"]')) close();
  });
  if (!sticky) setTimeout(close, duration == null ? 3200 : duration);
  return close;
}

export const notify = {
  success(message) { return toastRaw(message, 'success'); },
  info(message, duration) { return toastRaw(message, 'info', duration); },
  warn(message) { return toastRaw(message, 'warning'); },
  error(message) { return toastRaw(message, 'error'); },
};

export function clearToasts() {
  const root = document.getElementById('toast-root');
  if (root) root.textContent = '';
}
