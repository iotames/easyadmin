/** 审计日志：最近记录、结果过滤与客户端检索。 */
import { esc, bindActions } from '../../../core/dom.js';
import { fmtDateTime } from '../../../core/format.js';
import { createDataTable } from '../../../components/data-table.js';
import { badgeHTML } from '../../../components/badge.js';
import { errText } from '../../_shared.js';

let cleanup = null;

const OUTCOMES = {
  success: { text: '成功', tone: 'green' },
  failure: { text: '失败', tone: 'red' },
  denied: { text: '拒绝', tone: 'orange' },
};

function outcomeBadge(value) {
  const spec = OUTCOMES[value] || { text: value || '—', tone: 'gray' };
  return badgeHTML(spec);
}

export async function mount(ctx) {
  ctx.setTitle('审计日志');
  ctx.setDescription('关键操作留痕，最多显示最近 300 条');
  ctx.el.innerHTML =
    '<div class="card">' +
    '<div class="form-inline" id="audit-toolbar">' +
    '<input class="input input-sm" data-audit="keyword" placeholder="操作人 / 动作 / 摘要" style="width:240px">' +
    '<select class="select select-sm" data-audit="outcome">' +
    '<option value="">全部结果</option><option value="success">成功</option>' +
    '<option value="failure">失败</option><option value="denied">拒绝</option></select>' +
    '<button class="btn btn-sm" data-act="audit-reload">刷新</button>' +
    '</div><div id="audit-table"></div></div>';

  let alive = true;
  let all = [];
  let off = () => {};

  const table = createDataTable({
    ns: 'audit',
    rowKey: (row) => row.id,
    empty: '暂无审计记录',
    columns: [
      { key: 'created_at', title: '时间', render: (row) => esc(fmtDateTime(row.created_at)), nowrap: true },
      { key: 'actor_name', title: '操作人' },
      { key: 'action', title: '动作', render: (row) => '<span class="mono">' + esc(row.action) + '</span>' },
      { key: 'outcome', title: '结果', render: (row) => outcomeBadge(row.outcome) },
      { key: 'target_type', title: '对象', render: (row) => esc((row.target_type || '—') + (row.target_id ? ' · ' + row.target_id : '')) },
      { key: 'summary', title: '摘要' },
      { key: 'ip', title: 'IP', nowrap: true },
    ],
  });
  table.bind(ctx.el.querySelector('#audit-table'));

  function applyFilter() {
    const keywordEl = ctx.el.querySelector('[data-audit="keyword"]');
    const outcomeEl = ctx.el.querySelector('[data-audit="outcome"]');
    const keyword = keywordEl ? keywordEl.value.trim().toLowerCase() : '';
    const outcome = outcomeEl ? outcomeEl.value : '';
    const rows = all.filter((row) => {
      if (outcome && row.outcome !== outcome) return false;
      if (!keyword) return true;
      const hay = [row.actor_name, row.action, row.summary, row.target_type, row.target_id].join(' ').toLowerCase();
      return hay.indexOf(keyword) >= 0;
    });
    table.render(rows);
  }

  async function load() {
    try {
      const data = await ctx.api.get('/api/system/audit', { query: { limit: 300 } });
      if (!alive) return;
      all = data.items || [];
      applyFilter();
    } catch (err) {
      if (!alive) return;
      ctx.el.querySelector('#audit-table').innerHTML = '<div class="empty">' + esc(errText(err)) + '</div>';
      ctx.notify.error(errText(err));
    }
  }

  const onInput = () => applyFilter();
  const keywordEl = ctx.el.querySelector('[data-audit="keyword"]');
  const outcomeEl = ctx.el.querySelector('[data-audit="outcome"]');
  if (keywordEl) keywordEl.addEventListener('input', onInput);
  if (outcomeEl) outcomeEl.addEventListener('change', onInput);

  off = bindActions(ctx.el, { 'audit-reload': () => load() });

  cleanup = () => {
    alive = false;
    off();
    if (keywordEl) keywordEl.removeEventListener('input', onInput);
    if (outcomeEl) outcomeEl.removeEventListener('change', onInput);
    table.destroy();
  };
  ctx.onCleanup(cleanup);
  await load();
}

export function unmount() {
  if (cleanup) {
    cleanup();
    cleanup = null;
  }
}

export default { mount, unmount };
