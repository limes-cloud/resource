# 模型与分页 (model)

> 包路径：`github.com/limes-cloud/kratosx/model`

## 概述

提供通用的数据模型基础结构和分页查询工具。

---

## model/page — 分页查询

### PageOptions 分页选项

```go
type PageOptions struct {
    Page     uint32  // 页码（从 1 开始）
    PageSize uint32  // 每页数量
    Model    any     // GORM 模型
    Scopes   []func(*gorm.DB) *gorm.DB  // 查询条件
}
```

### 使用方式

```go
import "github.com/limes-cloud/kratosx/model/page"

// 分页查询
var list []*entity.Example
total, err := page.Page(ctx.DB(), &page.PageOptions{
    Page:     req.Page,
    PageSize: req.PageSize,
    Model:    &entity.Example{},
    Scopes: []func(*gorm.DB) *gorm.DB{
        func(db *gorm.DB) *gorm.DB {
            return db.Where("user_id = ?", req.UserID)
        },
        func(db *gorm.DB) *gorm.DB {
            if req.Type != nil {
                return db.Where("type = ?", *req.Type)
            }
            return db
        },
    },
}, &list)
```

---

## model/hook — 模型钩子

提供 GORM 模型的全局钩子机制，用于自动填充字段（如 created_by、updated_by）。

### 使用方式

```go
import "github.com/limes-cloud/kratosx/model/hook"

// 注册全局钩子
hook.Register(func(ctx context.Context, model any, operation string) {
    // operation: "create" / "update" / "delete"
    // 可以从 ctx 中获取当前用户信息并填充到 model
})
```

### 跳过钩子

```go
// 某些场景需要跳过钩子（如数据迁移）
ctx := kratosx.MustContext(c, kratosx.WithSkipDBHook())
ctx.DB().Create(&entity)  // 不触发钩子
```
