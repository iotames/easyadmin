/** 主题切换：只改 data-theme；首屏防闪烁由 index.html 内联脚本负责。 */
export const THEMES = [
  { name: 'light', label: '浅色' },
  { name: 'dark', label: '深色' },
  { name: 'green', label: '护眼绿' },
];

export const THEME_KEY = 'easyadmin.theme';
export const DEFAULT_THEME = 'light';

export function isValidTheme(name) {
  return THEMES.some((item) => item.name === name);
}

export function getTheme() {
  let value = null;
  try { value = localStorage.getItem(THEME_KEY); } catch (e) { /* 隐私模式用默认值 */ }
  return isValidTheme(value) ? value : DEFAULT_THEME;
}

export function applyTheme(name) {
  const theme = isValidTheme(name) ? name : DEFAULT_THEME;
  document.documentElement.setAttribute('data-theme', theme);
  try { localStorage.setItem(THEME_KEY, theme); } catch (e) { /* 忽略存储异常 */ }
  const select = document.getElementById('theme-select');
  if (select) select.value = theme;
  return theme;
}

/** 绑定顶栏主题下拉；元素不存在时静默跳过。 */
export function bindTheme() {
  const select = document.getElementById('theme-select');
  if (!select) return;
  select.value = getTheme();
  select.addEventListener('change', () => applyTheme(select.value));
}
