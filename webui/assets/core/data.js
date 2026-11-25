/**
 * 数据供应商缓存：stale-while-revalidate + 单飞 + TTL + 失败兜底。
 * 只用于全局低频道数据；高频轮询数据留在页面内。
 */
const DEFAULT_TTL_MS = 10 * 60 * 1000;
const providers = Object.create(null);

export function provide(name, opts) {
  providers[name] = {
    fetch: opts.fetch,
    ttl: opts.ttl === undefined ? DEFAULT_TTL_MS : opts.ttl,
    fallback: opts.fallback,
    value: null,
    at: 0,
    inflight: null,
  };
}

function fresh(provider) {
  return provider.value != null && provider.ttl >= 0 &&
    (provider.ttl === 0 || Date.now() - provider.at < provider.ttl);
}

function kick(name, force) {
  const provider = providers[name];
  if (!provider) return null;
  if (provider.inflight) return provider.inflight;
  if (!force && fresh(provider)) return null;
  provider.inflight = Promise.resolve()
    .then(provider.fetch)
    .then((value) => {
      provider.value = value == null ? provider.fallback : value;
      provider.at = Date.now();
    })
    .catch(() => { /* 拉取失败保留旧值/默认值，页面自行降级 */ })
    .finally(() => { provider.inflight = null; });
  return provider.inflight;
}

export function get(name) {
  const provider = providers[name];
  if (!provider) return undefined;
  kick(name, false);
  return provider.value != null ? provider.value : provider.fallback;
}

export function peek(name) {
  const provider = providers[name];
  if (!provider) return undefined;
  return provider.value != null ? provider.value : provider.fallback;
}

export function load(name) {
  const provider = providers[name];
  if (!provider) return Promise.resolve(undefined);
  if (fresh(provider)) return Promise.resolve(provider.value);
  return kick(name, provider.ttl < 0).then(() => (
    provider.value != null ? provider.value : provider.fallback
  ));
}

export function refresh(name) {
  kick(name, true);
}

export function reload(name) {
  const provider = providers[name];
  if (!provider) return Promise.resolve(undefined);
  const pending = kick(name, true);
  if (!pending) return Promise.resolve(provider.value != null ? provider.value : provider.fallback);
  return pending.then(() => (provider.value != null ? provider.value : provider.fallback));
}

export function set(name, value) {
  const provider = providers[name];
  if (!provider) return;
  provider.value = value;
  provider.at = Date.now();
}

export function reset() {
  Object.keys(providers).forEach((key) => { delete providers[key]; });
}
