# Infra 层规则

> 完全参照 limes-cloud/manager 的 dbs 风格，使用 kratosx 封装的 GORM（`ctx.DB()` 返回 `*gorm.DB`）。

## dbs/ — 数据仓储实现

### 单例模式

```go
type Example struct{}

var (
    exampleIns  *Example
    exampleOnce sync.Once
)

func NewExample() *Example {
    exampleOnce.Do(func() {
        exampleIns = &Example{}
    })
    return exampleIns
}
```

---

## 方法命名规范

方法名必须与 `domain/repository/` 接口定义**完全一致**，不得自行重命名。本项目接口的主要命名模式：

| 操作 | 本项目命名示例 |
|---|---|
| 创建单条（主键自增） | `CreateExample` |
| 根据 ID 查单条 | `GetExample` / `GetByID` |
| 根据其他条件查单条 | `GetByName` |
| 无分页列表 | `ListExample` / `ListAll` |
| 批量硬删除 | `DeleteExampleByIDs` |
| 写入或更新 | `Upsert` |

**注意：** 接口定义是权威依据，infra 层的签名必须完全匹配，包括参数类型和返回值类型。

---

## CRUD 操作规范

### 主键自增

本项目主键为 **uint32，由 MySQL 自增生成**（entity 嵌入 `model.BaseModel`，`Id` 带 `primaryKey;autoIncrement`），**不在 dbs 层手动赋值**。创建方法返回 `(uint32, error)`，回传自增后的 `ent.Id`：

```go
func (r Example) CreateExample(ctx core.Context, ent *entity.Example) (uint32, error) {
    if err := ctx.DB().Create(ent).Error; err != nil {
        return 0, err
    }
    return ent.Id, nil
}
```

`Id` / `CreatedAt` / `UpdatedAt` 由 `model.BaseModel` 承载，GORM 按约定自动填充，无需手动赋值。

---

### Get — 查询单条

```go
// GetByID 根据 ID 获取实体
func (r *Example) GetByID(ctx core.Context, id uint32) (*entity.Example, error) {
    var e entity.Example
    err := ctx.DB().First(&e, "id = ?", id).Error
    if err != nil {
        return nil, err
    }
    return &e, nil
}

// GetByName 根据名称精确查询
func (r *Example) GetByName(ctx core.Context, name string) (*entity.Example, error) {
    var e entity.Example
    err := ctx.DB().Where("name = ?", name).First(&e).Error
    if err != nil {
        return nil, err
    }
    return &e, nil
}
```

---

### List — 查询列表

```go
// ListAll 获取全部实体（按创建时间正序）
func (r *Example) ListAll(ctx core.Context) ([]*entity.Example, error) {
    var list []*entity.Example
    err := ctx.DB().Order("created_at ASC").Find(&list).Error
    return list, err
}

// ListTopN 获取 top-N 实体（按创建时间倒序）
func (r *Example) ListTopN(ctx core.Context, limit int) ([]*entity.Example, error) {
    var list []*entity.Example
    err := ctx.DB().Order("created_at DESC").Limit(limit).Find(&list).Error
    return list, err
}
```

---

### Update — 更新与软删除

```go
// Disable 软删除/停用（设置 status=0，status 为 *uint8 指针列）
func (r *Example) Disable(ctx core.Context, ids []uint32) error {
    if len(ids) == 0 {
        return nil
    }
    return ctx.DB().Model(&entity.Example{}).Where("id IN ?", ids).
        Select("status").Updates(&entity.Example{Status: value.Pointer(uint8(0))}).Error
}

// IncrCount 累加 count（非零值字段用 UpdateColumn + Expr）
func (r *Example) IncrCount(ctx core.Context, ids []uint32) error {
    if len(ids) == 0 {
        return nil
    }
    return ctx.DB().Model(&entity.Example{}).Where("id IN ?", ids).
        UpdateColumn("count", gorm.Expr("count + 1")).Error
}

// MarkDone 指针位 + Select 指定列（不用 map）。entity.Example 的 Done 为 *bool
func (r *Example) MarkDone(ctx core.Context, id uint32) error {
    ent := &entity.Example{Done: value.Pointer(true)}
    return ctx.DB().Model(&entity.Example{}).Where("id = ?", id).Select("done").Updates(ent).Error
}
```

**注意：更新一律用实体，不用 map，也不用 `Save`**
- 更新走 `Updates(ent)`（实体）。要限定更新范围时配 `Select("col_a", "col_b")` 显式列出列，**不要**改用 `Updates(map[string]any{...})`。
- **禁止 `Save(ent)`**：`Save` 会把整行所有列（含 `0`/`""`/`false` 零值）无条件写回，等于强制全列覆盖，既绕过"零值跳过"保护、又在并发下放大 read-modify-write 竞争。需要写某零值列时，把该列设为指针用 `Updates(ent)` 写（见下）。
- **凡"允许写入零值"的列，必须在 entity 里定义为指针**——状态位、开关（`bool`）、可留空的描述 / 简介（`string`）、允许设成 `0` 的数值，以及任何可空列。`Updates(struct)` 会跳过非指针的零值（`0` / `""` / `false`），这些列会被**静默丢弃、永远写不进去**。指针配 `value.Pointer()` / `value.Value()` 读写（见 code-style.md「指针字段」）。
- 判据：只要"合法值里包含零值"（能被设成 `0` / `""` / `false`），该列就必须是指针；仅当列必填且永不为零（如 `name`）才允许用值类型。
- 单列原子自增 / 表达式更新用 `UpdateColumn` + `gorm.Expr`，避免触发 GORM 钩子。

**选择更新方式（本服务多次返工的根因，按此判定，别自己发明第三种）**

1. **默认：直接 `Updates(ent)`**。handler 把 proto optional 指针透传给 service，service 构造稀疏实体（只落传入字段），零值可写列在 entity 用指针（非 nil 即写、nil 跳过），**不读旧行**。app_config / directory / file 都走这条。
2. **只有需要"旧行 + 本次改动"合并后的完整行做跨字段校验时**，才读当前行 → 叠加 → 用 `Updates(ent)` 写回（零值可写列在 entity 用指针，Updates 才不会跳过）。唯一现例：store 的 `CheckCDN` 要 endpoint / public_base_url / cdn_auth_type 同时在场。**仍然禁用 `Save`**——即便已读到完整行也用 `Updates(ent)`。没有这种跨字段校验就别读旧行再 overlay，那是过度设计。
3. **不要用 `Select(columns)` 清单来表达"局部更新"**：没有合法零值的列（name/directory_id）靠 `Updates(ent)` 自动跳过零值即可；有合法零值的列用指针。`Select` 只在确有「必须强制写某零值列」的特殊需要时才用。
4. WHERE 用**真实唯一键**：app_config 以 `app` 不是 `id`；用无意义键更新是 bug。
5. **每次 update 都刷 `updated_by`**；子树 / 多行的 raw SQL 同样要带 `updated_at`、`updated_by`（见 `moveTreeSQL`）。
6. 只顶时间戳（滑动过期）用 `UpdateColumn("updated_at", time.Now().Unix())`，**禁止**空 `Updates(map[string]any{})` 靠钩子"碰巧"触发——钩子一旦失效就变静默 no-op。
7. 一个操作里**关联的多次写**（如移子树重写 path + 改节点 name）必须包进同一 `ctx.Transaction`，否则半成功留下不一致状态。

**repo 读返回权威值，脱敏只在响应边界**

`GetXxx` / `ListXxx` 必须返回**明文权威值**（含 sk 等凭证），脱敏一律在 app 层 `toXxx` 响应转换里做。**绝不在 repo 读里脱敏**——否则上面第 2 条的 read-merge-`Save` 会把脱敏值当真值写回库，瞬间毁掉凭证（接口注释也不要写成"返回掩码"误导后人）。

---

### Delete — 删除记录

```go
// DeleteExampleByIDs 批量硬删除
func (r *Example) DeleteExampleByIDs(ctx core.Context, ids []uint32) error {
    if len(ids) == 0 {
        return nil
    }
    return ctx.DB().Where("id IN ?", ids).Delete(&entity.Example{}).Error
}
```

---

### Upsert — 写入或更新

使用 `clause.OnConflict`，`Columns` 需与数据库唯一约束列完全对应：

```go
// Upsert 写入或更新（主键 uint32 自增，无需手动赋值）
func (r *Example) Upsert(ctx core.Context, ent *entity.Example) error {
    return ctx.DB().Clauses(clause.OnConflict{
        Columns:   []clause.Column{{Name: "name"}},
        DoUpdates: clause.AssignmentColumns([]string{"status", "remark"}),
    }).Create(ent).Error
}
```

---

## 事务操作

统一使用 `ctx.Transaction(func(ctx core.Context) error { ... })` 闭包，不使用原生 `Begin/Commit`：

```go
err = ctx.Transaction(func(txCtx core.Context) error {
    if err := t.childRepo.DeleteByIDs(txCtx, ids); err != nil {
        return err
    }
    return t.parentRepo.MarkDone(txCtx, parentID)
})
```

---

## Redis 缓存

### 手动 Redis 操作（详情缓存模式）

对于简单的 JSON 序列化缓存，直接操作 `ctx.Redis()`：

```go
const (
    detailCacheTTL    = 10 * time.Minute
    detailCachePrefix = "example:detail:"
)

// GetDetail 获取实体详情（Redis 缓存优先，miss 则查 DB 并回填）
func (r *Example) GetDetail(ctx core.Context, id uint32) (*entity.Example, error) {
    key := detailCachePrefix + strconv.FormatUint(id, 10)

    // 尝试读缓存
    val, err := ctx.Redis().Get(ctx.Ctx(), key).Result()
    if err == nil && val != "" {
        var cached entity.Example
        if err := json.Unmarshal([]byte(val), &cached); err == nil {
            return &cached, nil
        }
    }

    // cache miss：查 DB
    ent, err := r.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // 回填缓存
    if data, err := json.Marshal(ent); err == nil {
        ctx.Redis().Set(ctx.Ctx(), key, data, detailCacheTTL)
    }
    return ent, nil
}

// InvalidateDetailCache 使缓存失效
func (r *Example) InvalidateDetailCache(ctx core.Context, id uint32) error {
    return ctx.Redis().Del(ctx.Ctx(), detailCachePrefix+strconv.FormatUint(id, 10)).Err()
}
```

### Redis key 命名规则

```
example:detail:{id}          // 实体详情缓存
```

---

## 高级查询

### 原则：GORM 链式 API 优先

**所有查询统一使用 GORM 链式 API**，复杂条件通过 `gorm.Expr(...)` / `Where` 字符串表达，而不是裸写 `ctx.DB().Raw().Scan()`。

- 复杂对称条件（`OR`）直接写入 `Where` 字符串
- 读取完整 struct 后在 Go 层计算布尔结果

---

### 复杂 OR 条件与 Go 层计算

```go
// QueryByEither 查询名称或备注命中关键字的实体（OR 条件保留在一个 Where）
func (r *Example) QueryByEither(ctx core.Context, keyword string) ([]*entity.Example, error) {
    var list []*entity.Example
    err := ctx.DB().Where(
        "(name = ? OR remark = ?)", keyword, keyword).
        Where("status = ?", 1).Find(&list).Error
    return list, err
}

// IsEnabled 读取完整 struct，在 Go 层计算布尔结果（无需 Raw SQL）
func (r *Example) IsEnabled(ctx core.Context, id uint32) (bool, error) {
    var e entity.Example
    if err := ctx.DB().First(&e, "id = ?", id).Error; err != nil {
        return false, err
    }
    return value.Value(e.Status) == 1, nil
}
```
