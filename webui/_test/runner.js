/** 零依赖浏览器测试执行器：任一用例失败即为红。 */
const tests = [];
let suiteName = '未分组';

window.__TEST_RESULT__ = { status: 'pending', total: 0, passed: 0, failed: 0, cases: [] };

export function suite(name, fn) {
  const prev = suiteName;
  suiteName = name;
  try { fn(); } finally { suiteName = prev; }
}

export function test(name, fn) {
  tests.push({ suite: suiteName, name, fn });
}

function fail(message) { throw new Error(message || '断言失败'); }

export function ok(value, message) {
  if (!value) fail(message || '期望为真，实际为假');
}

export function eq(actual, expected, message) {
  if (actual !== expected) {
    fail((message ? message + '：' : '') + '期望 ' + JSON.stringify(expected) + '，实际 ' + JSON.stringify(actual));
  }
}

export function deepEq(actual, expected, message) {
  const a = JSON.stringify(actual);
  const b = JSON.stringify(expected);
  if (a !== b) fail((message ? message + '：' : '') + '期望 ' + b + '，实际 ' + a);
}

export function contains(haystack, needle, message) {
  if (String(haystack == null ? '' : haystack).indexOf(needle) < 0) {
    fail((message ? message + '：' : '') + '未包含 ' + JSON.stringify(needle));
  }
}

export function notContains(haystack, needle, message) {
  if (String(haystack == null ? '' : haystack).indexOf(needle) >= 0) {
    fail((message ? message + '：' : '') + '不应包含 ' + JSON.stringify(needle));
  }
}

export async function throws(fn, message) {
  try {
    await fn();
  } catch (err) {
    return err;
  }
  fail(message || '期望抛出异常，实际正常返回');
}

function el(tag, cls, text) {
  const node = document.createElement(tag);
  if (cls) node.className = cls;
  if (text != null) node.textContent = text;
  return node;
}

export async function run() {
  const started = Date.now();
  document.documentElement.setAttribute('data-test-status', 'running');
  const host = document.getElementById('results');

  for (let i = 0; i < tests.length; i += 1) {
    const item = tests[i];
    const caseResult = { suite: item.suite, name: item.name, ok: true, message: '' };
    try {
      await item.fn();
    } catch (err) {
      caseResult.ok = false;
      caseResult.message = err && err.message ? err.message : String(err);
    }
    window.__TEST_RESULT__.cases.push(caseResult);
    window.__TEST_RESULT__.total += 1;
    if (caseResult.ok) window.__TEST_RESULT__.passed += 1;
    else window.__TEST_RESULT__.failed += 1;
  }

  const result = window.__TEST_RESULT__;
  result.status = result.failed ? 'fail' : 'pass';
  result.durationMs = Date.now() - started;
  document.documentElement.setAttribute('data-test-status', 'done');
  document.documentElement.setAttribute('data-test-result', result.status);

  if (host) {
    host.textContent = '';
    const summary = el('div', 'summary ' + (result.failed ? 'summary-red' : 'summary-green'));
    summary.textContent = result.failed
      ? '失败 ' + result.failed + ' / ' + result.total + '，用时 ' + result.durationMs + 'ms'
      : '全部通过：' + result.total + ' 个用例，用时 ' + result.durationMs + 'ms';
    host.appendChild(summary);
    const table = el('table', 'rt');
    const head = el('tr');
    ['层', '用例', '结果', '说明'].forEach((title) => head.appendChild(el('th', null, title)));
    table.appendChild(head);
    result.cases.forEach((item) => {
      const row = el('tr', item.ok ? 'r-ok' : 'r-bad');
      row.appendChild(el('td', null, item.suite));
      row.appendChild(el('td', null, item.name));
      row.appendChild(el('td', null, item.ok ? 'PASS' : 'FAIL'));
      row.appendChild(el('td', 'detail', item.ok ? '' : item.message));
      table.appendChild(row);
    });
    host.appendChild(table);
  }
  return result;
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', () => run());
} else {
  setTimeout(() => run(), 0);
}
