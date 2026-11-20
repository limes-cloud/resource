# 02 · store —— 存储后端注册表

> **本册范围**：`store` 表的 entity / 仓储接口 / 实现约定、`endpoint` 归一化、`Client` 接口。
> **依赖**：01-基础设施（`pkg/provider`、唯一构造入口、gorm 陷阱）。**上游**：`.prd/SQL设计.md` 第 1 节。

---

## 1. 载体：进程内全量 registry

`infra/store/registry.go` 持有 `map[string]Client`（key = `keyword`），启动时一次性加载，**含 `status=0` 的停用行**。

`store` 的 `keyword` 不可改、行只增不减 ⇒ 进程内启动加载最省且零网络跳。这与 `app_config` 走 Redis 是刻意的不同（见 03-app_config「为什么 `store` 不走 Redis」）。

## 2. 仓储接口

```go
// repository/store.go
type Store interface {
	// ListStore 分页查询存储后端（不含 sk 明文）
	ListStore(ctx core.Context, req *types.ListStoreRequest) ([]*entity.Store, uint32, error)
	// GetStore 按主键查询存储后端（不含 sk 明文）
	GetStore(ctx core.Context, id uint64) (*entity.Store, error)
	// CreateStore 新建存储后端（写入前归一化 endpoint）
	CreateStore(ctx core.Context, ent *entity.Store) (uint64, error)
	// UpdateStore 更新存储后端（字段白名单，不含 keyword；无删除方法）
	UpdateStore(ctx core.Context, ent *entity.Store) error
	// GetClient 取进程内客户端（读路径热调用，零网络跳）
	GetClient(keyword string) (Client, bool)
	// ListClients 取全部客户端（GC / 离线核对遍历用）
	ListClients() []Client
	// LoadAll 启动自检与热加载共用：全量加载并逐个构造客户端，任一失败返回 error
	LoadAll(ctx core.Context) error
}
```

## 3. 四条实现约定

1. **`LoadAll` 必须包含 `status=0` 的行**。这是"停用"与"删除"在实现上唯一的区别点——漏加载等于该 store 名下存量对象整片 404。实现上不加 `WHERE status=1`，并在函数注释里写明原因。
2. **没有 `DeleteStore` 方法**。"行只增不减"靠**接口形状**兑现，不靠写入时校验——没有入口的操作不会被误做。
3. **更新接口的字段白名单**：`name` / `provider` / `endpoint` / `region` / `bucket` / `ak` / `sk` / `public_base_url` / `cdn_auth_type` / `cdn_auth_key` / `status` / `description`。**`keyword` 不在其中**——它既是全库引用目标，又是亿级历史行的桶指针。
4. **`Client` 接口定义在 `repository/store.go`，由 `infra/store` 实现**。它暴露：`Put` / `Get` / `Head` / `Delete` / `NewMultipart` / `ResumeMultipart` / `ListParts` / `Complete` / `Abort` / **`PresignPart`** / `GenTemporaryURL`。
   - **`PresignPart` 是"上传字节直传厂商"的落地手段**（2026-10-13 定，见 06-upload §5/§6.2，含 `chunkCount=1` 单片）：`PresignPart(uploadID string, partNumber int) (url string, err error)`，产出厂商 `UploadPart` 的预签名地址。
   - `ResumeMultipart`（按 `upload_id` 取回分片句柄）的**唯一调用方是会话召回**（06-upload §6/§7）：召回命中后靠它 `ListParts` 验活并取已传片号；`NewMultipart` 只服务新建会话。召回是只读的——不写厂商侧任何状态。
   - 它与 `GenTemporaryURL`（读侧签名）**语义不同**：一个授"写一片"、一个授"读一份"；**TTL 也分属不同配置**（`upload.partUrlTtl` vs `app_config.temp_url_ttl`），不要合并。
   - **签名由 provider 行为表决定**：`cdn_auth_type` 管的是**读**路径的签名方式，**不参与**分片上传地址的生成；分片签名一律走 provider 自己的 `UploadPart` 签名规则（provider 知识，属代码）。

## 4. 写入前必须归一化 `endpoint`

归一化走 `pkg/provider.CanonicalEndpoint`。规则：小写、去尾斜杠、去 scheme、剥掉 `{bucket}.` 前缀、按 provider 的域名族等价规则统一到 S3 兼容形态；归一化不出来的（provider 不在白名单、host 不匹配该 provider 的域名族）**拒绝写入**。

> **为什么必须归一化**：唯一键 `uidx_store_endpoint_bucket` 想表达"一个物理桶一行"，但它约束的是**两个字符串**。同一个桶有多种合法写法（域名族、vhost 形态、带不带 scheme、带不带尾斜杠），字符串不相等就放行，于是同一桶出现两行。而对象键不含 store 段 ⇒ 两行写同一个桶就落到**同一个物理对象**，`object` 的身份是 `(store_keyword, md5)` ⇒ 同一份字节分裂成两行；回收时删到某一行最后一个引用就 `Delete(key)`，**物理对象被删，另一行立刻悬空**。所以唯一键必须建在**物理身份**上，而不是**输入拼写**上。

归一化规则演进后，旧行与新行会重新分裂成两行（**纯代码变更、DB 无感知**）⇒ 启动自检必须做归一化断言，见 12-工程与交付。

## 5. `SK` 的处置

`store` 含 `AK` / `SK`。**`SK` 永不进响应**——任何查询 / 列表接口只返回掩码，明文只在写入路径接受。对应 `StoreService` **没有 Delete**（见 10-接口契约）。

`store` 初始行含真实 SK，其写入 SQL **不进 git**，由运维在部署后单独执行一次（见 12-工程与交付「运维脚本」）。
