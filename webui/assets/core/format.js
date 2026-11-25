/** 时间与数值格式化。所有函数对空值安全，不抛错。 */
export const pad2 = (value) => String(value).padStart(2, '0');

export function fmtInt(value) {
  const n = Math.round(Number(value) || 0);
  const sign = n < 0 ? '-' : '';
  return sign + String(Math.abs(n)).replace(/\B(?=(\d{3})+(?!\d))/g, ',');
}

export function toDate(value) {
  if (value === null || value === undefined || value === '') return null;
  if (value instanceof Date) return Number.isNaN(value.getTime()) ? null : value;
  const date = new Date(String(value));
  return Number.isNaN(date.getTime()) ? null : date;
}

export function fmtTime(value) {
  const date = toDate(value);
  return date ? pad2(date.getHours()) + ':' + pad2(date.getMinutes()) + ':' + pad2(date.getSeconds()) : '—';
}

export function fmtDate(value) {
  const date = toDate(value);
  return date ? date.getFullYear() + '-' + pad2(date.getMonth() + 1) + '-' + pad2(date.getDate()) : '';
}

export function fmtDateTime(value) {
  const date = toDate(value);
  if (!date) return '—';
  return fmtDate(date) + ' ' + pad2(date.getHours()) + ':' + pad2(date.getMinutes()) + ':' + pad2(date.getSeconds());
}

export function toInputDateTime(value) {
  const date = toDate(value) || new Date();
  return fmtDate(date) + 'T' + pad2(date.getHours()) + ':' + pad2(date.getMinutes());
}

export function fmtBytes(bytes) {
  const b = Number(bytes) || 0;
  if (b < 1024) return b + ' B';
  if (b < 1024 * 1024) return (b / 1024).toFixed(1) + ' KB';
  if (b < 1024 * 1024 * 1024) return (b / 1024 / 1024).toFixed(2) + ' MB';
  return (b / 1024 / 1024 / 1024).toFixed(2) + ' GB';
}

export function fmtDuration(seconds) {
  const raw = Math.max(0, Number(seconds) || 0);
  if (raw > 0 && raw < 1) return Math.round(raw * 1000) + ' 毫秒';
  const total = Math.round(raw);
  const days = Math.floor(total / 86400);
  const hours = Math.floor((total % 86400) / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = total % 60;
  if (days > 0) return days + ' 天 ' + hours + ' 小时';
  if (hours > 0) return hours + ' 小时 ' + minutes + ' 分';
  if (minutes > 0) return minutes + ' 分 ' + secs + ' 秒';
  return secs + ' 秒';
}

export function fmtSpeed(bytes, seconds) {
  const b = Number(bytes) || 0;
  const s = Number(seconds) || 0;
  if (b <= 0 || s <= 0) return '—';
  const rate = b / s;
  if (rate >= 1024 * 1024) return (rate / 1024 / 1024).toFixed(1) + ' MB/s';
  if (rate >= 1024) return Math.round(rate / 1024) + ' KB/s';
  return '<1 KB/s';
}

export function fmtMinutes(minutes) {
  const n = Math.max(0, Math.round(Number(minutes) || 0));
  if (n >= 1440 && n % 1440 === 0) return n / 1440 + ' 天';
  if (n >= 60 && n % 60 === 0) return n / 60 + ' 小时';
  return n + ' 分钟';
}

export function truncate(value, length) {
  const text = String(value || '');
  return text.length > length ? text.slice(0, length) + '…' : text;
}
