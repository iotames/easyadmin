/** 配置中心：只读展示启动配置元数据；敏感项只显示是否已配置。 */
import { esc, bindActions } from '../../../core/dom.js';
import { badgeHTML } from '../../../components/badge.js';
import { errText } from '../../_shared.js';

let cleanup = null;

const SOURCE_LABELS = {
  default: '默认值',
  file: '配置文件',
  environment: '环境变量',
  command_line: '命令行',
};

const HOT_LABELS = {
  none: '不热更',
  hot: '热更',
  restart: '需重启',
};

function valueHTML(item) {
  if (item.secret) {
    return item.configured
      ? badgeHTML({ text: '已配置', tone: 'green' })
      : badgeHTML({ text: '未配置', tone: 'gray' });
  }
  return '<span class="cfg-current">' + esc(item.value === '' ? '—' : item.value) + '</span>';
}

function rowHTML(item) {
  return '<div class="cfg-row">' +
    '<div><div class="cfg-title">' + esc(item.title || item.key) +
    (item.secret ? ' ' + badgeHTML({ text: '敏感', tone: 'orange' }) : '') + '</div>' +
    '<div class="cfg-key">' + esc(item.key) + '</div>' +
    (item.example ? '<div class="cfg-usage">' + esc(item.example) + '</div>' : '') + '</div>' +
    '<div>' + valueHTML(item) +
    '<div class="cfg-usage">默认：' + esc(item.default === '' ? '—' : item.default) + '</div></div>' +
    '<div class="cfg-actions">' +
    badgeHTML({ text: SOURCE_LABELS[item.source] || item.source || '—', tone: 'blue' }) +
    badgeHTML({ text: HOT_LABELS[item.hot_reload] || item.hot_reload || '—', tone: item.hot_reload === 'hot' ? 'green' : 'gray' }) +
    badgeHTML({ text: item.writable ? '可写' : '只读', tone: 'gray' }) +
    '</div></div>';
}

export async function mount(ctx) {
  ctx.setTitle('配置中心');
  ctx.setDescription('启动配置只读视图；敏感项不显示原值');
  ctx.el.innerHTML = '<div class="card">' +
    '<div class="form-inline cfg-filter">' +
    '<input class="input input-sm" data-cfg="keyword" placeholder="搜索键名或说明" style="width:260px">' +
    '<button class="btn btn-sm" data-act="cfg-reload">刷新</button>' +
    '</div><div id="cfg-groups"><div class="empty">加载中…</div></div></div>';

  let alive = true;
  let items = [];
  let off = () => {};

  function render() {
    const keywordEl = ctx.el.querySelector('[data-cfg="keyword"]');
    const keyword = keywordEl ? keywordEl.value.trim().toLowerCase() : '';
    const groups = {};
    const order = [];
    items.forEach((item) => {
      if (keyword && (item.key + ' ' + (item.title || '')).toLowerCase().indexOf(keyword) < 0) return;
      const name = item.group || 'general';
      if (!groups[name]) {
        groups[name] = [];
        order.push(name);
      }
      groups[name].push(item);
    });
    const host = ctx.el.querySelector('#cfg-groups');
    if (!order.length) {
      host.innerHTML = '<div class="empty">没有匹配的配置项</div>';
      return;
    }
    host.innerHTML = order.map((name) =>
      '<div class="card" style="margin-bottom:16px"><div class="card-title">' + esc(name) +
      '<span class="card-sub">' + groups[name].length + ' 项</span></div>' +
      groups[name].map(rowHTML).join('') + '</div>').join('');
  }

  async function load() {
    try {
      const data = await ctx.api.get('/api/system/config');
      if (!alive) return;
      items = data.items || [];
      render();
    } catch (err) {
      if (!alive) return;
      ctx.el.querySelector('#cfg-groups').innerHTML = '<div class="empty">' + esc(errText(err)) + '</div>';
      ctx.notify.error(errText(err));
    }
  }

  const keywordEl = ctx.el.querySelector('[data-cfg="keyword"]');
  const onInput = () => render();
  if (keywordEl) keywordEl.addEventListener('input', onInput);
  off = bindActions(ctx.el, { 'cfg-reload': () => load() });

  cleanup = () => {
    alive = false;
    off();
    if (keywordEl) keywordEl.removeEventListener('input', onInput);
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
