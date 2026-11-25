/** 空态与加载失败卡片。 */
import { esc } from '../core/dom.js';

export function emptyHTML(options) {
  const opts = options || {};
  if (opts.title != null || opts.description != null) {
    return '<div class="empty">' +
      (opts.title != null ? '<div class="empty-title">' + esc(opts.title) + '</div>' : '') +
      (opts.description != null ? '<div class="empty-desc">' + esc(opts.description) + '</div>' : '') +
      (opts.action || '') + '</div>';
  }
  return '<div class="empty">' + esc(opts.text) + (opts.br ? '<br>' : '') + (opts.action || '') + '</div>';
}

export function emptyCardHTML(options) {
  return '<div class="card">' + emptyHTML(options) + '</div>';
}
