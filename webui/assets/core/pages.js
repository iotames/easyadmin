/**
 * core 页面注册表：Page ID 必须与 Go Manifest Pages 一一对应。
 * 业务页面由项目自己的 /assets/app/pages.js 注册，不在此处登记。
 */
import { registerEntries } from './router.js';

export const CORE_PAGES = {
  'dashboard': () => import('../pages/dashboard/index.js'),
  'profile': () => import('../pages/profile/index.js'),
  'system.users': () => import('../pages/system/users/index.js'),
  'system.roles': () => import('../pages/system/roles/index.js'),
  'system.audit': () => import('../pages/system/audit/index.js'),
  'system.config': () => import('../pages/system/config/index.js'),
};

registerEntries(CORE_PAGES);