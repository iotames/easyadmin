/** 页内标签。items: [{name,label,count}]。 */
import { esc } from '../core/dom.js';

export function tabsHTML(items, active, options) {
  const opts = options || {};
  const act = opts.act || 'tab';
  const attr = opts.nameAttr || 'data-name';
  return '<div class="tabs">' + (items || []).map((item) =>
    '<div class="tab' + (item.name === active ? ' active' : '') + '"' +
    ' data-act="' + esc(act) + '" ' + attr + '="' + esc(item.name) + '">' +
    esc(item.label) + (item.count ? '<span class="tab-count">' + esc(item.count) + '</span>' : '') +
    '</div>').join('') + '</div>';
}
