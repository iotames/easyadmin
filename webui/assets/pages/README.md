# 页面入口目录

每个可路由页面放在 `assets/pages/<module-id>/index.js`，默认导出：

```js
export default {
  id: 'system.users',
  async mount(ctx) {
    // ctx.el / ctx.route / ctx.api / ctx.notify / ctx.perm / ctx.store
    // ctx.setTitle / setDescription / setActions / setRefresh / onCleanup
  },
  unmount() {},
};
```

Page ID 必须与 Go Manifest 声明一致，并在 `core/router.js` 的 `registerEntries` 中显式登记；缺失或重复会在启动期报错。当前目录为空，P3 页面建立后填充。
