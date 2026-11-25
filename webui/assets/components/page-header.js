/** 页头：标题、描述、页面级操作；刷新按钮全站唯一。 */
import { esc } from '../core/dom.js';

export const REFRESH_BUTTON = '<button class="btn btn-sm" data-act="page-refresh">⟳ 刷新</button>';

export function refreshButtonHTML() {
  return REFRESH_BUTTON;
}

/**
 * @param {{title?:string,titleHTML?:string,description?:string,desc?:string,actions?:string,refresh?:boolean}} options
 */
export function pageHeaderHTML(options) {
  const opts = options || {};
  const actions = (opts.actions || '') + (opts.refresh === false ? '' : REFRESH_BUTTON);
  const desc = opts.description != null ? opts.description : opts.desc;
  return '<div class="page-head">' +
    '<div><div class="page-title">' + (opts.titleHTML != null ? opts.titleHTML : esc(opts.title)) + '</div>' +
    '<div class="page-desc">' + esc(desc) + '</div></div>' +
    (actions ? '<div class="page-head-actions">' + actions + '</div>' : '') +
    '</div>';
}
