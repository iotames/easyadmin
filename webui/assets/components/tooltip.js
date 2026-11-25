/** 全局 tooltip：[data-tip] 跟随鼠标，视口内自动翻转。 */
let tipEl = null;
let bound = false;

function hide() {
  if (tipEl) tipEl.style.display = 'none';
}

function show(el) {
  const text = el.getAttribute('data-tip');
  if (!text) { hide(); return; }
  if (!tipEl) {
    tipEl = document.createElement('div');
    tipEl.className = 'tip-popup';
    document.body.appendChild(tipEl);
  }
  tipEl.textContent = text;
  tipEl.style.display = 'block';
  const rect = el.getBoundingClientRect();
  const pad = 12;
  let x = rect.left + rect.width / 2 - tipEl.offsetWidth / 2;
  x = Math.min(Math.max(x, pad), window.innerWidth - tipEl.offsetWidth - pad);
  let y = rect.top - tipEl.offsetHeight - 10;
  if (y < pad) y = rect.bottom + 10;
  if (y + tipEl.offsetHeight > window.innerHeight - pad) {
    y = Math.max(pad, window.innerHeight - tipEl.offsetHeight - pad);
  }
  tipEl.style.left = x + 'px';
  tipEl.style.top = y + 'px';
}

const onMouseOver = (event) => {
  const el = event.target.closest('[data-tip]');
  if (el) show(el); else hide();
};
const onMouseOut = (event) => { if (!event.target.closest('[data-tip]')) hide(); };
const onScroll = () => hide();

export function initTooltip() {
  if (bound || typeof document === 'undefined') return;
  bound = true;
  document.addEventListener('mouseover', onMouseOver);
  document.addEventListener('mouseout', onMouseOut);
  document.addEventListener('scroll', onScroll, true);
  window.addEventListener('resize', hide);
}

export function destroyTooltip() {
  if (!bound) return;
  bound = false;
  document.removeEventListener('mouseover', onMouseOver);
  document.removeEventListener('mouseout', onMouseOut);
  document.removeEventListener('scroll', onScroll, true);
  if (tipEl) { tipEl.remove(); tipEl = null; }
}
