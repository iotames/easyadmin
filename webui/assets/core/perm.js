/** 前端权限辅助：只用于按钮级显示，安全边界永远在服务端。 */
export function createPerm(codes) {
  const set = new Set(Array.isArray(codes) ? codes : []);
  return {
    can(code) {
      if (!code) return true;
      return set.has('*') || set.has(code);
    },
    hasAny(list) {
      return (list || []).some((code) => set.has('*') || set.has(code));
    },
    list() { return Array.from(set); },
  };
}
