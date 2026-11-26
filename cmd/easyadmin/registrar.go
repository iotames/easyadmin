package main

import (
	"net/http"

	"github.com/iotames/easyadmin/infra/httpapi"
	"github.com/iotames/easyadmin/kernel/module"
)

// registrar 把 kernel 的模块登记契约桥接到 infra/httpapi 的路由与元信息目录。
// 组合根是唯一同时看见 kernel 与 infra 的层，因此由这里做类型适配。
type registrar struct {
	router  *httpapi.Router
	catalog *httpapi.Catalog
}

func (r registrar) Handle(pattern string, h http.Handler) error {
	return r.router.Handle(pattern, h)
}

func (r registrar) RegisterPage(p module.PageMeta) error {
	return r.catalog.AddPage(httpapi.Page{
		ID: p.ID, Title: p.Title, Path: p.Path, Menu: p.Menu, Order: p.Order, Perm: p.Perm,
	})
}

func (r registrar) RegisterPermission(p module.PermissionMeta) error {
	return r.catalog.AddPermission(httpapi.Permission{Code: p.Code, Name: p.Name})
}
