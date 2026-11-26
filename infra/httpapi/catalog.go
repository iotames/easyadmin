package httpapi

import (
	"fmt"
	"sort"
	"sync"
)

// Page 是页面元信息（与 kernel/module.PageMeta 语义一致，由组合根桥接）。
type Page struct {
	ID    string
	Title string
	Path  string
	Menu  string
	Order int
	Perm  string
}

// Permission 是权限码元信息。
type Permission struct {
	Code string
	Name string
}

// MenuNode 是 /api/meta 菜单树节点，结构与前端 menu.js 消费契约一致。
type MenuNode struct {
	Key      string     `json:"key"`
	Title    string     `json:"title"`
	Icon     string     `json:"icon,omitempty"`
	Path     string     `json:"path,omitempty"`
	Children []MenuNode `json:"children,omitempty"`
}

// Catalog 收集页面与权限并检测冲突。
type Catalog struct {
	mu      sync.Mutex
	pages   map[string]Page
	perms   map[string]Permission
	pageSeq []string
}

// NewCatalog 创建目录。
func NewCatalog() *Catalog {
	return &Catalog{pages: make(map[string]Page), perms: make(map[string]Permission)}
}

// AddPage 登记页面；重复 Page ID 返回错误。
func (c *Catalog) AddPage(p Page) error {
	if p.ID == "" {
		return fmt.Errorf("httpapi: 页面 ID 为空")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.pages[p.ID]; ok {
		return fmt.Errorf("httpapi: 页面 ID 重复: %s", p.ID)
	}
	c.pages[p.ID] = p
	c.pageSeq = append(c.pageSeq, p.ID)
	return nil
}

// AddPermission 登记权限码；重复 Code 返回错误。
func (c *Catalog) AddPermission(p Permission) error {
	if p.Code == "" {
		return fmt.Errorf("httpapi: 权限码为空")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.perms[p.Code]; ok {
		return fmt.Errorf("httpapi: 权限码重复: %s", p.Code)
	}
	c.perms[p.Code] = p
	return nil
}

// PermissionCodes 返回按字典序排序的权限码。
func (c *Catalog) PermissionCodes() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.perms))
	for code := range c.perms {
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}

// Menus 由带 Menu 的页面聚合菜单树；无 Menu 的页面为顶层叶子。
func (c *Catalog) Menus() []MenuNode {
	c.mu.Lock()
	pages := make([]Page, 0, len(c.pageSeq))
	for _, id := range c.pageSeq {
		pages = append(pages, c.pages[id])
	}
	c.mu.Unlock()

	groups := map[string][]Page{}
	var groupOrder []string
	var leaves []MenuNode
	for _, p := range pages {
		if p.Menu == "" {
			leaves = append(leaves, MenuNode{Key: p.ID, Title: p.Title, Path: p.Path})
			continue
		}
		if _, ok := groups[p.Menu]; !ok {
			groupOrder = append(groupOrder, p.Menu)
		}
		groups[p.Menu] = append(groups[p.Menu], p)
	}
	out := make([]MenuNode, 0, len(groupOrder)+len(leaves))
	for _, name := range groupOrder {
		node := MenuNode{Key: name, Title: name}
		for _, p := range groups[name] {
			node.Children = append(node.Children, MenuNode{Key: p.ID, Title: p.Title, Path: p.Path})
		}
		out = append(out, node)
	}
	out = append(out, leaves...)
	return out
}
