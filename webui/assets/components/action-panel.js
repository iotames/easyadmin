/** 维护类动作卡：标题 / 说明 / 一排主色按钮 / 底部提示。 */
import { esc } from '../core/dom.js';

export function actionPanelHTML(options) {
  const opts = options || {};
  const title = opts.title != null ? opts.title : '维护';
  const buttons = (opts.buttons || []).map((button) =>
    '<button class="btn btn-sm btn-primary" data-act="' + esc(button.act) + '"' +
    (button.disabled ? ' disabled' : '') +
    (button.title ? ' title="' + esc(button.title) + '"' : '') + '>' + esc(button.label) + '</button>').join('');
  return '<div class="card"><div class="card-title">' + esc(title) +
    (opts.description || opts.desc ? '<span class="card-sub">' + esc(opts.description || opts.desc) + '</span>' : '') +
    '</div>' + (buttons ? '<div class="form-inline">' + buttons + '</div>' : '') +
    (opts.hint ? '<div class="form-hint">' + esc(opts.hint) + '</div>' : '') + '</div>';
}

export const actionPanel = { html: actionPanelHTML };
