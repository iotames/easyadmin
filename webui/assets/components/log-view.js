/** 任务日志视图：按级别过滤，正文与 attrs 一律转义。 */
import { esc } from '../core/dom.js';
import { fmtDateTime } from '../core/format.js';

export const LOG_LEVELS = [['', '全部级别'], ['debug', 'debug'], ['info', 'info'], ['warn', 'warn'], ['error', 'error']];

export function logLineHTML(entry) {
  const attrs = entry && entry.attrs && Object.keys(entry.attrs).length
    ? ' ' + esc(JSON.stringify(entry.attrs)) : '';
  const level = esc(entry.level || 'info');
  return '<span class="seq">#' + esc(entry.seq) + '</span> ' +
    '<span class="' + level + '">[' + level + ']</span> ' +
    esc(fmtDateTime(entry.logged_at)) + ' ' + esc(entry.message) + attrs;
}

export function logViewHTML(entries, level) {
  const list = (entries || []).filter((entry) => !level || entry.level === level);
  if (!list.length) return '<span class="muted">（没有日志）</span>';
  return list.map(logLineHTML).join('\n');
}

export const logView = { LEVELS: LOG_LEVELS, html: logViewHTML, lineHTML: logLineHTML };
