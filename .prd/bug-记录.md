# Bug 记录

> 代码评审发现的实现缺陷，与 `设计方案/` 的差异。每条含：位置、证据、后果、修复。
> 评审时间：2026-10-14。评审范围：pkg / domain·service / infra·dbs / infra·store。`go build ./...` 通过。

---

## BUG-1（严重 · 静默误删数据）删目录回收时 file 子树查询未转义 LIKE

**位置**
- `internal/infra/dbs/file.go:98`（`CollectObjectIDsInSubtree`）
- `internal/infra/dbs/file.go:109`（`DeleteFileInSubtree`）

**证据**

两处把 `path+"/%"` 直接当 LIKE 模式，既没走 `pkg/pattern.PrefixPattern`，也没带 `ESCAPE '\\'`：

```go
// file.go:98
.Where("directory.app = ? AND (directory.id = ? OR directory.path LIKE ?)", app, id, path+"/%")
// file.go:109
.Where("app = ? AND (id = ? OR path LIKE ?)", app, id, path+"/%")
```

而**同一次删除**里 directory 表的删除是转义过的（`internal/infra/dbs/directory.go:20/121`，用 `pattern.PrefixPattern(path)` + `ESCAPE '\\'`）。两者子树范围不一致，直接违反 `设计方案/04-directory.md §5`「子树谓词一律走同一个转义函数」，以及项目设立 `pkg/pattern` 的唯一目的。

**后果**（`设计方案/04-directory.md §6` 明确允许目录名含 `%` / `_` / `\`）
- 删名为 `/a%` 的目录 → file 侧模式 `/a%/%`，`%` 当通配符 → 命中 `/a`、`/abc`、`/anything…` 下**所有 file 行**，`DeleteFileInSubtree` 全删、`CollectObjectIDsInSubtree` 把它们的对象收进 A，提交后 `reclaimObjects` 可能真删桶里字节。
- directory 树删除是转义过的、范围更窄 → 兄弟目录行**仍在**，但目录下 file 行已被删 → 悬空目录 + 误删文件 + 误回收对象，排障看不出异常。
- 常见的 `/user_avatars`、`/2024_01` 等带 `_` 的目录同样会把 `/userXavatars` 之类兄弟子树扫进来。

**修复**

两处改用 `pattern.PrefixPattern(path)` 构造模式并补 `ESCAPE '\\'`，与 `directory.go` 完全对齐。注意 `CollectObjectIDsInSubtree` 是 `JOIN directory` 后对 `directory.path` 过滤、`DeleteFileInSubtree` 是子查询对 `directory.path` 过滤，两处的 LIKE 都要改。

---

## BUG-2（功能失效）目录 `accept` 无法被更新为 `""`（禁止所有类型）

**位置**
- `internal/domain/service/directory.go:134`（无变更短路）
- `internal/infra/dbs/directory.go:95`（`Updates(ent)` 结构体零值跳过）

**证据**

`设计方案/04-directory.md §9` 规定 `accept=''` 是合法且有意义的取值（不允许任何类型）。但 `UpdateDirectory` 永远写不进这个值，两道关各挡一次：

```go
// service/directory.go:134 —— “只改 accept 为空” 被判成 “无变更” 直接返回 nil
if node.Name == "" && node.Accept == "" && node.MaxSize == 0 {
    return nil
}
// infra/dbs/directory.go:95 —— 即便搭配其它字段，Updates(struct) 跳过零值 ""
return ctx.DB().Model(&entity.Directory{}).
    Where("id = ? AND app = ?", ent.Id, ent.App).
    Updates(ent).Error
```

`Updates(ent)` 用结构体会跳过零值，正是 `.claude/rules/infra-layer.md`、`code-style.md` 反复警告的 GORM 陷阱；且 repo 实现把设计接口里的「显式列更新」能力丢了（`repository/directory.go:23` 的签名已无 `columns ...string`）。

**后果**：用户想把某目录锁成「禁止上传任何类型」，接口返回成功、实际什么都没改。`max_size` 不受影响（0 本就被 `ValidateMaxSize` 判非法，不是合法目标值）。

**修复**（按新规范：零值字段全链路指针 + 直接 `Updates(ent)`，见 proto-and-api.md 规则 11、types-dto.md、infra-layer.md）
- proto `UpdateDirectoryRequest` 的 `path` / `accept` / `max_size` 改 `optional`（生成 Go 指针），重新生成代码。
- app handler 直接透传指针（`d.Accept = req.Accept`），不再用 `v != ""` 判存在性。
- `entity.Directory.Accept` 改 `*string`（`""` 是合法值 ⇒ 零值可写列必须指针）；`max_size` 仍 `uint32`（`0` 非法、永不作为合法目标值）。
- `infra/dbs/directory.go` 的 `UpdateDirectory` **保持直接 `Updates(ent)`**（不加 `columns` / `Select`）——指针列非 nil 即写入，GORM 自动处理零值。
- service 的「无变更」短路改成 `node.Accept == nil`（而非 `== ""`）。

---

## BUG-3（设计一致性）app_config 更新应以 app（唯一键）为 WHERE，`id` 冗余无意义

**位置**
- `internal/app/app_config/app_config.go` `UpdateAppConfig`（`ent.Id = uint64(req.GetId())`）
- `internal/infra/dbs/app_config.go` `UpdateAppConfig`（`Where("id = ?", ent.Id)`）
- `api/app_config/proto/app_config.proto` `UpdateAppConfigRequest.id`

**问题**：`app_config` 一个 app 只配置一次（`app` 唯一索引），更新请求里已带 `app`。再带一个 `id` 并以 `id` 作 WHERE 是冗余——`id` 字段、`ent.Id = req.GetId()` 赋值都没有意义，还给了"用 id 指到另一行"的错误空间。

**修复**
- proto `UpdateAppConfigRequest` 删除 `id`，`UpdateAppConfigResponse` 的 `id` 改 `app`；重新生成。
- handler 去掉 `ent.Id = ...`，响应返回 `app`。
- dbs 改 `Where("app = ?", ent.App)`（缓存删除本就按 app，无需改）。

**附带扫描结论**：`store` 的更新仍以 `id` 为 WHERE 是**合理的**——`keyword` 不可改且不在更新请求里，`id` 是唯一可用标识；`directory` 以 `(id, app)` 更新也合理（唯一键 `(app, path)` 中的 path 会在移动时变）。list 过滤里的 `if v := req.GetX(); v != "" {...}` 是"空=不过滤"的合法语义，保留。

---

## 规范符合性整改（对照 `.claude/rules/`）

> 以下为代码与规范的偏差。R3 的规范侧已更新（infra-layer.md 改为「更新用实体 + 零值列指针」，并连带 proto-and-api.md 规则 11、types-dto.md），代码按新规范整改。

### R1（普遍）注释引用设计文档章节 / 决策日期 — 违反 code-style.md 注释规范

规范明令注释不写 `见 0X-xxx §N` 文档引用、设计论证、决策记录。但 18+ 处注释带 `§4.2`/`§6`/`§7.3`、`见 09-...`、`2026-10-13 定`，集中在：
- `internal/domain/service/upload.go`（8 处）、`recycle.go`（4 处）、`file.go:112/153`
- `internal/types/upload.go:20`、`internal/initiator/recycle/recycle.go:1`

**整改**：改成事实描述，去掉章节号与日期。

### R2 service 层吞底层 error 不记日志 — 违反 code-style.md 错误处理 + 日志规范

`upload.go` / `file.go` 两个 service 几乎无 `ctx.Logger()`，返回 `GetError/SystemError/CreateError` 前不记底层 DB/S3 错误（`directory.go`/`store.go`/`app_config.go` 均已记）。线上 500 查不到根因。

**整改**：返回系统类错误前补 `ctx.Logger().Warnw("msg", ..., "err", err.Error())`。

### R3 directory.go 更新用 `Updates(struct)` — 已随规范更新

`infra/dbs/directory.go` 曾是唯一用 `Updates(ent)` 结构体的（即 BUG-2 根因）。规范侧已更新为「更新用实体，零值可写列必须指针」，代码随 BUG-2 一并整改。

### R4（轻微）service 层 import gorm — 擦边 domain-layer.md「service 不得 import DB」

`internal/domain/service/errors.go` import `gorm.io/gorm` 与 `kratosx/.../gormtranserror` 做 `isNotFound`/`isDuplicate`。仅用常量不做 DB 操作，属灰色；可后续把错误分类下沉到 infra/pkg。**本轮不改**，仅登记。

---

## 已复核正确、无需改动

- `pkg/chunk`：`Size` 的 uint32 上界与 `MaxTotalSize`（≈42.94TB）推导、空文件/超大拒绝、`Digest` 分片摘要的摘要。
- `infra/store/s3.go`：`ListParts` 的 `IsTruncated` 全量分页。
- `infra/dbs/app_config.go`：延迟双删（map 更新 + `ctx.Clone()` 起 goroutine + 不包事务）、正负缓存同 key、重建锁双检 + token 比对。
- object / upload 撞唯一键重查复用赢家、`DeleteObjectIfUnreferenced` 的 `NOT EXISTS` CAS。
