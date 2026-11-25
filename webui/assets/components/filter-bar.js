/**
 * 数据列表页筛选栏：字段声明 → HTML / 收集 / 重置 / 变更即查。
 * 不感知 API 与业务语义；查询动作仍由页面 onQuery 提供。
 */
import { esc, captureFocus, restoreFocus } from '../core/dom.js';
import { DR_PARTS } from './date-range.js';

function optionsHTML(list, selected) {
  return (list || []).map((item) => {
    const value = String(item[0] == null ? '' : item[0]);
    return '<option value="' + esc(value) + '"' + (String(selected) === value ? ' selected' : '') + '>' +
      esc(item[1]) + '</option>';
  }).join('');
}

export function createFilterBar(options) {
  const opts = options || {};
  const ns = opts.ns || 'fb';
  const live = !!opts.live;
  const onQuery = opts.onQuery || function noop() {};
  const fields = opts.fields || [];
  const state = {};
  let host = null;

  function applyDefaults(field) {
    if (field.type === 'dateRange') {
      DR_PARTS.forEach((part) => { state[field.key + part] = (field.default && field.default[part]) || ''; });
    } else {
      state[field.key] = field.default != null ? field.default : '';
    }
  }
  fields.forEach(applyDefaults);

  function bar() {
    return host ? host.querySelector('#' + ns + '-filterbar') : document.getElementById(ns + '-filterbar');
  }

  function optionsOf(field) {
    return (typeof field.options === 'function' ? field.options() : field.options) || [];
  }

  function pickLabel(list, value) {
    const want = String(value == null ? '' : value);
    for (let i = 0; i < list.length; i += 1) {
      if (String(list[i][0] == null ? '' : list[i][0]) === want) return String(list[i][1] == null ? '' : list[i][1]);
    }
    return '';
  }

  function matchOptions(list, query) {
    const terms = String(query || '').toLowerCase().trim().split(/\s+/).filter(Boolean);
    if (!terms.length) return list.slice();
    return list.filter((item) => {
      const hay = (String(item[1] == null ? '' : item[1]) + ' ' + String(item[0] == null ? '' : item[0])).toLowerCase();
      return terms.every((term) => hay.indexOf(term) >= 0);
    });
  }

  function widthStyle(field) {
    return field && field.width ? ' style="width:' + esc(field.width) + '"' : '';
  }

  function comboHTML(field) {
    const key = field.key;
    const selected = String(state[key] == null ? '' : state[key]);
    const label = pickLabel(optionsOf(field), selected) || selected;
    return '<div class="combo" data-combo="' + esc(key) + '"' + widthStyle(field) + '>' +
      '<input class="input input-sm combo-input" data-combo-input="' + esc(key) + '"' +
      ' placeholder="' + esc(field.placeholder || '输入关键字筛选') + '" value="' + esc(label) + '"' +
      ' autocomplete="off" spellcheck="false" role="combobox" aria-autocomplete="list" aria-expanded="false">' +
      '<input type="hidden" data-fb="' + esc(key) + '" value="' + esc(selected) + '">' +
      '<div class="combo-menu" data-combo-menu="' + esc(key) + '" role="listbox" hidden></div></div>';
  }

  function fieldHTML(field) {
    if (field.type === 'dateRange') {
      const value = (part) => esc(state[field.key + part] || '');
      return '<div class="tool-group"><span class="muted">开始</span>' +
        '<input type="date" class="input input-sm" data-fb="' + field.key + 'fromDate" value="' + value('fromDate') + '">' +
        '<input type="time" class="input input-sm" data-fb="' + field.key + 'fromTime" value="' + value('fromTime') + '"></div>' +
        '<div class="tool-group"><span class="muted">结束</span>' +
        '<input type="date" class="input input-sm" data-fb="' + field.key + 'toDate" value="' + value('toDate') + '">' +
        '<input type="time" class="input input-sm" data-fb="' + field.key + 'toTime" value="' + value('toTime') + '"></div>';
    }
    if (field.type === 'searchSelect') return comboHTML(field);
    if (field.type === 'select') {
      return '<select class="select select-sm" data-fb="' + esc(field.key) + '"' + widthStyle(field) + '>' +
        optionsHTML(optionsOf(field), state[field.key]) + '</select>';
    }
    if (field.type === 'check') {
      return '<label class="chk"><input type="checkbox" data-fb="' + esc(field.key) + '"' +
        (state[field.key] ? ' checked' : '') + '> ' + esc(field.label || '') + '</label>';
    }
    return '<input class="input input-sm" data-fb="' + esc(field.key) + '" placeholder="' + esc(field.placeholder || '') + '"' +
      (field.width ? ' style="width:' + esc(field.width) + '"' : '') +
      ' value="' + esc(state[field.key] || '') + '">';
  }

  function html() {
    return '<div class="form-inline" id="' + esc(ns) + '-filterbar">' + fields.map(fieldHTML).join('') + '</div>';
  }

  function collect() {
    const node = bar();
    if (!node) return state;
    node.querySelectorAll('[data-fb]').forEach((el) => {
      const key = el.getAttribute('data-fb');
      state[key] = el.type === 'checkbox' ? !!el.checked : (el.value || '');
    });
    return state;
  }

  function stateObj() {
    collect();
    return state;
  }

  function syncComboDisplay(key) {
    const node = bar();
    if (!node) return;
    const input = node.querySelector('[data-combo-input="' + key + '"]');
    const hidden = node.querySelector('[data-fb="' + key + '"]');
    const field = fields.filter((item) => item.key === key)[0];
    if (!input || !hidden || !field) return;
    input.value = pickLabel(optionsOf(field), hidden.value) || hidden.value || '';
  }

  function reset() {
    fields.forEach(applyDefaults);
    const node = bar();
    if (node) {
      node.querySelectorAll('[data-fb]').forEach((el) => {
        const key = el.getAttribute('data-fb');
        if (el.type === 'checkbox') el.checked = !!state[key];
        else el.value = state[key] || '';
      });
    }
    fields.forEach((field) => { if (field.type === 'searchSelect') syncComboDisplay(field.key); });
    return onQuery(state);
  }

  function set(values) {
    Object.keys(values || {}).forEach((key) => {
      if (!(key in state)) return;
      state[key] = values[key] == null ? '' : String(values[key]);
      const node = bar();
      if (!node) return;
      const el = node.querySelector('[data-fb="' + key + '"]');
      if (!el) return;
      if (el.type === 'checkbox') el.checked = !!state[key];
      else el.value = state[key];
    });
    fields.forEach((field) => { if (field.type === 'searchSelect') syncComboDisplay(field.key); });
    return state;
  }

  function trigger() {
    setTimeout(runQuery, 0);
  }

  function runQuery() {
    const node = bar();
    const snap = node ? captureFocus(node) : null;
    const result = onQuery(collect());
    const restore = () => {
      const next = bar();
      if (next) restoreFocus(next, snap);
    };
    if (result && typeof result.then === 'function') result.then(restore, restore);
    else setTimeout(restore, 0);
  }

  let comboActive = null;

  function comboMenu(key) {
    const node = bar();
    return node ? node.querySelector('[data-combo-menu="' + key + '"]') : null;
  }

  function comboInput(key) {
    const node = bar();
    return node ? node.querySelector('[data-combo-input="' + key + '"]') : null;
  }

  function closeCombo(key) {
    const menu = comboMenu(key);
    const input = comboInput(key);
    if (menu) menu.hidden = true;
    if (input) input.setAttribute('aria-expanded', 'false');
    if (comboActive === key) comboActive = null;
  }

  function renderComboMenu(key, query) {
    const field = fields.filter((item) => item.key === key)[0];
    const menu = comboMenu(key);
    const input = comboInput(key);
    if (!field || !menu || !input) return;
    const list = matchOptions(optionsOf(field), query);
    const limit = 200;
    let content;
    if (!list.length) {
      content = '<div class="combo-empty">没有匹配的选项</div>';
    } else {
      content = list.slice(0, limit).map((item) => {
        const value = String(item[0] == null ? '' : item[0]);
        const label = String(item[1] == null ? '' : item[1]);
        return '<div class="combo-option" role="option" data-combo-value="' + esc(value) +
          '" data-combo-label="' + esc(label) + '">' + esc(label) + '</div>';
      }).join('');
      if (list.length > limit) {
        content += '<div class="combo-empty">还有 ' + esc(String(list.length - limit)) + ' 项未显示，请继续输入缩小范围</div>';
      }
    }
    menu.innerHTML = content;
    menu.hidden = false;
    input.setAttribute('aria-expanded', 'true');
    comboActive = key;
  }

  function chooseComboOption(optionEl) {
    const key = comboActive;
    const input = comboInput(key);
    const node = bar();
    const hidden = node ? node.querySelector('[data-fb="' + key + '"]') : null;
    if (!key || !input || !hidden) return;
    hidden.value = optionEl.getAttribute('data-combo-value') || '';
    state[key] = hidden.value;
    input.value = optionEl.getAttribute('data-combo-label') || '';
    closeCombo(key);
    trigger();
  }

  function triggerIfRangeComplete() {
    const node = bar();
    if (!node) return;
    let complete = true;
    node.querySelectorAll('[data-fb]').forEach((el) => {
      if (el.type !== 'date' && el.type !== 'time') return;
      if (!el.value) complete = false;
    });
    if (complete) runQuery();
    else collect();
  }

  function bind(target) {
    if (!target || host) return instance;
    host = target;
    const node = bar();
    if (!node || !live) return instance;
    const onField = typeof opts.onField === 'function' ? opts.onField : null;

    node.addEventListener('change', (event) => {
      const el = event.target.closest('[data-fb]');
      if (!el) return;
      if (onField && onField(el) === 'handled') return;
      if (el.type === 'date' || el.type === 'time') triggerIfRangeComplete();
      else trigger();
    });
    node.addEventListener('input', (event) => {
      const combo = event.target.closest('[data-combo-input]');
      if (!combo) return;
      const key = combo.getAttribute('data-combo-input');
      const hidden = node.querySelector('[data-fb="' + key + '"]');
      if (hidden && String(combo.value).trim() === '' && hidden.value !== '') {
        hidden.value = '';
        state[key] = '';
      }
      renderComboMenu(key, combo.value);
    });
    node.addEventListener('focusin', (event) => {
      const combo = event.target.closest('[data-combo-input]');
      if (combo) renderComboMenu(combo.getAttribute('data-combo-input'), '');
    });
    node.addEventListener('click', (event) => {
      const optionEl = event.target.closest('[data-combo-value]');
      if (optionEl) {
        event.preventDefault();
        chooseComboOption(optionEl);
        return;
      }
      const combo = event.target.closest('[data-combo-input]');
      if (combo) renderComboMenu(combo.getAttribute('data-combo-input'), '');
    });
    node.addEventListener('mousedown', (event) => {
      if (event.target.closest('[data-combo-value]')) event.preventDefault();
    });
    node.addEventListener('keydown', (event) => {
      const combo = event.target.closest('[data-combo-input]');
      if (!combo) return;
      const key = combo.getAttribute('data-combo-input');
      const menu = comboMenu(key);
      if (!menu || menu.hidden) {
        if (event.key === 'ArrowDown') {
          renderComboMenu(key, combo.value);
          event.preventDefault();
        }
        return;
      }
      const list = Array.prototype.slice.call(menu.querySelectorAll('[data-combo-value]'));
      let active = list.findIndex((el) => el.classList.contains('is-active'));
      if (event.key === 'ArrowDown') {
        event.preventDefault();
        active = Math.min(list.length - 1, active + 1);
      } else if (event.key === 'ArrowUp') {
        event.preventDefault();
        active = Math.max(0, active - 1);
      } else if (event.key === 'Enter') {
        if (active >= 0 && list[active]) chooseComboOption(list[active]);
        event.preventDefault();
        return;
      } else if (event.key === 'Escape') {
        closeCombo(key);
        return;
      } else {
        return;
      }
      list.forEach((el, index) => el.classList.toggle('is-active', index === active));
      if (list[active]) list[active].scrollIntoView({ block: 'nearest' });
    });
    node.addEventListener('focusout', (event) => {
      const combo = event.target.closest('[data-combo-input]');
      if (!combo) return;
      const key = combo.getAttribute('data-combo-input');
      setTimeout(() => {
        const box = node.querySelector('[data-combo="' + key + '"]');
        if (!box || !box.contains(document.activeElement)) closeCombo(key);
      }, 150);
    });
    return instance;
  }

  function destroy() {
    host = null;
  }

  const instance = { html, bind, collect, state: stateObj, reset, set, destroy };
  return instance;
}

export const filterBar = { create: createFilterBar };
