/**
 * 前端唯一请求入口。
 * - 统一解析 { ok, data, error, meta }；
 * - 写操作自动附加 /api/meta 返回的 X-CSRF-Token；
 * - 401 交回入口处理，不在页面各自跳转；
 * - 不自动弹提示，调用方决定 notify.error 或静默。
 */
import { queryString } from './dom.js';

export class ApiError extends Error {
  constructor(message, options) {
    super(message);
    const opts = options || {};
    this.name = 'ApiError';
    this.code = opts.code || '';
    this.status = opts.status || 0;
    this.requestId = opts.requestId || '';
    this.details = opts.details || {};
    this.meta = opts.meta || {};
  }
}

const DEFAULT_TIMEOUT_MS = 15000;
const loading = { count: 0, host: null, label: null };

let config = {
  baseUrl: '',
  timeout: DEFAULT_TIMEOUT_MS,
  onUnauthorized: null,
  onWarning: null,
  onUnreachable: null,
};

let csrfToken = '';

export function configure(options) {
  config = Object.assign({}, config, options || {});
}

export function setCsrfToken(token) { csrfToken = token || ''; }
export function getCsrfToken() { return csrfToken; }

export function setUnauthorizedHandler(fn) { config.onUnauthorized = fn; }

function ensureLoadingHost() {
  if (loading.host) return loading.host;
  loading.host = document.createElement('div');
  loading.host.id = 'ea-api-loading';
  const bar = document.createElement('div');
  bar.className = 'ea-loading-bar';
  loading.label = document.createElement('div');
  loading.label.className = 'ea-loading-text';
  loading.host.appendChild(bar);
  loading.host.appendChild(loading.label);
  document.body.appendChild(loading.host);
  return loading.host;
}

function loadingStart(text) {
  if (typeof document === 'undefined') return;
  const host = ensureLoadingHost();
  loading.count += 1;
  loading.label.textContent = text || '数据请求中…';
  host.classList.add('active');
}

function loadingEnd() {
  if (!loading.host) return;
  loading.count = Math.max(0, loading.count - 1);
  if (loading.count === 0) loading.host.classList.remove('active');
}

function reachable(value) {
  if (typeof config.onUnreachable === 'function') config.onUnreachable(value);
}

function timeoutSignal(ms) {
  if (typeof AbortSignal !== 'undefined' && typeof AbortSignal.timeout === 'function') {
    return AbortSignal.timeout(ms);
  }
  const controller = new AbortController();
  setTimeout(() => controller.abort(), ms);
  return controller.signal;
}

function payloadError(payload, status, fallback) {
  const error = (payload && payload.error) || {};
  const meta = (payload && payload.meta) || {};
  return new ApiError(error.message || fallback, {
    code: error.code || '',
    status,
    requestId: meta.request_id || '',
    details: error.details || {},
    meta,
  });
}

async function readPayload(response) {
  const text = await response.text();
  if (!text) return null;
  try {
    return JSON.parse(text);
  } catch (e) {
    throw new ApiError('服务端响应不是合法 JSON（HTTP ' + response.status + '）：请查看服务端日志后重试', {
      status: response.status,
    });
  }
}

/**
 * @param {'GET'|'POST'|'PUT'|'PATCH'|'DELETE'} method
 * @param {string} path
 * @param {{query?:object,body?:any,timeout?:number,loadingText?:string,silent?:boolean}} [options]
 */
export async function request(method, path, options) {
  const opts = options || {};
  const url = (config.baseUrl || '') + path + queryString(opts.query);
  const headers = { Accept: 'application/json', 'X-Requested-With': 'fetch' };
  let body;
  if (opts.body !== undefined && opts.body !== null) {
    headers['Content-Type'] = 'application/json';
    body = JSON.stringify(opts.body);
  }
  const needsCsrf = method !== 'GET' && method !== 'HEAD';
  if (needsCsrf && csrfToken) headers['X-CSRF-Token'] = csrfToken;

  const effectiveTimeout = opts.timeout || config.timeout || DEFAULT_TIMEOUT_MS;
  const showLoading = !opts.silent;
  if (showLoading) loadingStart(opts.loadingText);

  try {
    let response;
    try {
      response = await fetch(url, {
        method,
        headers,
        body,
        cache: 'no-store',
        credentials: 'same-origin',
        signal: timeoutSignal(effectiveTimeout),
      });
    } catch (err) {
      if (err && (err.name === 'TimeoutError' || err.name === 'AbortError')) {
        throw new ApiError('请求超时（' + Math.round(effectiveTimeout / 1000) + ' 秒）：可缩小查询范围或稍后重试', {
          code: 'NETWORK_TIMEOUT',
          status: 408,
        });
      }
      if (err instanceof TypeError) {
        reachable(true);
        throw new ApiError('接口不可达：浏览器无法连接服务端，请确认服务正在运行后重试', {
          code: 'NETWORK_UNREACHABLE',
          status: 0,
        });
      }
      throw err;
    }

    if (response.status === 401) {
      if (typeof config.onUnauthorized === 'function') config.onUnauthorized();
      throw new ApiError('未登录或会话已失效：请重新登录后重试', {
        code: 'AUTH_REQUIRED',
        status: 401,
      });
    }

    const payload = response.status === 204 ? null : await readPayload(response);
    reachable(false);

    if (!response.ok) {
      throw payloadError(payload, response.status,
        '请求失败（HTTP ' + response.status + '）：请查看服务端日志后重试');
    }
    if (payload && payload.ok === false) {
      throw payloadError(payload, response.status, '服务端返回失败但未给出原因：请查看服务端日志后重试');
    }

    const meta = (payload && payload.meta) || {};
    if (meta.warning && typeof config.onWarning === 'function') config.onWarning(meta.warning);
    if (meta.page !== undefined || meta.page_size !== undefined) {
      return {
        items: (payload.data && payload.data.items) || [],
        page: meta.page,
        pageSize: meta.page_size,
        total: meta.total,
      };
    }
    return payload ? payload.data : null;
  } finally {
    if (showLoading) loadingEnd();
  }
}

export function get(path, options) {
  return request('GET', path, options);
}

export function post(path, body, options) {
  return request('POST', path, Object.assign({}, options, { body }));
}

export function put(path, body, options) {
  return request('PUT', path, Object.assign({}, options, { body }));
}

export function patch(path, body, options) {
  return request('PATCH', path, Object.assign({}, options, { body }));
}

export function del(path, options) {
  return request('DELETE', path, options);
}

export const api = { get, post, put, patch, del, request, configure, setCsrfToken, getCsrfToken, setUnauthorizedHandler, ApiError };
export default api;
