# 07 · file —— 业务文件条目

> **本册范围**：`file` 表的 entity / 仓储接口、结构不变式、四条诞生路径、`user_id` 的落地理由、命名策略。
> **依赖**：01-基础设施、05-object、06-upload。**上游**：`.prd/SQL设计.md` 第 6 节。
> **下游**：09-回收与后台任务（删目录回收的 `A` 集合来自本表）。

---

## 1. 表语义

一行 = 一个业务文件条目：某个 app 的某个目录下、有名字的一个入口。物理字节在 `object`，本表只存归属与位置。**文件的身份是行 id**，不是内容、也不是名字——同一份内容可以出现在任意多个位置，重传和重名都不产生冲突。

```go
// repository/file.go
type File interface {
	// GetFile 按主键查询
	GetFile(ctx core.Context, id uint64) (*entity.File, error)
	// GetFileByUploadId 完成事务的幂等键（重复 Complete / GC 重放落这里）
	GetFileByUploadId(ctx core.Context, uploadID string) (*entity.File, error)
	// ListFileByDirectory 目录内分页（走 idx_file_app_directory）
	ListFileByDirectory(ctx core.Context, app string, directoryID uint64, req *types.PageRequest) ([]*entity.File, uint32, error)
	// ListFileByUser "我的文件"分页（走 idx_file_app_user）
	ListFileByUser(ctx core.Context, app string, userID uint64, req *types.PageRequest) ([]*entity.File, uint32, error)
	// CreateFile 建行
	CreateFile(ctx core.Context, ent *entity.File) (uint64, error)
	// UpdateFile 改名 / 移动（显式列更新）
	UpdateFile(ctx core.Context, ent *entity.File, columns ...string) error
	// DeleteFile 删单行
	DeleteFile(ctx core.Context, id uint64) error
	// CollectObjectIDsInSubtree 回收第 1 步：删之前收集候选 A（join directory）
	CollectObjectIDsInSubtree(ctx core.Context, app string, id uint64, path string) ([]uint64, error)
	// DeleteFileInSubtree 回收第 2 步：一条 join-delete 删整棵子树的 file 行
	DeleteFileInSubtree(ctx core.Context, app string, id uint64, path string) error
	// FilterReferencedObjectIDs 回收第 3 步：返回 A ∩ B（按 500 分批）
	FilterReferencedObjectIDs(ctx core.Context, ids []uint64) ([]uint64, error)
}
```

`entity.File` 的 `UploadId *string`：秒传 / 直传为 NULL。

后三个方法是**回收专用**，语义与调用顺序见 09-回收与后台任务。

## 2. 结构不变式：`file.app ≡ directory.app`

file 行只能生在它自己 app 的目录里。这个值不独立存在，由完成事务 `SELECT app FROM directory WHERE id=?` 派生（会话只带 `directory_id`，不带 app），所以不存在"file 的 app 与目录的 app 不一致"的形态；这次查询同时承担"目录仍存在"的检查。

## 3. `user_id` 必须落库、不能靠目录反推

目录是共享的，同一个目录里放的是多个用户的文件，"这份文件是谁传的"是目录表达不了的事实。三个消费方：①"我的文件"列表；②上传会话鉴权；③追责。

## 4. 行只从四条路径诞生

| 路径 | 诞生时机 | `upload_id` |
|---|---|---|
| 秒传 | 预上传命中 object，建行即返回 URL，不建会话 | NULL |
| 多段直传（含单片） | 所有非秒传上传建会话、走 `upload` 表 | 厂商 uploadId |
| 分片完成 | 客户端 `CompleteUpload` 的完成事务内 | 厂商 uploadId |
| GC 补合 | 会话过期但 `ListParts` 判据齐，完成事务内 | 厂商 uploadId |

**`user_id` 的来源**：秒传（不经会话）取自**请求上下文**；分片/单片完成（建会话，`chunkCount≥1`）取请求上下文，并**与会话的 `user_id` 比对、不一致拒绝**；GC 补合取**会话的 `user_id`**。

## 5. file 行的出生时刻 = 会话行的死亡时刻

两者在同一事务内（T1，见 06-upload §7.2），所以 `upload` 表"只有活跃会话"不留残渣；反过来，任何 file 行都对应一次明确的产出事件，没有"半成品行"需要读接口自己去挡。

## 6. 同目录同名允许

本表对名字没有任何唯一约束——上传的语义是"产生一个新条目"而不是"抢占一个名字"，加唯一会迫使定义"同名是覆盖还是拒绝"，两者都会让并发上传出现不可预期的失败。重名如何展示由接入方决定。

## 7. 删除

单文件删除与删目录的完整回收流程（含 `A − B` 集合代数、执行顺序、并发保守语义）在 **09-回收与后台任务**。

**读路径不受删除影响**：URL 由 `object` 的 `(store_keyword, md5, ext)` 拼出、对象公网可读，删除只影响条目本身。
