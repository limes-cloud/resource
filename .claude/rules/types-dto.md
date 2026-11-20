# Types DTO 规则

> 参考 limes-cloud/manager/internal/types/ 的 DTO 模式。

## 请求 DTO

### 基本规则

- 仅包含 JSON tag，不含 DB tag
- 查询条件中的可选参数使用指针
- 必填字段使用值类型
- **可被设置成零值的更新字段必须用指针**：更新类 DTO 里凡"合法值包含 `0` / `""` / `false`"的字段（状态、开关、描述 / 简介、允许设 `0` 的数值），一律 `*T`——proto 侧对应 `optional`（生成 Go 指针），service 据"指针是否为 nil"判断是否要改该列，再透传给 infra 的实体更新（见 proto-and-api.md 编写规则 11、infra-layer.md「更新用实体、零值列用指针」）。用值类型会让"改成零值"与"不改"无法区分而被静默丢弃。

```go
package types

// CreateExampleRequest 创建请求
type CreateExampleRequest struct {
    Name   string  `json:"name"`   // 必填
    Status *uint8  `json:"status"` // 可选，允许设 0（停用）
    Remark *string `json:"remark"` // 可选，允许设 ""
}

// GetExampleRequest 获取请求
type GetExampleRequest struct {
    ID   *uint32 `json:"id"`   // 可选，ID 或 Name 二选一
    Name *string `json:"name"` // 可选
}
```

### 列表查询 DTO

列表查询嵌入 `page.Search`，分页/排序由 `page.SearchScopes` 统一处理：

```go
import "github.com/limes-cloud/kratosx/model/page"

// ListExampleRequest 列表查询请求
type ListExampleRequest struct {
    page.Search        // 含 Page、PageSize、OrderBy、Order

    // 过滤条件
    Name   *string  `json:"name"`   // 可选，按名称过滤
    Status *uint8   `json:"status"` // 可选，按状态过滤
    InIDs  []uint32 `json:"inIds"`  // 可选，ID 包含过滤
}
```

### 中转参数

有些参数不在 proto 中定义，而是在 service 层逻辑中推导后设置，称为"中转参数"：

```go
type ListExampleRequest struct {
    // ... 查询条件

    // 中转参数（由 service 层根据权限推导设置）
    InParentIDs []uint32 `json:"inParentIds"` // 由 ParentID 推导
}
```

中转参数不在 proto 请求中暴露，由 service 层设置后传给 repository 用于数据库查询。

## 响应 DTO

通常不单独定义响应 DTO，而是直接使用 entity 结构体 + `value.Transform` 转换为 proto 响应。

对于列表查询，repository 返回 `([]*entity.Xxx, uint32, error)`，第二个值为总数：

```go
// repository 接口
type Example interface {
    ListExample(ctx core.Context, req *types.ListExampleRequest) ([]*entity.Example, uint32, error)
}

// service 层
func (s *Example) ListExample(ctx core.Context, req *types.ListExampleRequest) ([]*entity.Example, uint32, error) {
    list, total, err := s.repo.ListExample(ctx, req)
    if err != nil {
        ctx.Logger().Warnw("msg", "list example error", "err", err.Error())
        return nil, 0, errors.ListError()
    }
    return list, total, nil
}

// app 层
func (h *Example) ListExample(c context.Context, req *example.ListExampleRequest) (*example.ListExampleResponse, error) {
    var (
        ctx = core.MustContext(c)
        in  types.ListExampleRequest
    )
    if err := value.Transform(req, &in); err != nil {
        return nil, errors.TransformError()
    }
    list, total, err := h.srv.ListExample(ctx, &in)
    if err != nil {
        return nil, err
    }
    reply := &example.ListExampleResponse{Total: total}
    if err := value.Transform(list, &reply.List); err != nil {
        return nil, errors.TransformError()
    }
    return reply, nil
}
```

## 搜索结果 DTO

对于需要附加计算字段（如统计值）的查询，返回带额外字段的结果 DTO：

```go
// ExampleResult 示例查询结果（带附加字段），定义在 entity 包内
type ExampleResult struct {
    ID    uint32 `json:"id"    gorm:"column:id"`
    Name  string `json:"name"  gorm:"column:name"`
    Total uint32 `json:"total" gorm:"column:total"` // 聚合计数
}
```
