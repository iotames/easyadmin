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

// pagesSnapshot 返回按登记顺序的页面快照。
func (c *Catalog) pagesSnapshot() []Page {
	c.mu.Lock()
	defer c.mu.Unlock()
	pages := make([]Page, 0, len(c.pageSeq))
	for _, id := range c.pageSeq {
		pages = append(pages, c.pages[id])
	}
	return pages
}

// Menus 返回全部页面聚合的菜单树（供未启用认证的默认 meta 使用）。
func (c *Catalog) Menus() []MenuNode { return buildMenus(c.pagesSnapshot(), nil, false) }

// MenusFor 返回按权限过滤后的菜单树；perms 含 "*" 时全量可见。
func (c *Catalog) MenusFor(perms []string) []MenuNode {
	return buildMenus(c.pagesSnapshot(), perms, true)
}

func allowed(perms []string, required string) bool {
	if required == "" {
		return true
	}
	for _, p := range perms {
		if p == "*" || p == required {
			return true
		}
	}
	return false
}

func buildMenus(pages []Page, perms []string, filter bool) []MenuNode {
	type group struct {
		name  string
		order int
		pages []Page
	}
	groups := map[string]*group{}
	var groupOrder []string
	var leaves []Page
	for _, p := range pages {
		if filter && !allowed(perms, p.Perm) {
			continue
		}
		if p.Menu == "" {
			leaves = append(leaves, p)
			continue
		}
		g, ok := groups[p.Menu]
		if !ok {
			g = &group{name: p.Menu, order: p.Order}
			groups[p.Menu] = g
			groupOrder = append(groupOrder, p.Menu)
		} else if p.Order < g.order {
			g.order = p.Order
		}
		g.pages = append(g.pages, p)
	}

	type entry struct {
		order int
		key   string
		node  MenuNode
	}
	entries := make([]entry, 0, len(groupOrder)+len(leaves))
	for _, name := range groupOrder {
		g := groups[name]
		sort.SliceStable(g.pages, func(i, j int) bool {
			if g.pages[i].Order != g.pages[j].Order {
				return g.pages[i].Order < g.pages[j].Order
			}
			return g.pages[i].ID < g.pages[j].ID
		})
		node := MenuNode{Key: g.name, Title: g.name}
		for _, p := range g.pages {
			node.Children = append(node.Children, MenuNode{Key: p.ID, Title: p.Title, Path: p.Path})
		}
		entries = append(entries, entry{order: g.order, key: g.name, node: node})
	}
	for _, p := range leaves {
		entries = append(entries, entry{
			order: p.Order,
			key:   p.ID,
			node:  MenuNode{Key: p.ID, Title: p.Title, Path: p.Path},
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].order != entries[j].order {
			return entries[i].order < entries[j].order
		}
		return entries[i].key < entries[j].key
	})
	out := make([]MenuNode, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.node)
	}
	return out
}
