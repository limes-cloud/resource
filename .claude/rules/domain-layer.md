# Domain 层规则

## entity/ — 领域实体

### 结构体规范

- 纯结构体，只包含字段和简单方法，不含业务逻辑
- DB tag 使用 `gorm`（`gorm:"column:xxx"`，主键用 `gorm:"primaryKey"`）
- JSON tag 使用小驼峰
- 公共字段（id、created_at、updated_at）通过嵌入 kratosx 的 `model.BaseModel`（`github.com/limes-cloud/kratosx/model`，与 manager 一致）提供，放在结构体最后一行。主键 `Id` 为 **uint32 自增**（`gorm:"primaryKey;autoIncrement"`），`CreatedAt` / `UpdatedAt` 为 `int64`（Unix 秒）

```go
package entity

type Example struct {
    Name   string  `json:"name"   gorm:"column:name"`
    Status *uint8  `json:"status" gorm:"column:status"` // 1启用 0停用
    Remark *string `json:"remark" gorm:"column:remark"` // 备注；NULL=无
    model.BaseModel       // 提供 uint32 自增 Id、CreatedAt、UpdatedAt，放最后
}
```

### 指针字段

两类字段用指针（判据见 code-style.md「指针字段」）：DB 可空列，**以及合法取值可能是 Go 零值（`0`/`""`/`false`）的列**（如 `status` 的 `0`=禁用，DB `NOT NULL` 仍要指针，否则 `Updates` 会跳过）：

```go
type Example struct {
    Name       string   `json:"name"       gorm:"column:name"`
    Status     *uint8   `json:"status"     gorm:"column:status"`     // 0=停用 是合法值，DB 非空仍用指针
    Score      *float64 `json:"score"      gorm:"column:score"`      // DB 可空
    ExpireAt   *int64   `json:"expireAt"   gorm:"column:expire_at"`  // DB 可空，NULL = 永不过期
    model.BaseModel
}
```

### 关联字段

关联关系在 repository 层手动查询，不定义外键标签：

```go
type Example struct {
    ParentID uint32 `json:"parentId" gorm:"column:parent_id"`
    Name     string `json:"name"     gorm:"column:name"`
    Remark   string `json:"remark"   gorm:"column:remark"`
    model.BaseModel
}
```

### 接口定义规范

- 第一个参数必须是 `ctx core.Context`
- 方法命名：`GetXxx` / `GetXxxByYyy` / `ListXxx` / `CreateXxx` / `UpdateXxx` / `DeleteXxx`（动词 + 实体名，与 manager 完全一致）
- 查询单条返回 `(*entity.Xxx, error)`
- 查询列表（无分页）返回 `([]*entity.Xxx, error)`
- 查询列表（分页）返回 `([]*entity.Xxx, uint32, error)`（第二个值为总数 uint32）
- 创建方法返回 `(uint32, error)`（本项目主键为 uint32 自增，返回自增后的 `ent.Id`）

```go
package repository

import (
    "github.com/limes-cloud/resource/internal/core"
    "github.com/limes-cloud/resource/internal/domain/entity"
    "github.com/limes-cloud/resource/internal/types"
)

// Example 示例仓储接口
type Example interface {
    // GetExample 根据 ID 查询
    GetExample(ctx core.Context, id uint32) (*entity.Example, error)

    // ListExample 分页查询列表
    ListExample(ctx core.Context, req *types.ListExampleRequest) ([]*entity.Example, uint32, error)

    // CreateExample 创建记录
    CreateExample(ctx core.Context, ent *entity.Example) (uint32, error)

    // DeleteExampleByIDs 批量删除
    DeleteExampleByIDs(ctx core.Context, ids []uint32) error
}
```

### 特殊查询方法

对于固定过滤维度的查询，可在接口中定义专用方法（而非全部挤进通用列表查询）：

```go
// Example 示例仓储接口
type Example interface {
    // ListExample 分页查询列表
    ListExample(ctx core.Context, req *types.ListExampleRequest) ([]*entity.Example, uint32, error)

    // ListExampleByStatus 按状态查询（无分页）
    ListExampleByStatus(ctx core.Context, status uint8, limit int) ([]*entity.Example, error)
}
```

### 事务方法

事务不在 repository 接口中定义，而是通过 `ctx.Transaction()` 在 service 层编排：

```go
// service 层使用事务
err := ctx.Transaction(func(ctx core.Context) error {
    if err := u.repo.DeleteExampleByIDs(ctx, ids); err != nil {
        return err
    }
    return u.other.UpdateExample(ctx, ent)
})
```

## service/ — 领域服务

### 构造函数

通过 `NewXxx()` 注入 repository 接口：

```go
type Example struct {
    repo  repository.Example
    other repository.Other
}

func NewExample(
    repo repository.Example,
    other repository.Other,
) *Example {
    return &Example{
        repo:  repo,
        other: other,
    }
}
```

### 业务方法

- 第一个参数 `ctx core.Context`
- 错误日志 + 返回 proto 错误函数
- 复杂业务逻辑可拆分为私有方法

```go
// Create 创建记录
func (s *Example) Create(ctx core.Context, req *types.CreateExampleRequest) (uint32, error) {
    ent := &entity.Example{
        Name:   req.Name,
        Status: req.Status,
        Remark: req.Remark,
    }
    id, err := s.repo.CreateExample(ctx, ent)
    if err != nil {
        ctx.Logger().Warnw("msg", "create example error", "err", err.Error())
        return 0, errors.CreateError()
    }
    return id, nil
}
```

### 禁止事项

1. **禁止在 service 中直接 import DB / Redis 客户端**：数据库操作必须通过 repository 接口
2. **禁止在 service 中直接 import infra 包**：依赖注入在 app 层完成
3. **禁止在 service 中返回原生 error**：必须使用 proto 生成的错误函数
4. **禁止在 service 中处理 HTTP/gRPC 协议细节**：proto 转换在 app 层完成