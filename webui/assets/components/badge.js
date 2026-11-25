/** 语义标签。颜色由调用方按 tone 传入，组件不解释业务含义。 */
import { esc } from '../core/dom.js';

const TONES = {
  green: 'tag-green', red: 'tag-red', orange: 'tag-orange', blue: 'tag-blue', gray: 'tag-gray',
};

export function badgeHTML(spec) {
  const s = spec || {};
  return '<span class="tag ' + (TONES[s.tone] || TONES.gray) + '">' + esc(s.text == null ? '—' : s.text) + '</span>';
}

export function badgeList(specs) {
  return (specs || []).map(badgeHTML).join('');
}
