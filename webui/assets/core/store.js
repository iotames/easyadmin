/** 极简显式状态容器：不做深层响应式，不做隐式依赖收集。 */
export function createStore(initial) {
  const data = Object.assign(Object.create(null), initial || {});
  const listeners = new Map();

  function set(key, value) {
    data[key] = value;
    const list = listeners.get(key);
    if (list) list.slice().forEach((fn) => fn(value, key));
  }

  function get(key) { return data[key]; }
  function has(key) { return Object.prototype.hasOwnProperty.call(data, key); }
  function remove(key) { delete data[key]; set(key, undefined); }
  function snapshot() { return Object.assign({}, data); }

  function subscribe(key, fn) {
    if (!listeners.has(key)) listeners.set(key, []);
    listeners.get(key).push(fn);
    return () => {
      const list = listeners.get(key) || [];
      const index = list.indexOf(fn);
      if (index >= 0) list.splice(index, 1);
    };
  }

  return { set, get, has, remove, snapshot, subscribe };
}

/** 全局低频状态：应用信息、当前用户、权限、菜单、主题。页面数据不进这里。 */
export const store = createStore();
