# 04 · directory —— 物化路径目录树

> **本册范围**：`directory` 表的 entity / 仓储接口、三个 CRUD 语义、`LIKE` 转义、限深与防环、自动建目录、继承语义。
> **依赖**：01-基础设施（`pkg/pattern`、gorm 陷阱）、03-app_config（默认值来源）。**上游**：`.prd/SQL设计.md` 第 3 节。

---

## 1. 仓储接口

```go
// repository/directory.go
type Directory interface {
	// GetDirectory 按主键查询（必须带 app 限定）
	GetDirectory(ctx core.Context, app string, id uint64) (*entity.Directory, error)
	// GetDirectoryByPath 按路径查询
	GetDirectoryByPath(ctx core.Context, app, path string) (*entity.Directory, error)
	// ListDirectory 全量取回某 app 的目录（内存建树用）
	ListDirectory(ctx core.Context, app string) ([]*entity.Directory, error)
	// ListDirectoryInPaths 按路径集合批量查询（自动建目录"先定位最深已存在层"用）
	ListDirectoryInPaths(ctx core.Context, app string, paths []string) ([]*entity.Directory, error)
	// CreateDirectoryIgnore 幂等新建，返回最终行（INSERT IGNORE 后重查）
	CreateDirectoryIgnore(ctx core.Context, ent *entity.Directory) (*entity.Directory, error)
	// UpdateDirectory 更新（显式列更新，防 gorm 零值跳过）
	UpdateDirectory(ctx core.Context, ent *entity.Directory, columns ...string) error
	// MoveDirectory 移动子树：一条 UPDATE 重写整棵子树 path
	MoveDirectory(ctx core.Context, app string, id uint64, newPath string) error
	// DeleteDirectoryTree 删子树：一条语句删掉自己与全部后代
	DeleteDirectoryTree(ctx core.Context, app string, id uint64, path string) error
}
```

## 2. 父目录完全由 `path` 推出

`/图片/头像` 的父级就是去掉最后一段的 `/图片`，根的父级是 `""`。**不存 `parent_id`**：它只服务"列出直接子目录"这一种查询，而那条查询可以由 `path` 前缀 range 或内存树完成；存它反而带来一个真实的一致性风险——移目录时只重写子树 `path`，`parent_id` 会漂移。

## 3. 三个 CRUD 语义

| 操作 | 实现 |
|---|---|
| **删目录 = 删子树** | `WHERE app=? AND (id=? OR path LIKE ?)` 一条语句，第二个参数 = `PrefixPattern(path)` |
| **移目录 = 重写子树 path** | 一条 `UPDATE ... SET path = CONCAT(新path, SUBSTRING(path, CHAR_LENGTH(旧path)+1)) WHERE app=? AND (id=? OR path LIKE ?)` |
| **按 id 取祖先链 / 子树** | 祖先链用 `path` 拆前缀集合再 `path IN (...)`；子树用前缀 range。都不需要递归 CTE |

> 删目录会连带回收对象，回收的集合代数与执行顺序见 **09-回收与后台任务**。

## 4. 移目录的两条硬约束

1. **必须用 `CHAR_LENGTH`，不能用 `LENGTH`**。`SUBSTRING(str, pos)` 的 `pos` 对非二进制串是**字符**位置，而 `LENGTH()` 返回**字节**数。`/图片` 是 3 个字符 7 个字节，于是 `LENGTH(旧path)+1 = 8`，`SUBSTRING('/图片/头像', 8)` 从第 8 个字符开始取——串只有 6 个字符，**直接返回空串**，整棵子树的 `path` 被截成裸的新前缀。这是**只在含多字节字符的路径上才发作的静默错误**。两个函数都是字符语义时才配对。
2. **目标 `path` 已存在则拒绝**，不重试、不合并、不自动改名。自动改名会让用户以为搬到了 A、实际落在 B；合并则要先定义"同名目录怎么合、同名文件怎么办"。注意这条 UPDATE 是**整棵子树一条语句**，所以要么全成功、要么因唯一键冲突全失败，不存在"半个子树搬过去"的中间态。
3. **防环**：新父级的 `path` 是自己的 `path` 的前缀 → 拒绝。路径前缀判断一条规则挡住任意深度的环。

## 5. 子树谓词一律走同一个转义函数

```go
// pkg/pattern
// EscapeLike 依次转义 \ → \\，% → \%，_ → \_；顺序不能反
func EscapeLike(s string) string
// PrefixPattern 返回子树前缀模式：EscapeLike(path) + "/%"
func PrefixPattern(path string) string
```

```sql
WHERE app = ? AND path LIKE :pattern ESCAPE '\\'
```

**为什么必须转义**：`_` / `%` / `\` 在 `LIKE` 里是元字符——`_` 匹配任意单字符（目录 `图_片` 的前缀会命中 `/图A片/...`），`%` 匹配任意多字符，`\` 是默认转义符本身。后果是**误删 / 误移**（删目录和移目录用的是同一条前缀谓词）。**先转 `\` 再转 `%` / `_`**，顺序不能反；**`ESCAPE` 必须显式写**，不写就依赖 MySQL 的默认转义符与 `NO_BACKSLASH_ESCAPES` 之类的模式开关。

**索引不受影响**：`LIKE` 的常量前缀（转义后仍是字面量）照常让优化器取 `uidx_directory_app_path` 的范围扫描，`ESCAPE` 不破坏它。

## 6. `name` 的值域（代码校验，不写 CHECK）

**五条**，逐条都有理由：

1. **非空**，且去首尾空白后仍非空。
2. **≤ 64 字符**（`CHAR_LENGTH` 语义，一个汉字算 1 个字符）。这条不是审美约束——它是 `path` 列长（`VARCHAR(660)`）能成立的前提：`10 级 × 64 + 10 个 '/' = 650 ≤ 660`。**漏了它，"限深 10 就保证了 `path` 不溢出"这句话是空的**（`name` 无上限时 10 级可以远超 660）。
3. **不含 `/`**（它是 `path` 的分隔符，一旦允许，`path` 就不再能唯一地切回层级）。
4. **不是 `.` 或 `..`**。
5. **不含控制字符**（`< 0x20`、`0x7F`），且**不含四字节字符**（`rune > 0xFFFF`，含绝大多数 emoji）——2026-10-13 16:55 定。

**`%` / `_` / `\` 允许**——它们是合法的文件名字符，误匹配问题由查询侧转义解决，不靠禁用字符回避。

**为什么把四字节字符与控制字符并列禁掉**：两者是同一个理由的两半——**它们都会让"字符数"与"人看到的长度"脱钩**。控制字符在日志 / JSON / URL 里不可见或需要转义；四字节字符让"一个字符"在 `VARCHAR(64)` 里占 4 字节，于是 `path` 的 660 字符上限在字节侧变成 2640。**索引侧没问题**（`uidx_directory_app_path` 的 `32×4 + 660×4 + 2 = 2770` 本来就按 4 字节/字符算过），真正的理由是**运营与排障**：emoji 目录名进 URL 要 percent-encoding（一个 emoji 变 12 个 ASCII 字符）、进日志污染终端、进 `Content-Disposition` 要额外处理 RFC 5987 编码。禁掉它换来"目录名可以直接拼进 URL 与日志"，这个交换划算。

**这是"按码点禁"，不是"按语义禁"**：`rune > 0xFFFF` 是**机械可判**的边界，而"这个字符算不算 emoji"需要一张随 Unicode 版本漂移的表——按码点禁，规则不腐化。**代价要说明白**：BMP 内的符号类字符（如 `☀` U+2600、`❤` U+2764）是 3 字节，**仍在允许范围内**。要连它们一起禁就得回到"按语义判"，不划算。

**校验点**：`NameInvalidError`（10-接口契约 §4）；任一段不合法即拒绝整次请求，不做部分建目录（§7 护栏 5）。

## 7. 自动建目录（`EnsurePath`）

按路径上传是主要入口，保留"路径不存在则逐级建"，**五条护栏**：

```go
// service/directory.go
// EnsurePath 按路径逐级确保目录存在，返回最深一级的目录
func (s *Directory) EnsurePath(ctx core.Context, app, path string, cfg *entity.AppConfig) (*entity.Directory, error) {
	// 1) 限深 10，超限返回 PathTooDeep（这条同时也是 path 列长的保证）
	// 2) 前缀集合在内存按 "/" 切出（≤10 个值），一次查询定位最深已存在层
	//    —— 不每层一次 SELECT
	// 3) 从那一层往下逐层 INSERT IGNORE，撞唯一键后重查取别人刚建好的行
	//    —— 不加锁、不依赖事务串行化
	// 4) 新行的 accept / max_size 从 cfg 抄一份，抄前过值域校验（非法即拒绝整次上传）
	// 5) 每段先过 name 值域校验；任一段不合法则整次请求拒绝，不做部分建目录
}
```

- **护栏 4 的含义**：快照式是"抄一份当下**合法**的策略"，不是"抄一份什么进来都行"。`accept` 只能是 `'*'` / `''` / 后缀白名单；`max_size` 必须落在 `[1, 4294967295]`。
- **护栏 5 的含义**：半个子树落地比直接失败更难收拾。

## 8. 继承是快照式

默认值只在**建目录时**抄一次，`directory.accept` / `directory.max_size` 取当时的 `app_config.default_accept` / `default_max_size`，之后独立演化。上传校验只读 `directory` 一行，不回读 `app_config`。

**这是语义决策，不是性能决策**，三条理由：

1. **行为稳定**：目录的校验规则在其生命周期内稳定。跟随式（`NULL` = 继承）下改一次 `default_*`，会让**所有未覆写的目录立即变行为**——同一个目录昨天能传 `.zip`、今天不能，而且没有任何记录说明为什么。
2. **可审计**：目录行自己就是当时的完整策略。排障时"这个目录为什么拒了"一行就能答；跟随式要连 `app_config` 一起读，答案分散在两行、两个层级。
3. **能力不对称**：快照式能做"把某批目录冻结在当前值"的回溯脚本（全量覆写）；跟随式做不到"冻结"。

**明确接受的代价**：改 `app_config.default_*` **不会回溯**到已建目录。运营改了默认值却发现老目录行为不变，是这类设计最常见的困惑来源；真要回溯只能写一次性脚本，按 `path LIKE '前缀%'` 批量 UPDATE 子树，且必须区分"老目录本来就被人工覆写过"和"老目录在跟随默认值"——快照式**没有这个信息**，所以回溯脚本只能全量覆写，会冲掉人工设定。

**如果将来真要做跟随式**：正确形态是给 `directory` 加一个显式的 `follow_default` 布尔位，而不是把 `NULL` 当开关——布尔位把"跟随"与"显式设为不限"分开表达，语义不含糊，也不会让"`'*'` / `4294967295` = 不限"这条贯穿两级的写法出现例外。

## 9. `accept` / `max_size` 取值约定

`app_config.default_*` 与 `directory.*` 两级**逐字同规**，且**四处 DDL 都无 DEFAULT**：

| 字段 | 值域 |
|---|---|
| `accept` | `'*'` 允许全部 / `''` **不允许任何类型** / 其它为后缀白名单 |
| `max_size` | **`1 .. 4294967295`**（`0` 非法）；`4294967295`（`INT UNSIGNED` 上界 ≈ 4.29 PB，类型自身边界、非新造常量）= **不限** |

两级**都没有"未设定"这个态**。`max_size = 4294967295` 时有效上限取 **42.94 TB 物理天花板**（"不限"≠"跳过校验"，见 06-upload）。

## 10. 移动

见 §4（`CHAR_LENGTH`、目标已存在则拒绝、路径前缀防环）。
