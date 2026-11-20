# 代码风格规则

> 参考 limes-cloud/manager 和 kratosx 的代码风格。

## 函数参数

除 `ctx` 外，入参**超过 3 个（即 4 个及以上）必须收敛成一个结构体**传递，不得平铺一长串参数。

- `ctx core.Context` / `context.Context` 不计入限额，且永远是第一个参数、**不进**结构体。
- 返回值不受此限制。
- 结构体定义位置：
  - 跨层的纯数据请求 / 参数（**不含 `entity`**）放 `internal/types/`（命名 `XxxRequest` / `XxxParams`）。
  - 携带 `entity` 的参数、或仅包内使用的私有 helper 参数，放**该函数所在的包**（`types` 不得依赖 `entity`，见 architecture.md，故这类只能就地定义）。
  - `internal/pkg/**` 工具函数的参数结构体放**该工具函数所在目录**。
  - infra 仓储 / 客户端方法的入参收敛为 `XxxInput`，随接口定义在 `domain/repository/`。
- **构造函数 `NewXxx` 的依赖注入（DI）豁免**：注入多个 repository / service 时仍按位置传入，不收敛成 Deps 结构体。
- 字段语义自解释；只为凑数硬塞无关字段是反模式——参数本该高内聚，塞不进一个合理结构体说明函数职责过宽，应先拆函数。

## 命名规范

| 类型 | 规范 | 示例 |
|---|---|---|
| 文件名 | 全小写，下划线分隔 | `app_config.go`, `store.go` |
| 包名 | 全小写，无下划线，与目录名一致 | `entity`, `repository`, `dbs` |
| 结构体 | 大驼峰，与业务领域对应 | `AppConfig`, `Store`, `Directory` |
| 接口 | 大驼峰，与领域名词对应 | `Store`, `File`, `Directory` |
| 方法 | 大驼峰，动词开头 | `GetStore()`, `CreateFile()`, `ListDirectory()` |
| 私有方法 | 小驼峰 | `appendChildren()`, `buildStoreClient()` |
| 常量 | 大驼峰或全大写 | `StatusEnabled`, `defaultBatchSize` |
| 错误函数 | 大驼峰 + Error 后缀 | `ParamsError()`, `GetError()`, `CreateError()` |
| 错误判断函数 | Is 前缀 | `IsParamsError()`, `IsNotFoundError()` |

## 指针字段

**判据：字段的合法取值是否可能是 Go 零值（`0` / `""` / `false`）——是，就必须声明为指针。**

这与「DB 列是否可空」是两个独立概念，别混为一谈：

- `status`：DB 里 `NOT NULL`（`0`=禁用、`1`=启用都是合法值），但 Go 把 `0` 当零值。`Updates(struct)` 会跳过零值字段，用值类型 `uint8` 时「把 status 改成 0」会被静默丢弃——所以**即便 DB 非空，只要合法值含 `0`，也必须用指针** `*uint8`。
- `description`：DB 可空（`NULL`=无备注），Go 用指针 `*string` 表达「未设置」。
- `id` / `name`：必填且**永不为零值**，用值类型；给它们套指针是反模式。

```go
// 正确：合法值可能是 Go 零值 → 指针
type Store struct {
    Status      *uint8  `gorm:"column:status"`       // 0=禁用 也是合法值，DB NOT NULL 仍要指针
    Description *string `gorm:"column:description"`   // DB 可空，NULL=无
    CdnAuthKey  *string `gorm:"column:cdn_auth_key"`  // DB 可空
}

// 错误：必填、永不为零值的字段不要用指针
type Store struct {
    Id      *uint32 `gorm:"column:id"`      // 主键永不为零，不要用指针
    Keyword *string `gorm:"column:keyword"` // 必填且非空，不要用指针
}
```

指针为什么是必须的（而不是风格偏好）：`Updates(struct)` 跳过零值、`optional` 区分「未传」与「显式设零值」，都依赖指针承载「存在性」。详见 infra-layer.md「更新用实体、零值列用指针」与 proto-and-api.md 规则 11。

指针操作使用 `value.Pointer()` 和 `value.Value()`：

```go
// 创建指针
status := value.Pointer(uint8(1))
name := value.Pointer("test")

// 解引用（nil 安全）
isEnabled := value.Value(entity.Status)       // 0 if nil
desc := value.Value(entity.Description)        // "" if nil
```

条件查询中使用指针判断：

```go
if req.Status != nil {
    db = db.Where("status = ?", *req.Status)
}
```

## 错误处理

### 原则

1. **service 层**：捕获底层 error，日志记录，返回 proto 生成的错误函数
2. **app 层**：直接透传 service 返回的错误
3. **infra 层**：返回原生 error，不做包装

### 正确示例

```go
// service 层
func (u *Example) GetExample(ctx core.Context, id uint32) (*entity.Example, error) {
    res, err := u.repo.GetExample(ctx, id)
    if err != nil {
        ctx.Logger().Warnw("msg", "get example error", "err", err.Error())
        return nil, errors.GetError()
    }
    return res, nil
}

// app 层
func (h *Example) GetExample(c context.Context, req *example.GetExampleRequest) (*example.GetExampleResponse, error) {
    ctx := core.MustContext(c)
    ent, err := h.srv.GetExample(ctx, req.Id)
    if err != nil {
        return nil, err
    }
    // ... 转换响应
}
```

### 错误示例

```go
// 错误 — 直接暴露底层错误
return nil, fmt.Errorf("database query failed: %w", err)

// 错误 — 在 app 层重新包装错误
return nil, errors.New(500, "GET_ERROR", err.Error())
```

## 构造函数模式

使用 `NewXxx()` 手动依赖注入，不使用 DI 容器：

```go
// service 层
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

// infra 层 — 单例模式
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

## 日志规范

使用结构化日志，通过 `ctx.Logger()` 获取：

```go
// 警告级别 — 可预期的业务错误
ctx.Logger().Warnw("msg", "get example error", "err", err.Error())

// 错误级别 — 不可预期的系统错误
ctx.Logger().Errorw("msg", "database connection failed", "err", err.Error())

// 键名规范
// "msg" — 操作描述
// "err" — 错误信息
// "key" — 业务键名
// "id"  — 资源ID
```

## 注释规范

**总纲：注释只解释"这段代码在做什么"，不解释"为什么这样设计"。**

能用一行说清的不要写多行；代码本身已经说清的就不写。
设计论证、决策过程、"前提 / 代价 / 已显式接受"这类内容属于 `.prd/`，不属于代码注释。

### 函数注释

一行，形如 `// 方法名 做什么`：

```go
// GetAppConfig 按 app 读取应用配置
// CreateAppConfig 创建应用配置
// LoadAll 全量加载 store 客户端
```

方法名后面最多跟一个**事实性**限定（做什么 / 不做什么），不跟理由：

- ✅ `// UpdateDirectory 更新目录（改名 / 移动 / 改策略）`
- ✅ `// LoadAll 全量加载 store 客户端（含停用行）`
- ❌ `// CreateAppConfig 新建（不删缓存——删除触发点是闭集 {Update}，Create 不在其中）`

### 类型与字段注释

- 类型：一行说是什么，如 `// AppConfig 应用存储配置`。
- entity / 配置字段：只写**值域事实**——取值、单位、NULL 语义、是否只读。

```go
Provider  string   // 厂商标识：aws|bos|oss|cos|obs|minio|uss
Status    uint8    // 状态：1启用 0停用
CdnAuthKey *string // CDN 鉴权 key；NULL = 未配置
MaxSize   uint32   // 单文件上限（MB）
SK        string   // 访问密钥；永不进响应
```

不写"为什么"——`全库引用用它而非 id`、`创建后不可改` 这类推理一律去掉。

### 函数体内注释

只标**非显然的步骤**与**会造成静默错误的技术陷阱**，各一句话：

```go
// 前缀谓词必须转义：目录 图_片 会命中 /图A片/...，误删整棵树
pattern := pattern.PrefixPattern(path)
```

以下一律不写：

- 设计论证、决策记录、`见 0X-xxx §N` 这类文档引用
- 对业务策略的解释（`status=0 是上传闸门`、`第二次删必须晚于 commit` …）
- 多行注释块——一句话说得完的不要拆成块
