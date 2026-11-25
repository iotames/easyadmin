/** 时间范围：四输入组装 RFC3339；严格校验，拒绝 2026-02-31 这类静默进位。 */
export const DR_PARTS = ['fromDate', 'fromTime', 'toDate', 'toTime'];
const pad2 = (n) => String(n).padStart(2, '0');

export function today() {
  const now = new Date();
  return now.getFullYear() + '-' + pad2(now.getMonth() + 1) + '-' + pad2(now.getDate());
}

export function at(date, time, fallbackTime) {
  const dm = /^(\d{4})-(\d{2})-(\d{2})$/.exec(String(date == null ? '' : date));
  if (!dm) return null;
  const tm = /^(\d{2}):(\d{2})/.exec(String(time || fallbackTime || '00:00'));
  if (!tm) return null;
  const year = Number(dm[1]);
  const month = Number(dm[2]);
  const day = Number(dm[3]);
  const hour = Number(tm[1]);
  const minute = Number(tm[2]);
  if (month < 1 || month > 12 || day < 1 || day > 31 || hour > 23 || minute > 59) return null;
  const d = new Date(year, month - 1, day, hour, minute, 0, 0);
  if (d.getFullYear() !== year || d.getMonth() !== month - 1 || d.getDate() !== day) return null;
  return d.toISOString();
}

export function from(query) {
  const q = query || {};
  return at(q.fromDate || today(), q.fromTime, '00:00');
}

export function to(query) {
  const q = query || {};
  return at(q.toDate || today(), q.toTime, '23:59');
}

export function recentDays(n) {
  const now = new Date();
  const fromDate = new Date(now.getTime() - (Number(n) || 1) * 24 * 3600 * 1000);
  const day1 = fromDate.getFullYear() + '-' + pad2(fromDate.getMonth() + 1) + '-' + pad2(fromDate.getDate());
  const day2 = now.getFullYear() + '-' + pad2(now.getMonth() + 1) + '-' + pad2(now.getDate());
  return {
    fromDate: day1,
    fromTime: pad2(fromDate.getHours()) + ':' + pad2(fromDate.getMinutes()),
    toDate: day2,
    toTime: pad2(now.getHours()) + ':' + pad2(now.getMinutes()),
  };
}

export const dateRange = { today, at, from, to, recentDays, DR_PARTS };
