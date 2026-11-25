/**
 * 命名数据源表单：连接名 / 驱动 / 直连或 SSH 隧道 / DSN。
 * 只产出标记与读取值，不碰 api、不读业务 store；保存与测试动作由调用方接。
 */
import { esc } from '../core/dom.js';
import { form } from './form.js';

export function dsnFormHTML(options) {
  const opts = options || {};
  const prefix = opts.prefix || 'dsn';
  const values = opts.values || {};
  const drivers = opts.drivers || [];
  const servers = opts.servers || [];
  const scope = opts.scope ? { 'data-scope': opts.scope } : null;
  const field = (name) => (scope ? Object.assign({}, scope, { 'data-field': name }) : null);
  const driverOptions = drivers.map((driver) => [driver.value, driver.label]);
  const overSSH = (values.connect_mode || 'direct') === 'ssh';
  const serverOptions = [['0', '不使用 SSH 隧道']].concat(servers.map((server) => [
    String(server.id),
    server.name + (server.host ? '（' + server.host + '）' : ''),
  ]));
  const enabledDrivers = {};
  drivers.forEach((driver) => { enabledDrivers[driver.value] = true; });
  const templates = (opts.templates || []).map((template) => {
    const enabled = !!enabledDrivers[template.value];
    return '<button class="btn btn-sm" data-act="' +
      esc((opts.acts && opts.acts.template) || (prefix + '-template')) + '" data-template="' + esc(template.value) + '"' +
      (enabled ? '' : ' disabled') + ' title="' + esc(enabled ? (template.note || '') : '当前构建未启用该驱动') + '">' +
      esc(template.label) + (enabled ? '' : '（未启用）') + '</button>';
  }).join(' ');

  return form.inline(
    form.input({
      id: prefix + '-name', width: 'sm', placeholder: '连接名（唯一，如 erp-pg）', value: values.name || '',
      attrs: field('name'), disabled: !!opts.editing,
      title: opts.editing ? '连接名是数据源的唯一标识，编辑时不允许改名' : '',
    }) +
    form.select({ id: prefix + '-driver', width: 'sm', options: driverOptions, selected: values.driver || '', attrs: field('driver') }) +
    form.select({
      id: prefix + '-mode', width: 'xs', selected: values.connect_mode || 'direct',
      options: [['direct', '直连'], ['ssh', 'OverSSH']], attrs: field('connect_mode'),
    }) +
    form.select({
      id: prefix + '-server', width: 'sm', options: serverOptions, selected: String(values.server_id || 0),
      attrs: field('server_id'), disabled: !overSSH,
    }) +
    form.input({
      id: prefix + '-dsn', width: 'lg', mono: true,
      placeholder: opts.editing ? '连接串 DSN（留空保持原值）' : '连接串 DSN（含凭据，服务端脱敏展示）',
      value: values.dsn || '', attrs: field('dsn'),
    }) +
    '<button class="btn btn-sm" data-act="' + esc((opts.acts && opts.acts.test) || (prefix + '-test')) + '"' +
    (opts.testing ? ' disabled' : '') + '>' + (opts.testing ? '测试中…' : '连通测试') + '</button>' +
    '<button class="btn btn-sm btn-primary" data-act="' + esc((opts.acts && opts.acts.save) || (prefix + '-add')) + '">' +
    esc(opts.saveLabel || '添加') + '</button>'
  ) + (templates ? '<div class="form-inline" style="margin-top:8px">' + templates + '</div>' : '') +
    form.hint('直连不需要选服务器；OverSSH 必须选择已登记的服务器建立隧道，DSN 中的主机名按跳板视角填写。' +
      '连通测试不落盘；列表与接口只显示脱敏后的连接串，凭据不会回显。');
}

export function readDsnForm(host, prefix) {
  const p = prefix || 'dsn';
  const root = host || document;
  const value = (id) => {
    const el = root.querySelector('#' + id);
    return el ? el.value : '';
  };
  return {
    name: String(value(p + '-name') || '').trim(),
    driver: String(value(p + '-driver') || '').trim(),
    dsn: String(value(p + '-dsn') || '').trim(),
    connect_mode: String(value(p + '-mode') || 'direct').trim(),
    server_id: Number(value(p + '-server') || 0) || 0,
  };
}

export const dsnForm = { html: dsnFormHTML, read: readDsnForm };
