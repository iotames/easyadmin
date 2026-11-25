/** 弹窗与二次确认。所有调用点共用这一份遮罩/ESC/焦点行为。 */
import { esc } from '../core/dom.js';

let escBound = false;

function modalRoot() {
  let root = document.getElementById('modal-root');
  if (!root) {
    root = document.createElement('div');
    root.id = 'modal-root';
    document.body.appendChild(root);
  }
  return root;
}

function bindEscapeOnce() {
  if (escBound) return;
  escBound = true;
  document.addEventListener('keydown', (event) => {
    if (event.key !== 'Escape') return;
    const root = document.getElementById('modal-root');
    if (!root || !root.lastElementChild) return;
    event.preventDefault();
    const overlay = root.lastElementChild;
    if (typeof overlay.__close === 'function') overlay.__close();
    else overlay.remove();
  });
}

/**
 * 打开弹窗。body/footer 为调用方负责转义的 HTML。
 * @returns {HTMLElement} overlay，提供 close()
 */
export function openModal(options) {
  const opts = options || {};
  bindEscapeOnce();
  const overlay = document.createElement('div');
  overlay.className = 'modal-overlay';
  overlay.innerHTML =
    '<div class="modal" style="width:' + Number(opts.width || 800) + 'px">' +
    '<div class="modal-header"><span class="modal-title">' + esc(opts.title || '') + '</span>' +
    '<button class="modal-x" data-modal-act="close" aria-label="关闭">✕</button></div>' +
    '<div class="modal-body">' + (opts.body || '') + '</div>' +
    (opts.footer ? '<div class="modal-footer">' + opts.footer + '</div>' : '') +
    '</div>';
  modalRoot().appendChild(overlay);

  let downOnOverlay = false;
  overlay.addEventListener('mousedown', (event) => { downOnOverlay = event.target === overlay; });
  overlay.addEventListener('click', (event) => {
    const action = event.target.closest('[data-modal-act]');
    if (action) {
      event.stopPropagation();
      if (action.getAttribute('data-modal-act') === 'cancel' ||
          action.getAttribute('data-modal-act') === 'close') overlay.__close();
      return;
    }
    if (event.target === overlay && downOnOverlay) overlay.__close();
  });

  overlay.__close = () => {
    overlay.remove();
    if (typeof opts.onClose === 'function') opts.onClose();
  };
  return overlay;
}

/** 二次确认。resolve(true/false)。 */
export function confirmDialog(options) {
  const opts = options || {};
  return new Promise((resolve) => {
    let settled = false;
    const done = (value) => {
      if (settled) return;
      settled = true;
      resolve(value);
    };
    const overlay = openModal({
      title: opts.title || '操作确认',
      width: opts.width || 440,
      body: opts.message || '',
      footer:
        '<button class="btn" data-modal-act="cancel">' + esc(opts.cancelText || '取消') + '</button>' +
        '<button class="btn ' + (opts.danger ? 'btn-danger' : 'btn-primary') + '" data-modal-act="ok">' +
        esc(opts.confirmText || '确认') + '</button>',
      onClose: () => done(false),
    });
    overlay.addEventListener('click', (event) => {
      const action = event.target.closest('[data-modal-act]');
      if (!action) return;
      const kind = action.getAttribute('data-modal-act');
      if (kind === 'ok') {
        settled = true;
        overlay.remove();
        resolve(true);
      } else if (kind === 'cancel') {
        done(false);
      }
    });
  });
}
