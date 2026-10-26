package menu

import (
	"github.com/limes-cloud/kratosx/pkg/value"

	managerMenu "github.com/limes-cloud/manager/api/menu"
)

// appKeyword 本服务在 manager 侧的应用标识
const appKeyword = "resource"

// 菜单节点类型（与 manager entity.MenuType* 对应）
const (
	menuTypeApi   = "A"  // 受权限控制的接口
	menuTypePage  = "M"  // 菜单页面
	menuTypeGroup = "G"  // 菜单分组
	menuTypeBasic = "BA" // 基础接口：登录即可用，无需单独授权
)

// group 构造菜单分组节点（G）。
func group(title, key, icon string, hidden bool, children ...*managerMenu.RegisterMenuNode) *managerMenu.RegisterMenuNode {
	return &managerMenu.RegisterMenuNode{
		Title:    title,
		Type:     menuTypeGroup,
		Key:      value.Pointer(key),
		Icon:     value.Pointer(icon),
		Weight:   value.Pointer[int32](0),
		IsHidden: value.Pointer(hidden),
		Children: children,
	}
}

// page 构造菜单页面节点（M）；path/component 对应前端路由与组件。
func page(title, key, icon, path, component string, children ...*managerMenu.RegisterMenuNode) *managerMenu.RegisterMenuNode {
	return &managerMenu.RegisterMenuNode{
		Title:     title,
		Type:      menuTypePage,
		Key:       value.Pointer(key),
		Icon:      value.Pointer(icon),
		Path:      value.Pointer(path),
		Component: value.Pointer(component),
		Weight:    value.Pointer[int32](0),
		IsHidden:  value.Pointer(false),
		IsCache:   value.Pointer(true),
		Children:  children,
	}
}

// basic 构造基础接口节点（BA，无需授权）。
func basic(title, api, method string) *managerMenu.RegisterMenuNode {
	return apiNode(menuTypeBasic, title, api, method, "")
}

// apiPerm 构造受权限控制的接口节点（A）。
func apiPerm(title, api, method, permission string) *managerMenu.RegisterMenuNode {
	return apiNode(menuTypeApi, title, api, method, permission)
}

// apiNode 构造接口类节点（A / BA）。
func apiNode(tp, title, api, method, permission string) *managerMenu.RegisterMenuNode {
	node := &managerMenu.RegisterMenuNode{
		Title:  title,
		Type:   tp,
		Api:    value.Pointer(api),
		Method: value.Pointer(method),
		Weight: value.Pointer[int32](0),
	}
	if permission != "" {
		node.Permission = value.Pointer(permission)
	}
	return node
}

// menus 注册到 manager 的菜单/权限节点（append 模式：只新增/更新，不删旧节点）。
var menus = []*managerMenu.RegisterMenuNode{
	group("资源服务", "Resource", "iconsys-wenjian", false,
		group("基础接口", "ResourceBaseApi", "apps", true,
			basic("查询目录", "/resource/api/directory", "GET"),
			basic("目录详情", "/resource/api/directory/detail", "GET"),
			basic("存储厂商列表", "/resource/api/store/provider", "GET"),
			basic("文件详情", "/resource/api/file/detail", "GET"),
			basic("我的文件列表", "/resource/api/file/user", "GET"),
			basic("目录文件列表", "/resource/api/file", "GET"),
			basic("获取分片大小", "/resource/api/file/upload/chunk-size", "GET"),
			basic("预上传文件", "/resource/api/file/upload/prepare", "POST"),
			basic("获取分片上传地址", "/resource/api/file/upload/part-urls", "POST"),
			basic("完成分片上传", "/resource/api/file/upload/complete", "POST"),
			basic("重命名/移动文件", "/resource/api/file/user", "PUT"),
			basic("删除文件", "/resource/api/file/user", "DELETE"),
		),
		page("对象存储", "ResourceStore", "iconsys-shuju", "/resource/store", "/resource/store",
			apiPerm("查询对象存储", "/resource/api/store", "GET", "resource:store:list"),
			apiPerm("对象存储详情", "/resource/api/store/detail", "GET", "resource:store:detail"),
			apiPerm("新增对象存储", "/resource/api/store", "POST", "resource:store:add"),
			apiPerm("修改对象存储", "/resource/api/store", "PUT", "resource:store:update"),
		),
		page("应用配置", "ResourceAppConfig", "iconsys-shezhi2", "/resource/app-config", "/resource/app-config",
			apiPerm("查询应用配置", "/resource/api/app-config", "GET", "resource:appConfig:list"),
			apiPerm("应用配置详情", "/resource/api/app-config/detail", "GET", "resource:appConfig:detail"),
			apiPerm("新增应用配置", "/resource/api/app-config", "POST", "resource:appConfig:add"),
			apiPerm("修改应用配置", "/resource/api/app-config", "PUT", "resource:appConfig:update"),
		),
		page("文件管理", "ResourceDirectory", "iconsys-wenjian", "/resource/directory", "/resource/directory",
			apiPerm("新增目录", "/resource/api/directory", "POST", "resource:directory:add"),
			apiPerm("修改目录", "/resource/api/directory", "PUT", "resource:directory:update"),
			apiPerm("删除目录", "/resource/api/directory", "DELETE", "resource:directory:delete"),
		),
	),
}
