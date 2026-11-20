# Resource SQL 设计（APP 维度重构）

设计原则：**厂商常量属于代码，部署决策属于数据库**。表里只存"这一行是什么"的事实，不存"由它能推出什么"的结论——推导一律在代码里做（寻址方式、分片下限、校验和行为等）。

**这条原则有一个限定条件**（2026-10-13 补）：判断一个可推导的列该不该存，问的不是"它能不能被推导出来"，而是"**它是否被读路径高频使用、且有一个结构不变式保证它不会漂移**"。正反两个例子分居两侧——`directory.parent_id` 被删，因为它只服务"列直接子目录"这一种查询，而那条查询实际走的是"全量取回目录 + 内存建树"（**只写不读**，且移目录时还会漂移）；`file.app` / `export.app` 被留，因为它们是列表主查询的索引前导列（**每一页都要用**），且分别由"完成事务内从 `directory` 派生"与"创建请求的同一事务内从任务行派生"两条结构不变式锁死。没有这条限定，上面那句会被读成"任何冗余列都不许有"，而 `file.app` 恰恰是必需的。

DDL 只保留 PK 和 UNIQUE 这两类结构性约束（它们是身份唯一性，代码里实现同等规则要多一次查询且并发下仍会竞态）。**值域校验（原 CHECK）和跨表引用（FK）一律不进 DDL**，由服务层在写入时校验、在启动时自检；代价是手工 SQL 可以写进非法值和孤儿引用，唯一的兜底就是"启动时校验失败即拒绝启动"，这条是承重墙。

表清单（8 张）：`store`、`app_config`、`directory`、`object`、`file`、`upload`、`export_task`、`export`。原草案里的 `upload_part` 已删，理由见 `upload` 一节；导出拆成 `export_task`（产出物 + 生产重试）与 `export`（用户请求记录）两张，理由见 `export` 一节。

旧表的处置：`directory_closure` **整表废弃**（现网 0 行），`file` / `export` 沿用表名但列推倒重写。闭包表带来的三个缺陷（删目录不级联、移动后孙代指向旧祖先、`UpdateDirectory` 不防环）在物化路径下结构性消失，见 `directory` 一节。

**注释约定**：DDL 里的 `COMMENT` 只写字段"是什么"（一个名词短语）。取值域、单位、格式、互斥关系、推导规则、跨列约束一律写在本文件的说明里；注释与说明冲突时以说明为准。

**单位约定**（现网最大的歧义源：`file.size` 是 KB、`directory.max_size` 是 MB、proto 不带单位，全靠注释）：

| 列 | 单位 |
|---|---|
| `app_config.default_max_size`、`directory.max_size` | MB |
| `app_config.temp_url_ttl`、`app_config.export_ttl`、`created_at`、`updated_at`、`expired_at` | 秒（时间戳为 Unix 秒） |
| `object.size` | 字节（`BIGINT UNSIGNED`） |
| API 的 `size` / `chunkSize`（proto） | 字节（与 `object.size` 同单位、无换算；两者**单位相同但类型不同**，见下方"类型分层"） |
| 分片大小 | 不进库。代码推导：`chunkSize = max(5MB, ceilDiv(total_size, 10000 × 1MB) × 1MB)`，全整数运算，服务端**先用专门的取片大小接口下发**（客户端据此算内容摘要），`PrepareUpload` 响应里再带一份（见第 5 节"chunkSize 的推导"与"内容摘要的算法"） |

规则：**人工录入的配置用 MB / 秒**（让人填 `10485760` 只会填错数量级），**程序写入的度量用字节**（唯一无歧义单位）。**列名一律不带单位后缀**——MB 和秒都不写进名字，单位只在这张表里查，看到 size / ttl / expire 类列不要凭名字推断。现网唯一的换算点是 `pkg.GetKBSize`（MB→KB，`internal/pkg/pkg.go:15-17`）。

### 类型分层（DB / Go / proto 三层不要求一致）

三层的选型依据不同，**不要求逐字一致**：

| 层 | 依据 | 类型 |
|---|---|---|
| DB | 容量——不设上限、永不迁移 | `BIGINT UNSIGNED` |
| Go entity / types | 与 DB 对齐 | `int64` / `uint64` |
| proto | **前端能不能解析** | 见下 |

proto 层的规则来自 **proto3 的 JSON 映射**（gRPC 线上 `uint64` 与 `int64` 都是 varint、无差别；差别只在 JSON）：

| proto 类型 | JSON 里的形态 |
|---|---|
| `int32` / `uint32` / `sint32` / `fixed32` | **number** |
| `int64` / `uint64` / `sint64` / `fixed64` / `sfixed64` | **string** |

这是规范行为，protojson **没有**"64 位输出数字"的开关。所以 **proto 里只要出现 64 位整数，前端拿到的就是字符串**。逐字段决策：

| 字段 | proto | 理由 |
|---|---|---|
| ids（`id` / `directoryId` / `parentId` / `ids`） | `uint32` | API 契约上限 `2^32-1` = 42.9 亿行；DB 用 BIGINT 只是不让这个上限提前到来 |
| `total`（分页计数） | `uint32` | 同上 |
| `createdAt` / `updatedAt` / `expiredAt` | `uint32` | Unix 秒，上限 **2106-02-07**——写成显式事实，不靠"还很远" |
| `page` / `pageSize` / `chunkCount` / `index` / `width` / `height` | `uint32` | 数量级天然小 |
| `chunkSize`（分片大小，字节） | `uint32` | **有上界**（≤ 42.94 TB 的单文件才可能溢出），见下 |
| **`size`（字节度量）** | **`string`** | 无上界，见下 |

**只有 `size` 需要超过 32 位**：`object.size` / `upload.total_size` 是 `BIGINT UNSIGNED` 字节，理由就是"`INT UNSIGNED` 上限 4 GB，而这张表天生服务分片上传的大文件，会溢出"（第 4 节）。API 若停在 32 位字节，单文件上限就被压到 4 GB，与本表的意图直接冲突。

**`chunkSize` 不需要**——它有上界，而且是窄上界。`chunkSize = max(5MB, ceilDiv(total_size, 10000 × 1MB) × 1MB)`（第 5 节），要让 `chunkSize` 超过 `uint32` 上限，需要 `total_size > 4095 × 1MB × 10000 = 42,939,187,200,000` 字节，即 **≈ 42.94 TB**。这个数既低于厂商单对象上限（BOS 48.8 TB）、又远超任何实际需求，所以：

- `chunkSize` 保持 `uint32`，不跟着 `size` 走 `string`；
- 代价是**单文件上限被隐式钉在 42.94 TB**。这条要写成显式事实，并在两处落地：① `directory.max_size`（MB）的校验上界——**`max_size = 4294967295`（"不限"）时有效上限取 42.94 TB 这个物理天花板**，所以"不限"不等于"跳过校验"，它等于"取物理上限"；② 计算 `chunkSize` 处的断言——算出来超过 `uint32` 上限即报"文件过大"，**不得静默截断**（截断会让服务端下发的 `chunkSize` 与客户端实际切片不一致，表现为永远合不上）。
- **不要为它另造一个业务常量**：上界是公式与类型推导出来的，人为取一个更小的值只会多一个需要同步的地方。

**为什么单位不能退回 KB**（那样 32 位能到 4 TB）：P0-3 的两层 `size` 校验是"**实际收到的字节数 == 请求声明的 `size`**"的精确比对。KB 只能声明取整值——一个 1025 字节的文件，声明 1 KB（1024）或 2 KB（2048）都对不上 1025；要让它成立就得在两处各加一套取整规则，且方向必须逐字一致，失配表现为**随机拒绝上传**。这是拿取整约定去换上限，而它正好落在最敏感的那条线上。**所以 API 的 `size` 是字节，这是被校验倒逼的。**

**为什么声明 `string` 而不是 `uint64`**：proto3 JSON 本来就会把 64 位整数渲染成字符串，两者在前端看到的完全一样；但显式 `string` **不依赖"序列化器恰好这么做"**——换 protojson / grpc-gateway / 任何网关都是 string，契约诚实，也把"这个字段可能超过 32 位"写在了类型上。前端用 `Number(s)` 转换（JS 安全整数上限 `2^53` ≈ 9 PB，远超任何单文件）。

**边界转换：按"上界是否已知"分流**（"禁止静默截断"只约束**上界未知**的情形）——三层类型不同，每层交界处都有一次转换。判断"怎么转"只问一件事：**这个字段有没有上界，上界在不在文档里**：

1. DB(`BIGINT`) → Go(`int64`/`uint64`)：无转换。
2. Go(`int64`/`uint64`) → proto(`uint32`)：分两种情形——
   - **上界已知、且已在本文件写成显式事实**（时间戳、ids、`total`、`chunkCount` 等）：允许直接写 `uint32(x)`，但该转换点必须登记在下方清单里。理由：上界是文档事实，不是"大概不会超"。
   - **上界未知或无上界**：**不得** `uint32(x)`。要么把 proto 改成 `string`（`size` 就是这么处理的），要么在转换处显式校验并报错。
3. Go(`uint64` 字节) ↔ proto(`string`)：`strconv.FormatUint` / `ParseUint`，解析失败即报错。

**已登记的时间戳转换点（12 处，2026-10-13 核实）**：`internal/app/export.go:61/62/63/152/153/154`、`internal/app/file.go:76/77/118/119`、`internal/app/directory.go:52/53`。它们把 Go 侧的 `int64` 时间戳（`model.BaseModel.CreatedAt/UpdatedAt`、`entity.Export.ExpiredAt`）转成 proto 的 `uint32`。

**当前全部 lossless**：Unix 秒现为 `1.79e9` 量级，`uint32` 上界 `4,294,967,295` 对应 **2106-02-07 06:28:15 UTC**，余量约 80 年。**类型不一致本身不是问题**——`int64`（Go）+ `uint32`（proto）正是本节"三层不要求一致"的合法形态，不需要"统一"。

**本轮决定：不加范围护栏、不改 proto 类型**。理由：上界已知、已写进文档、余量 80 年；为一个不会发生的越界引入 12 处调用点改动不划算。**但"禁止静默截断"这条规则不因此作废**——它约束的是**上界未知**的转换；上面 12 处放行，是因为它们**有登记、有上界、有复核时点**。因此：

- 新增时间戳字段时，必须把转换点补进本清单；
- 任何一个字段**改变语义**（例如 `expired_at` 从"Unix 秒"改成毫秒或纳秒计数）→ 上界随之变化，本清单立即失效，必须重新评估。这是这 12 处唯一的失效触发条件；
- 清单之外出现新的 `uint32(时间戳)` 转换，视为违规。

**API 的 `size` 单位 = 字节**，与 `object.size` 一致、无换算。现网两侧都是 KB（响应 `entity.File.Size` 注释即"文件大小（KB）"、`internal/app/file.go:73` 直接透传；请求 `internal/domain/service/file.go:159` 与 `pkg.GetKBSize(limit.MaxSize)` 比较、`:232` 用它算 chunkCount），本次一并改为字节，`pkg.GetKBSize` 随之删除。

**定性**：proto 类型变更 + 单位变更都是**非兼容**改动，前端需配合改 `size` 的解析（`Number(s)`）。这是本次重构可接受的代价（前提：不考虑向后兼容）。

---

## 1. `store` — 对象存储后端

```sql
CREATE TABLE `store` (
  `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `keyword`         CHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '后端标识',
  `name`            VARCHAR(64)  NOT NULL COMMENT '显示名称',
  `provider`        VARCHAR(32)  NOT NULL COMMENT '厂商标识',
  `endpoint`        VARCHAR(255) NOT NULL COMMENT '厂商API域名',
  `region`          VARCHAR(64)  NOT NULL COMMENT '地域',
  `bucket`          VARCHAR(63)  NOT NULL COMMENT '桶名',
  `ak`              VARCHAR(128) NOT NULL COMMENT '访问密钥ID',
  `sk`              VARCHAR(512) NOT NULL COMMENT '访问密钥',
  `public_base_url` VARCHAR(255) NOT NULL COMMENT '公网访问地址前缀',
  `cdn_auth_type`   VARCHAR(16)  NOT NULL DEFAULT 'none' COMMENT '临时URL鉴权方式',
  `cdn_auth_key`    VARCHAR(255) NULL COMMENT 'CDN鉴权key',
  `status`          TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态 1启用 0停用',
  `description`     VARCHAR(256) NULL COMMENT '备注',
  `updated_by`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最后修改人',
  `created_at`      BIGINT UNSIGNED NOT NULL COMMENT '创建时间',
  `updated_at`      BIGINT UNSIGNED NOT NULL COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uidx_store_keyword` (`keyword`),
  UNIQUE KEY `uidx_store_endpoint_bucket` (`endpoint`, `bucket`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='对象存储后端';
```

### 说明

- **字段 `id`**：主键。**全库引用 `store` 一律用 `keyword`，不用 `id`**（`app_config`、`object`、`upload`、`export` 都是 `store_keyword`）——桶归属在数据里必须人可读：运维排查、GC 脚本、按桶统计、和 CDN 日志对账都能直接写 `WHERE store_keyword='bdemo'`，不需要先把数字 id 翻译成名字（那个翻译依赖进程内的 store map，人肉查库和离线脚本没有它）。代价是每条含 store 的索引每行多背 120 字节（`CHAR(32)`×4 = 128 vs `BIGINT` = 8），`object.uidx_object_store_md5` 因此是 160 字节（`store_keyword` 128 + `md5` 32，见第 4 节"字符集口径"），仍远低于 3072。**都不声明 FK**，引用完整性由服务层写入校验 + 启动自检保证，再加上一条结构性前提：`store` 行**永不删除**（见"行只增不减"）。
- **字段 `keyword`**：人类可读的后端标识，用于展示、全库引用（`app_config`/`object`/`upload`/`export` 的 `store_keyword`）、查重。`utf8mb4_bin` 是因为它与 manager 的 `app.keyword` 同一套命名习惯，现网数据大小写混用，不能做大小写不敏感比较。**不参与对象键生成**（对象键是 `{md5}.{ext}`，见第 4 节），但它是公网 URL 的 host 归属依据。**本列创建后不可修改**：它既是 `app_config` 的引用目标，又是 `object` 表里亿级历史行的桶指针——改一个 keyword 等于把全表历史数据指向一个不存在的桶。store 的更新接口不接受 `keyword` 字段，要换桶就新建行再重配 app；DB 拦不住 UPDATE，只能靠代码。**同理本表不提供删除接口**——行只增不减，见下方"行只增不减"一节。
- **字段 `name`**：界面上显示的名字，与 `keyword` 分开是为了改名时不动 `keyword`（`keyword` 可能出现在外链、日志、历史工单里）。
- **字段 `provider`**：取值 `aws`/`bos`/`oss`/`cos`/`obs`/`minio`/`uss`，不在白名单内即启动报错。它是唯一的厂商知识入口——寻址方式、分片大小下限、ETag/校验和行为、图片处理语法全部由它推导，代码里是一张 `(provider, region) → 行为` 的小表。
- **字段 `endpoint`**：只服务"我们 → 厂商"的直连请求（Put / CompleteMultipartUpload / ListParts / Delete）。CDN 读路径不使用它。**本列只存归一化后的形态**（见下方"endpoint 归一化"）——这是 `uidx_store_endpoint_bucket` 能真正表达"一个物理桶一行"的前提。
- **字段 `region`**：各家格式不同，不做归一化，按 provider 要求的原样存——`bos` 用 `bj`、`oss` 用 `cn-beijing`、`cos` 用 `ap-beijing`。
- **字段 `bucket`**：桶名。COS 需要 `name-appid` 形式，这个差异属于 provider 规则，代码校验。
- **字段 `ak` / `sk`**：桶访问凭证。**`sk` NOT NULL，数据库是唯一来源**，配置/env 不再参与注入。明文落库，与 `manager.oauther.sk` 同规范。
- **字段 `public_base_url`**：公网访问地址 = `{public_base_url}/{对象名}`，对象名规则见第 4 节（`{md5}.{ext}`，带后缀、不带桶名）。带后缀是为了让**只看见 URL 的一方**也能识别文件类型——CDN 的图片处理和缓存规则、接入方客户端的渲染判断都按后缀匹配，裸 md5 在这两处都不可识别。桶名出现在 host 还是 path 完全由这一列表达（回源 vhost → `https://cdn-host`；回源 path-style 或多桶共用一个 CDN 域名 → `https://cdn-host/{bucket}`），因此不需要额外的"桶名在路径里"开关列。**一个 store 只有一个公网域名**：多域名（多地域加速、内网域名、灰度域名）由 CDN 侧 CNAME 承担，不入库——库里存多个域名就要求代码决定"新文件返回哪个"，那会让历史文件用域名 1、新文件用域名 2，两边都没有道理。**这一列是对外承诺**：`object` 的公网 URL 由它拼出，且完整 URL 会被接入方持久化进自己的库，改动等于打断存量外链，所以更新它定性为"承诺变更"（要求显式确认标记，响应里回显受影响对象数量级），而不是普通字段修改。
- **字段 `cdn_auth_type`**：临时 URL 的鉴权方式。三种取值互斥：`none`（公有读）、`cdn_a`（CDN Type-A key 签名）、`s3_presign`（厂商 SigV4 预签名）。没有实现过的 `cdn_b`/`cdn_c` 不预留。
- **字段 `cdn_auth_key`**：CDN 侧单独生成的鉴权 key。必须独立成列——现状代码把**桶 SK 当作 CDN 签名 key 用**（`internal/infra/store/s3.go:91`），两者轮换周期和泄露面完全不同。
- **字段 `status`**：取值 1 启用、0 停用。0 只表示"不接受新建上传与新建导出"，不表示"存量对象不可读"。所以它绝不能参与读路径的判断，**也绝不能用来决定"这一行要不要加载进进程内 store map"**——停用的 store 必须照常加载，否则它名下的存量对象全部 404（见下方"行只增不减"）。它是本表**唯一的生命周期出口**：没有删除，只有停用与再启用。**本表不建 `status` 索引**（2026-10-13 定，原 `idx_store_status` 已删）：全表是几十行的配置数据，`status` 是二值，而所有读路径都走"启动时全量加载进进程内 map"，没有任何查询会按它过滤——索引只增加写入成本与一份要维护的元数据。
- **字段 `description`**：纯人工备注，不参与任何逻辑。
- **字段 `updated_by`**：最后修改人的用户 ID（manager 侧 `user.id`，0 表示非人工变更）。因为 `sk` 进了数据库，改密钥变成一件需要追责的事。
- **字段 `created_at` / `updated_at`**：Unix 秒，房内规范。若将来给 S3 客户端加缓存，失效键直接用 `updated_at`（当前 `internal/infra/store/store.go:17` 是每次调用新建客户端，无缓存，因此不引入 `rev` 列）。

### 行只增不减：没有删除，只有停用（2026-10-13 定）

**规则**：`store` 表**只增不减**。代码不提供删除接口，也不写任何 `DELETE FROM store`。

**理由**：`store.keyword` 是 `object` / `upload` / `export_task` 三张表里历史行的桶指针。删掉一行，那批字符串立刻变成**无法解析的孤儿**——它们的公网 URL 直接 404、临时 URL 签不出来、GC 构造不出 `AbortMultipartUpload` 的桶上下文，而**库里看不出任何异常**。这与"`keyword` 创建后不可修改"是同一件事的两面：一个禁的是改指向，一个禁的是删指向。两者都只能靠代码（DB 层既拦不住 `UPDATE` 也拦不住 `DELETE`），且都不在"手工 SQL 改库"这个威胁模型内。

**下线一个桶 = `status=0`**，语义边界：

| 动作 | 停用后 |
|---|---|
| 新建上传 / 新建导出 | **拒绝**（`store.status` 与 `app_config.status` 是 AND 关系，见第 2 节） |
| 存量对象的 URL 解析、CDN 回源 | 照常（读路径完全不看 `status`） |
| 临时 URL 签名 | 照常 |
| `upload` 的 GC 调 `AbortMultipartUpload` | 照常（靠 `upload.store_keyword` 构造桶上下文） |
| 重新启用 | 允许（`status` 双向可变） |

**由此产生一条最容易写错的实现约束**：**启动自检必须加载全部 store，含 `status=0` 的那些**。停用的 store 不进进程内 map，就等于把它名下的存量对象全部变成 404——这是"停用"与"删除"在实现上**唯一的区别点**，也是最容易被"只加载启用的"这种直觉写错的地方。"CHECK 移除后唯一兜底"那句里的"全量加载"，指的就是**含停用行**。

**与其余表对照**：`upload` / `file` 的哲学是"**行在即活**"（删行即终结，所以不存 `status`）；`store` 的哲学是"**行永不删**"（所以它存 `status`，且 `status` 只表达"还接不接新活"）。两者不冲突，因为前者是运行期的高频增删，后者是配置期的低频增改。

**连带（本决策的直接后果）**：

1. **不再需要"删 store 前反查引用"**：没有删除动作，就没有删除前的检查。`app_config` 的 `idx_app_config_store` 随之删除（文档原本写明它就是为那条检查存在），第 2 节代码侧义务第 3 条改写，第 4 节连带义务 7 与第 7 节 7.7 条 7 一并撤掉。
2. **引用完整性从此由"行永不消失"承担**，而不是由删除时的反查承担。这**强化了"不声明 FK"的立场**：原来要论证"删之前查三处，等价于 `ON DELETE RESTRICT`"，现在连这个论证都不需要——行不会消失，引用就不会悬空。剩下唯一要检查的是**写入方向**（`app_config` 写入时校验 keyword 存在，见第 2 节代码侧义务 2）。
3. `store` 行数单调增长。配置类数据，量级是个位数到几十，不需要回收策略。

### 为什么不存的东西，以及由谁负责

- **寻址方式（path-style / virtual-hosted）不入库**，代码推导：
  1. 按 provider 默认：`oss`/`cos` → vhost，其余 → path；
  2. vhost-only 的例外按 `(provider, region)` 白名单（AWS 2025 年后部分新区域强制 vhost）；
  3. `provider` 不在白名单 → **启动报错**，禁止静默默认。

  **原来那条"host 首段 == bucket → vhost"已删除**：它靠 `endpoint` 里带桶名来反推寻址方式，而"同一物理桶有多种 endpoint 写法"恰恰是要归一化掉的（见下）。归一化把 `{bucket}.` 前缀从 host 里剥掉后，这条规则永远不触发、沦为死代码；且"用哪个域名族"本身就决定了寻址方式（BOS 归一到 S3 兼容域名 → path），不需要再从字符串里猜。删掉它之后，寻址方式只由 `(provider, region)` 决定，判定入口从两个收敛成一个。
- **值域校验不写进 DDL**，由服务层承担，落在四处：① 值域（provider 白名单、`cdn_auth_type` 三值、`status` ∈ {0,1}）；② 上面那条寻址推导；③ `endpoint` 归一化（见下）——归一化失败即拒绝；④ `endpoint` / `bucket` / `public_base_url` / `cdn_auth_type` 四元组自洽性——`s3_presign` 配 CDN 域名会让 SigV4 的 `host` 签名失效整域 403，`cdn_a` 配厂商域名则没有执行方、`cdn_auth_key` 白配。（原来的"vhost 形状冲突 → 双桶段 404"随归一化消失：`endpoint` 不再携带桶名。）写入接口和启动加载共用同一套校验函数。
- **CHECK 移除后唯一的兜底是"启动时全量加载 `store` 并逐个构造客户端，任一失败拒绝启动"**（沿用 `internal/infra/store/s3.go:47-49` 缺 Secret 即早失败的语义）。手工 SQL 写进非法值只能靠它拦，所以这条是承重墙，不能省。**"全量"含 `status=0` 的停用行**——它们是存量对象的桶指针，漏加载即整片 404（见"行只增不减"）。
- **`public_base_url` 之间不得存在前缀包含关系**（启动自检的一条）：`https://cdn/a` 与 `https://cdn/a/b` 可以共存，但当某个 app 名恰好等于 `b` 时，前一个 store 里该 app 对象的 URL 与后一个 store 的前缀**完全同形**，按 URL 反解 store 的最长前缀匹配会把它判给错误的桶，表现为图片 404 或串成别人的图。等值已被唯一性挡住，前缀包含挡不住，必须显式校验。

### endpoint 归一化（`uidx_store_endpoint_bucket` 成立的前提）

**问题**：`uidx_store_endpoint_bucket(endpoint, bucket)` 想表达"一个物理桶一行"，但它约束的是**两个字符串**。同一个物理桶有多种合法写法，字符串不相等就放行，于是同一桶出现两行：

| 差异 | 例 |
|---|---|
| 域名族（BOS 两种合法入口） | `s3.bj.bcebos.com`（S3 兼容）vs `bj.bcebos.com`（原生） |
| vhost 形态（桶名进了 host） | `bdemo.bj.bcebos.com` vs `bj.bcebos.com` |
| 带不带 scheme | `https://s3.bj.bcebos.com` vs `s3.bj.bcebos.com` |
| 带不带尾斜杠 | `s3.bj.bcebos.com/` vs `s3.bj.bcebos.com` |

**为什么后果严重**：对象键是 `{md5}.{ext}`，**不含 store 段**。两行写同一个桶 ⇒ 落到**同一个物理对象**；而 `object` 的身份是 `(store_keyword, md5)` ⇒ 同一份字节分裂成**两行 `object`**。回收时删到某一行最后一个引用就调 `Delete(key)`，**物理对象被删，另一行立刻悬空**（其下文件全 404），且库里看不出异常。所以唯一键必须建在**物理身份**上，而不是**输入拼写**上。

**处置**：`endpoint` 列**只存归一化后的值**，写入接口负责归一化，DB 永远见不到原始拼写。归一化规则：

1. **小写、去尾斜杠、去 scheme**（本列是域名不是 URL；scheme 由 provider 表决定，默认 `https`）；
2. **剥掉 `{bucket}.` 前缀**——`bdemo.bj.bcebos.com` → `bj.bcebos.com`（vhost 形态还原为服务域名）；
3. **按 provider 的域名族等价规则归一**：BOS 的 `s3.<region>.bcebos.com` 与 `<region>.bcebos.com` 视为同一族，统一到 **S3 兼容形态**（与现网一致，且 path 寻址对桶名无约束）；
4. **归一化不出来的**（provider 不在白名单、host 不匹配该 provider 的域名族）→ **拒绝写入**，不落库。

第 3 条是厂商知识，放进 `provider` 那张 `(provider, region) → 行为` 的表（见字段 `provider` 说明），不散落在校验代码里。

**为什么落在写入路径、不加新列**：加一列 `endpoint_key` 就等于承认"存了一份原始值 + 一份规范值"，两份真相必然漂移。归一化后 `endpoint` 自身就是规范值，唯一键直接用现有列——**不新增列、不需要迁移**（本次是清库重建）。

**连带**：`public_base_url` 要做同类归一（至少去尾斜杠，否则 `{public_base_url}/{对象名}` 会拼出双斜杠）。它与 `endpoint` 是两个不同域名（一个是 API、一个是 CDN），各自归一。

### 本表决策带来的连带义务

1. **配置侧 `business.storage[]` 下线**：`internal/core/conf.go:15-27` 的 `Storage` 结构体去掉凭证字段，`internal/infra/store/store.go:13` 的 `createStore` 改为读 DB，`conf/conf.yaml` 删除该段。
2. **`sk` 永不回显**：任何查询/列表接口只返回掩码，明文只在写入路径接受。
3. **DDL 与 seed 分离**：含真实 SK 的 `INSERT` 不进 git，由运维部署后单独执行一次。
4. 安全遗留（未处理，需单独决策）：`.env` 目前被 git 跟踪且有多次提交历史，内含真实 BOS AK/SK，且该 SK 同时被复用为 CDN 鉴权 key。建议先轮换 SK，再让新 SK 只进 DB，然后 `git rm --cached .env` + 写 `.gitignore`。不改 git 历史。
5. **写入接口必须归一化 `endpoint`**（见"endpoint 归一化"）：不归一化就等于唯一键失效，同一桶可建两行。**启动自检加一条归一化断言**——全量加载时对每行跑 `canonicalEndpoint(provider, endpoint)`，必须等于存储值。它兜的不是"手工改库"（不在威胁模型内），而是**归一化规则演进后存量行没跟着变**：规则一改，旧行就可能与新行重新分裂成两行，而这是纯代码变更、DB 无感知。
6. **更新接口的字段白名单里没有 `keyword`，也没有删除动作**：可变更的是 `name` / `provider` / `endpoint` / `region` / `bucket` / `ak` / `sk` / `public_base_url` / `cdn_auth_type` / `cdn_auth_key` / `status` / `description`。`keyword` 不可修改、行不可删除（见"行只增不减"），两条都靠**接口形状**保证，而不是靠写入时的校验代码——没有入口的字段不会被误改，校验代码则会被绕过。
7. **图片处理能力整体下线**（2026-10-13 定）：删除 `ParserQuery` / `VerifyTemporaryURL` 两个接口方法（`internal/infra/store/s3.go:72,99`，目前是空实现），本表也不加 `image_process_dialect` 列。理由与它顺手消掉的那个 CDN 缓存坑，见下方"已结（2026-10-13）"。

### 待定项

无（2026-10-13 清空）。

已结（2026-10-13）：**图片处理方言不留**。原待定项是"`image_process_dialect` 留不留，决定 `ParserQuery` / `VerifyTemporaryURL`（`internal/infra/store/s3.go:72,99`，目前是空实现）补实现还是删掉"。定为**不留**：

- **本表不加这一列**（DDL 里本来也没有），**`ParserQuery` / `VerifyTemporaryURL` 两个接口方法一并删除**（它们目前是空实现，没有使用方）。
- 连带删掉"厂商图片处理语法"这条知识线——`provider` 那张 `(provider, region) → 行为` 的表里不再需要为它留位置。
- 图片处理改由**接入方 / CDN 侧**承担：URL 是公网可读的 `{public_base_url}/{md5}.{ext}`（带后缀，见第 4 节），接入方要缩略图就自己在 CDN 上配规则。服务不参与拼参数。
- 顺带消掉一个坑：走 CDN 时图片处理参数依赖 **CDN 缓存 key 是否含 query**，不含则所有尺寸命中同一份缓存并**静默返回错图**。这个坑不在本服务能控的范围内，所以更不该由本服务提供这个能力。

已结（2026-10-12）：`keyword` 退出对象键——对象键里既没有 keyword 也没有桶名，`store` 的归属由 URL 的 host（`public_base_url`）表达。现网缺陷的根因不是"键里编了 store"，而是有两处各自拼 key 且没有唯一的构造/解析函数（`internal/domain/service/file.go:53-55` 拼一层 store，`internal/app/file.go:377` 的 Redirect 又拼一层），拼出双桶段。

---

## 2. `app_config` — 应用存储配置

**状态：定稿**（2026-10-12）。`directory` 的继承方式已定为快照式；store 引用形态已全库统一为 `store_keyword`（见第 4 节），本表无待定项。

```sql
CREATE TABLE `app_config` (
  `id`                   BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `app`                  CHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '应用标识',
  `store_keyword`        CHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '上传落地的存储后端',
  `export_store_keyword` CHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NULL COMMENT '导出产物的存储后端',
  `default_accept`       VARCHAR(255) NOT NULL COMMENT '默认允许后缀',
  `default_max_size`     INT UNSIGNED NOT NULL COMMENT '默认单文件上限',
  `temp_url_ttl`         INT UNSIGNED NOT NULL DEFAULT 900 COMMENT '临时URL有效期',
  `export_ttl`           INT UNSIGNED NOT NULL DEFAULT 259200 COMMENT '导出产物保留期',
  `status`               TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态 1启用 0停用',
  `updated_by`           BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最后修改人',
  `created_at`           BIGINT UNSIGNED NOT NULL COMMENT '创建时间',
  `updated_at`           BIGINT UNSIGNED NOT NULL COMMENT '修改时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uidx_app_config_app` (`app`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='应用存储配置';
```

### 说明

- **字段 `app`**：`CHAR(32)` + `utf8mb4_bin` 逐字对齐 manager 的 `app.keyword`。不建外键（跨库，`manager`）。本表**不存**应用名、负责人、应用状态——那些属 manager，这里只存 resource 服务自己的策略。
- **字段 `store_keyword`**：引用 `store.keyword`，**不声明 FK**。两侧的类型、字符集、排序规则必须逐字一致，否则比较走隐式转换、`uidx_store_keyword` 可能不被命中；`utf8mb4_bin` 区分大小写，`bdemo` 与 `BDemo` 不匹配。
- **字段 `export_store_keyword`**：NULL = 使用 `store_keyword`。故意不用 `NOT NULL DEFAULT ''`——那会允许两列各存一份同样的值，其中一份过期也没人发现。
- **字段 `default_accept`**：逗号分隔后缀，与 `directory.accept` 同形态，255 长度约容 60 个常见后缀。**取值约定（2026-10-13 定，与 `directory.accept` 逐字同规）**：`'*'` = **允许全部类型**；`''`（空串）= **不允许任何类型**（"只放目录、不放文件"的目录）；其它 = 后缀白名单。它是**建目录时抄给 `directory.accept` 的模板初值**，不是运行时生效值（见"继承链"）。取代全局 `conf.defaultAcceptTypes`。
  **不给 DEFAULT**：旧稿用"空 = 不限"，漏写静默落成"不限"（宽松）；改成语义明确的 `*` / `''` 之后，零值 `''` 变成"**拒绝一切上传**"（严格，但同样是静默的错误配置）。所以按 `created_at` 同规去掉 DEFAULT——严格模式下漏写直接报错，而不是让某个 app 悄悄拒掉所有上传。
- **字段 `default_max_size`**：**单位 MB**，类型 `INT UNSIGNED` 与 `entity.Directory.MaxSize`（uint32）对齐。**取值约定（2026-10-13 定）**：值域 `1 .. 4294967295`，**`0` 非法**（校验拒绝，不再表示"不限"）；`4294967295`（`INT UNSIGNED` 上界 ≈ 4.29 PB）**表示"不限"**——它是类型自身的边界而不是新造的业务常量（与 `chunkSize` 的 42.94 TB 同一种"由类型 / 公式推导"的写法），任何真实上限都远小于它，所以不会与"真想设 4.29 PB"混淆；代码里用具名常量引用，不写字面量。作用方式同上。取代全局 `conf.DefaultMaxSize`（也是 MB，`internal/infra/dbs/directory.go:146`）。
  **不给 DEFAULT**，理由同上：`0` 现在是非法值，`DEFAULT 0` 只会让漏写变成一行校验不过的数据。
- **字段 `temp_url_ttl`**：临时访问 URL 有效期，**单位秒**（整数，不是 `conf.yaml` 里 `72h` 那种 duration 字符串；转换点只在读写 app_config 的代码里）。**900 是新设定值，不是从配置搬迁**——现网 `conf.yaml` 的 storage 条目根本没有配 `temporaryExpire`，所以 `core.Storage.TemporaryExpire = 0`，`s3.go:62` 拿到 0，`s3.go:90` 的签名时间戳就是"当下"（URL 只能活到分钟边界），`s3.go:93` 的 redis 缓存写入 TTL 是 `0 - 10s` = **负 TTL**。目前没暴露，是因为 `antiTheft` 也没配，`GenTemporaryURL` 走的是无签名分支（`s3.go:75-76`）直接返回；一打开防盗链两条都会现形。
  代码校验边界：下限 60（CDN Type-A 的时间戳是 `200601021504` 分钟粒度，短于一分钟没有意义）；`store.cdn_auth_type = 's3_presign'` 时上限 7 天（SigV4 硬限制）；`= 'none'` 时本列无效。
  **读法**：读路径经 Redis 取本列、**不查库**（见下方"缓存策略"）；`cdn_auth_type = 'none'` 时本列不参与，连 Redis 都不访问。
- **字段 `export_ttl`**：导出产物保留期，单位秒。取代 `conf.yaml` 的 `export.expire: 72h`（默认 259200 = 3 天，与现值一致）。改成按 app 配置后，导出清理任务必须**按 app 分别取 TTL**，不能再用一个全局常量扫全表。
- **字段 `status`**：1 启用、0 停用。0 只拒绝该 app 的新建上传和导出，存量对象仍可读，不参与读路径判断。与 `store.status` 是 **AND 关系**——两者任一为 0 都拒新建；读路径两个都不看。
- **本表不提供删除接口**（2026-10-13 16:33 定）：与 `store` 同立场（第 1 节"行只增不减"）——本表是**被数据面引用的配置行**（`status` 是上传闸门、`store_keyword` 是写入落桶的依据），删掉会让历史行为无法解释。下线一个 app = `status=0`，**走 `Update` 写动词**（因此自动带上延迟双删，见"代码侧义务"第 6 条）。DB 拦不住 `DELETE`，靠"代码里根本没有那个入口"兑现。
- **字段 `updated_by`**：最后修改人的用户 ID（manager 侧 `user.id`，0 表示非人工变更），与 `store.updated_by` 同规范。
- **字段 `created_at` / `updated_at`**：Unix 秒，由代码写入，**故意不给 DEFAULT**——严格模式下漏写会直接报错，而不是静默存 0。
- **本表只有 `uidx_app_config_app` 一个索引**（2026-10-13 定）：原 `idx_app_config_store` 的唯一用途是"删 store 前反查引用"，随"`store` 行不可删除"（第 1 节"行只增不减"）一并删除。本表的读取全部走唯一键（点查 + Redis 回填，见下方"缓存策略"），不需要第二个索引；这也顺带避免了在几十行的表上维护一个恒不命中的索引。

### 继承链（已定：快照式）

默认值只在**建目录时**抄一次：`directory.accept` / `directory.max_size` 在新建时取当时的 `app_config.default_accept` / `default_max_size`，之后独立演化。上传校验只读 `directory` 一行，不回读本表。

**这是语义决策，不是性能决策。** 旧稿这里给的理由是"上传校验只读 `directory` 一行……路径解析本来就是热路径"，站不住——继承是在**建目录时**解析的，与上传热路径无关；而且本表走 Redis 缓存之后（见下"缓存策略"），回读的代价已经只有一跳。真正选快照式的理由有三条：

1. **行为稳定**：目录的校验规则在其生命周期内稳定。跟随式（`NULL` = 继承）下改一次 `default_*`，会让**所有未覆写的目录立即变行为**——同一个目录昨天能传 `.zip`、今天不能，而且没有任何记录说明为什么。快照式只在新建目录时生效。
2. **可审计**：目录行自己就是当时的完整策略。排障时"这个目录为什么拒了"一行就能答；跟随式要连 `app_config` 一起读，答案分散在两行、两个层级。
3. **能力不对称**：快照式能做"把某批目录冻结在当前值"的回溯脚本（全量覆写）；跟随式做不到"冻结"。

跟随式省下的只是"建目录时抄一次字段"——一次 INSERT 的字段填充，不是任何热路径成本。

"不限"在两个层级用**同一套写法**（2026-10-13 定）：类型用 `'*'`、大小用 `4294967295`。所以 `app_config.default_*` 写成 `*` / 上界 = 该 app 不限类型、不限大小，`directory` 那行同理。**两级都是 `NOT NULL` 且没有"未设定"这个态**——`''` 与 `0` 都不表示"未设定"，前者是"不允许任何类型"，后者是**非法值**。**跟随式会给这条规则加一个例外**：目录层会出现 `NULL` / `''` / `'*'` 三态，API 与 entity 都要跟着区分"跟随默认"与"显式设为不限"。

**明确接受的代价**：改 `app_config.default_*` **不会回溯**到已建目录。运营改了默认值却发现老目录行为不变，是这类设计最常见的困惑来源；真要回溯只能写一次性脚本，按 `path LIKE '前缀%'` 批量 UPDATE 子树，且必须区分"老目录本来就被人工覆写过"和"老目录在跟随默认值"——快照式**没有这个信息**（NULL 才能表达"跟随"），所以回溯脚本只能全量覆写，会冲掉人工设定。这条是选快照式的实际代价，不是可以靠加列回避的问题。

**如果将来真要做跟随式**：正确形态是给 `directory` 加一个显式的 `follow_default` 布尔位，而不是把 `NULL` 当开关——布尔位把"跟随"与"显式设为不限"这两件事分开表达，语义不含糊，也不会让"`'*'` / `4294967295` = 不限"这条贯穿两级的写法出现例外。

### 行来源与缺失行为

- **必须预先显式配置**。缺行时写路径直接返回明确的"app 未配置"错误，**不做懒创建**——`GetDirectoryLimitByPath` 的 `FirstOrCreate`（`internal/infra/dbs/directory.go:128-159`）已经证明隐式写入会带来 N 次查询和并发重复键竞态。
- 缺行**不影响读路径**：已存对象走 `object` 上的 store 引用解析桶，不经过本表。所以"未配置"和 `status=0` 行为一致——拒写不拒读。
- 多余的行（manager 侧 app 已删除）无害：只有该 app 再次带流量才会被读到。

### 缓存策略（Redis + 延迟双删）

本表的读法决定它**必须被缓存**：读路径要 `temp_url_ttl` 才能签临时 URL，而本表是运行时可变的库表。若读路径直接查库，第 4 节的"读路径 0 查询"就不成立；若只在进程内启动加载，多实例下改了配置其他实例看不到。所以本表走 **Redis 缓存**，失效用**延迟双删**。

**载体与键**：Redis。键 `resource:app_config:{app}`，值 = 该 app 的**整行快照**（`store_keyword` / `export_store_keyword` / `default_accept` / `default_max_size` / `temp_url_ttl` / `export_ttl` / `status`），序列化后存储。**缓存整行而不是只缓存 `temp_url_ttl`**：写路径（`store_keyword` / `status`）读的是同一份，省一次点查；更重要的是只有一份快照，不会出现"同一次请求里 TTL 是新值、`store_keyword` 是旧值"的撕裂。

**读**：`GET` 命中即用（**0 次 DB 查询**）；未命中 → 查库（走 `uidx_app_config_app`）→ 回填（**带 TTL 兜底**，建议 1 小时）→ 用。查库也无行 → 回填**空结果负缓存**（**10s**）并按"app 未配置"处理（第 146 行的缺失行为）——它挡的是**穿透**（客户端拿不存在的 app 反复查，每次都落库等于把缓存让开）。**负缓存与正缓存同 key、用哨兵值区分**（2026-10-13 16:33 定）：`SET key "" EX 10`，读时 `GET` 返回 `nil` = 真 miss、返回 `""` = 已知不存在、返回 JSON = 命中，三者天然可区分；一次 `DEL` 同时清掉两种状态。代价是新配置**创建**后最多 10s 内仍被当"未配置"（`Create` 不删缓存，见"代码侧义务"第 6 条）。

**写（延迟双删）**：

1. 更新 `app_config` 行；
2. **删缓存**（第一次）；
3. **延迟后再删一次**（第二次），延迟建议 500ms。

第二次删的必要性：并发读可能"**在更新前读到旧值、在更新后才把旧值回填进缓存**"，第一步删不掉这份回填，延迟后的第二次删把它清掉。延迟时长必须 **> "读请求从查库到回填"的最坏耗时**——这是工程经验值（500ms–1s），给不出严格上界，因此**不做成可配置项**（配置化会诱导人以为它可精确调优）。

**一致性口径**：这是**最终一致 + TTL 兜底**，不是强一致。双删之后仍有极小概率的陈旧回填（读请求跨越两次删、且回填慢于延迟），概率低到可忽略，且 Redis TTL 会兜底。配置类数据的容忍度足够。

**为什么 `store` 不走 Redis、本表走**：`store` 的 `keyword` 不可修改、行**只增不减**（只停用不删除，见第 1 节"行只增不减"），进程内启动加载最省（零网络跳）；`app_config` 可变且**需要跨实例失效**，必须有共享载体，Redis 正好承担"失效广播"。两者载体不同是刻意的，不是不一致。

### 本表不存的东西

- **`quota_bytes` / `used_bytes`**：配额今天没有任何实现（配置里没有、代码里没有校验点），属假想需求；`used_bytes` 更是必须由 `object` 状态变更事务维护的计数器，一旦与事实表不一致就再无可信度，而它本质是可汇总出的派生值。将来真要配额时只加 `quota_bytes`，用量走后台汇总，不引入事务内计数器。
- **`upload_session_ttl_seconds`**：分片会话过期是厂商/协议约束（BOS/OSS 的 multipart upload 有自身上限，超时必须 `AbortMultipartUpload`），不是按 app 变化的业务策略。统一放代码/服务配置，由 GC 按它扫 `upload` 表。

### 代码侧义务

1. **`store.keyword` 不可修改**：store 的更新接口不接受 `keyword` 字段；换桶 = 新建 store 行 + 重配 app_config。引用改成 keyword 之后，改名会静默断开绑定，表现为该 app 突然找不到桶，而 DB 拦不住 UPDATE。爆炸半径从"几个应用配置"扩大到"亿级对象行的桶指针"，见第 4 节。与它成对的那一半——**行不可删除**——是下一条。
2. **写入时校验 keyword 存在**（替代 FK 的存在性检查）。
3. **本表没有删除路径**（2026-10-13 定）：代码不提供删除接口，也不写任何 `DELETE FROM store`；下线一个桶的动作是 `status=0`（见第 1 节"行只增不减"）。这条**取代**了原来的"删 store 前反查三处"（`app_config` / `object` / `export_task`）——没有删除动作，就没有删除前的检查，`app_config` 的 `idx_app_config_store` 因此删除。引用完整性从此由"**行永不消失**"承担，不再依赖删除时的反查；DB 层删不掉的这半条约束，只能靠"代码里根本没有那个入口"来兑现。
4. **启动自检扩一条**：`app_config` 引用的每个 keyword 必须存在且能构造出客户端，否则拒绝启动。**"全量加载"含 `status=0` 的停用行**（第 1 节"行只增不减"）——漏掉它们，该 store 名下的存量 URL 会整片 404。没有 FK 之后孤儿引用只会在第一个上传请求时才暴露，所以这条和 `store` 那节的"CHECK 移除后唯一兜底"是同一堵墙。
5. 值域校验（`status` ∈ {0,1}、各 TTL 非负、后缀格式）全部在代码，DDL 不写 CHECK。
6. **写本表的接口必须执行延迟双删**（见"缓存策略"）：更新行 → 删 `resource:app_config:{app}` → 延迟 500ms 再删一次。这条与"`store.keyword` 不可修改、`store` 行不可删除"同属代码义务——DB 拦不住，漏做只会表现为"配置改了不生效 / 延迟生效"，排查时看不到异常。**触发点只有 `Update` 一个**：本表**不提供删除接口**（2026-10-13 16:33 定，与 `store` 同立场），下线一个 app = `status=0`、走 `Update` 写动词，因此自动带上双删；DB 拦不住 `DELETE`，只能靠"代码里根本没有那个入口"兑现。**前提**：下线不能是一条裸 `UPDATE`——绕过双删会让已停用的 app 在缓存里继续以 `status=1` 存活到 TTL，而 `status=0` 是上传闸门。

---

## 3. `directory` — 应用目录树

```sql
CREATE TABLE `directory` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `app`        CHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '应用标识',
  `name`       VARCHAR(64)  NOT NULL COMMENT '目录名',
  `path`       VARCHAR(660) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '物化路径',
  `accept`     VARCHAR(255) NOT NULL COMMENT '允许后缀',
  `max_size`   INT UNSIGNED NOT NULL COMMENT '单文件上限',
  `updated_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最后修改人',
  `created_at` BIGINT UNSIGNED NOT NULL COMMENT '创建时间',
  `updated_at` BIGINT UNSIGNED NOT NULL COMMENT '修改时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uidx_directory_app_path` (`app`, `path`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC COMMENT='应用目录树';
```

### 说明

- **字段 `app`**：应用标识，沿用 manager 的 `app.keyword`，`CHAR(32)` + `utf8mb4_bin` 与 `app_config.app` 逐字一致。它是本表所有查询的第一限定词——现网 `GetDirectory`（`internal/infra/dbs/directory.go:28-31`）只按 `id` 查，A 应用可以读到 B 应用的目录，本次修掉。
  选字符串而不是 `app_id BIGINT` 引用 `app_config.id`，代价已经算过：`directory`/`file`/`object`/`upload`/`export` 每张含 app 的索引每行多背 128 字节（`CHAR(32)` × 4）vs 8 字节。收益是查询链路不需要多一次 join 或 id 映射、应用迁移/重建时数据面表不用改、以及 `uidx_directory_app_path` 这种跨列表述在导出 SQL 和人工排查时直接可读。**注意这多出来的 128 字节已经算进下面 `uidx_directory_app_path` 的 2770 里**，如果换成 `app_id`，这张唯一索引就有 2770-120=2650 字节的余量，也就失去了"必须限深 10"这条硬约束的来源。
- **不设 `parent_id`（本次删除）**：父目录完全由 `path` 推出——`/图片/头像` 的父级就是 `path` 去掉最后一段（`/图片`），根的父级是 `""`。它原本只服务"列出直接子目录"这一种查询，而**现网这条查询也没有真的用到它**：`ListDirectory`（`internal/app/directory.go`）一次取回全量目录、在内存里用 `tree.BuildArrayTree` 按 `ParentId` 拼树。也就是说 `parent_id` 在当前代码里是一个**只写不读**的冗余列，却带来一个真实的一致性风险：移目录时只重写了子树 `path`，`parent_id` 会漂移（见下方"移目录"）。删掉它，这类漂移在结构上不可能再发生，也不需要第二条 UPDATE 去维护它。
  代价：对外 API 的 `parentId` 字段改成从 `path` 派生（取最后一段之前的父路径，再按 `uidx_directory_app_path` 反查 id，或直接在内存树里用路径索引回填）；"列出直接子目录"由 `path` 前缀 range 或内存树完成，不再单独建索引。这与本表的设计原则一致——**不存能由 `path` 推出的结论**。
- **字段 `name`**：目录名单独一段，最长 64 字符（一个汉字算 1 个字符，VARCHAR 长度按字符计）。**值域五条**：非空；≤64 字符；不含 `/`；不是 `.` / `..`；不含控制字符（`< 0x20`、`0x7F`）与**四字节字符**（`rune > 0xFFFF`，含绝大多数 emoji；2026-10-13 16:55 定）。**≤64 这条不是审美约束**——它是 `path` 列长成立的前提（`10×64 + 10 = 650 ≤ 660`），漏了它"限深 10 保证 `path` 不溢出"就是空话。**四字节字符按码点禁、不按语义禁**：`rune > 0xFFFF` 机械可判且不随 Unicode 版本腐化，代价是 BMP 内的符号类字符（`☀` U+2600 等 3 字节字符）仍在允许范围内。理由见 04-directory §6。
  **值域（代码校验，不写 CHECK）**：非空且去首尾空白后非空；**不含 `/`**——它是 `path` 的分隔符，一旦允许，`path` 就不再能唯一地切回层级，"先定位最深已存在层"、子树前缀匹配、防环判断会同时失效；**不是 `.` 或 `..`**（与路径语义冲突）；**不含控制字符**（`< 0x20`、`0x7F`——它们会破坏 URL / JSON / 日志）。
  **`%` / `_` / `\` 允许**：它们是合法的文件名字符，误匹配问题由**查询侧转义**解决（见"路径前缀查询必须转义"），不靠禁用字符回避。
- **字段 `path`**：物化路径，形如 `/图片/头像`，每级 = 一个 `name` 加一个 `/`。**`utf8mb4_bin` 是"同级 `A` 与 `a` 允许共存"的实现方式**（现网同级唯一性靠 `UNIQUE(parent_id, name)`，改成路径唯一后大小写敏感这个属性必须显式声明，否则会继承表级 `_ci` 排序规则、静默变成大小写不敏感）。
  **限深 10**：`10×64 + 10 = 650 ≤ 660`，所以 `path` 的长度上限完全被深度上限包住，唯一的溢出错误就是 `PathTooDeep`——不再需要区分"名字太长"和"层级太深"。**不做静默截断**：截断会让两个不同目录落到同一个 `path`，报错变成难以归因的 1062 重复键。
- **字段 `accept` / `max_size`（快照式，见 `app_config` 的"继承链"）**：新建目录时从当时的 `app_config.default_accept` / `default_max_size` 抄一份写死进本行，之后独立演化。**取值约定与 `app_config.default_*` 逐字相同**（2026-10-13 定）：`accept` 用 `'*'` = 全部、`''` = 不允许任何类型；`max_size` 值域 `1 .. 4294967295`（**`0` 非法**），`4294967295` = 不限。两级同规、`NOT NULL` 且不给 DEFAULT，**不存在"未设定"这个态**。运行时上传校验**只读本行**，不回读 `app_config`。
  连带必须改的一处：现网 `UpdateDirectory` 用 `ctx.DB().Updates(struct)`（`internal/infra/dbs/directory.go:71`），gorm 只写非零字段，把 `accept` 清成 `''`、`max_size` 改成 `0` 会静默失效（用户以为改成功了）。更新路径必须显式指定列（`Select("accept", "max_size").Updates(...)`）。**这条在新语义下更要紧**：`''` 现在是"不允许任何类型"、`0` 是非法值，都是用户会**主动设置**的值，不再是"清零"；而 gorm 的零值跳过恰好把这两个都挡掉，表现为"设了不生效"。
- **字段 `updated_by`**：最后修改人的用户 ID（manager 侧 `user.id`，`0` 表示非人工变更），与 `store.updated_by` / `app_config.updated_by` 同规范。
  它与那两张表同属"配置面审计"，但理由不同：那两张表加它是因为 **`sk` 进库、改密钥要追责**；本表加它是因为**目录的改名与移动是跨用户的管理动作**——目录是共享的，动它的人未必是建它的人，而本表**没有任何别的"人"字段**，不加就完全无法追责。`file` 表**不加**这一列，理由见第 6 节"本表不存的列"。
- **字段 `created_at` / `updated_at`**：Unix 秒，代码写入，故意不给 DEFAULT，与 `store` / `app_config` 同规。
- **索引 `uidx_directory_app_path`**：承担三件事——① 按路径直查（`internal/domain/service/file.go:69-96` 每次上传都按 `strings.Split(*dirPath, "/")` 解析目录，是热路径）；② 并发隐式建目录的**唯一**去重手段（配合 `INSERT IGNORE` 后重查，不引入分布式锁）；③ 子树前缀 range（`WHERE app=? AND path LIKE '/图片/%'`，`app` 等值 + `path` 前缀，走索引起扫；**`LIKE` 必须转义，见"路径前缀查询必须转义"**）。
  条目长度 `32×4 + 660×4 + 2 = 2770 < 3072`（InnoDB 唯一索引上限），装得下；这也是上面"限深 10"能成立的算术前提。**3072 这个上限只在 `ROW_FORMAT=DYNAMIC` + `innodb_large_prefix=ON` 下成立**，所以本表显式声明 `ROW_FORMAT=DYNAMIC`——5.7.9 起两者都是默认值，但"默认"不是承诺，写出来才不会因为一个老实例或一次参数调整变成 767 字节上限（那样 2770 的索引根本建不起来）。其余 7 张表的索引都远小于 767，不需要这一句。
- **不设 `idx_directory_app_parent`（本次删除）**：它原本只给"列子目录"用（`WHERE app=? AND parent_id=?`）。删掉 `parent_id` 之后这条查询改由 `path` 前缀 range（`WHERE app=? AND path LIKE '/图片/%'`，注意要排除自己那一行）或内存树完成，`uidx_directory_app_path` 已经覆盖，不需要第二棵索引。顺带也解决了原设计里"同级 `A` / `a` 允许共存、所以不能加 `UNIQUE(parent_id, name)`"这个别扭点——现在同级唯一性由 `path` 唯一性独家保证，大小写敏感由列级 `utf8mb4_bin` 显式声明，不再有第二套约束来打架。

### 三个 CRUD 语义

- **删目录 = 删子树**：`WHERE app=? AND (id=? OR path LIKE ?)`（第二个参数为 `escapeLike(path) + '/%'`，**必须转义**，见"路径前缀查询必须转义"），一条语句。现网 `DeleteDirectory`（`directory.go:76-85`）依赖闭包表取子 id，而闭包表 0 行，所以**只删了自己**，子目录变成挂在不存在父级上的孤儿。
- **移目录 = 重写子树 path**：一条 `UPDATE ... SET path = CONCAT(新path, SUBSTRING(path, CHAR_LENGTH(旧path)+1)) WHERE app=? AND (id=? OR path LIKE ?)`（前缀参数同样用 `escapeLike(path) + '/%'`）。这是物化路径的代价项（闭包表在这里是免费的），但本服务移动目录极少发生，而删除和按路径查是常态。
  **必须用 `CHAR_LENGTH`，不能用 `LENGTH`**（2026-10-13 定）：`SUBSTRING(str, pos)` 的 `pos` 对非二进制串是**字符**位置，而 `LENGTH()` 返回**字节**数。`/图片` 是 3 个字符 7 个字节，于是 `LENGTH(旧path)+1 = 8`，`SUBSTRING('/图片/头像', 8)` 从第 8 个字符开始取——而串只有 6 个字符，**直接返回空串**，整棵子树的 `path` 被截成裸的新前缀。这是**只在含多字节字符的路径上才发作的静默错误**，中文路径的生产环境必然踩到。两个函数都是字符语义时才配对。
  **目标 `path` 已存在则拒绝移动，不做覆盖**（2026-10-13 定）：新 `path` 撞 `uidx_directory_app_path` 会报 1062，把它翻译成明确的"目标位置已存在同名目录"即可，**不重试、不合并、不自动改名**。理由与"新建目录不做静默截断"一致——自动改名会让用户以为搬到了 A、实际落在 B；合并则要先定义"同名目录怎么合、同名文件怎么办"，那是另一个功能。注意这条 UPDATE 是**整棵子树一条语句**，所以它要么全成功、要么因 1062 全失败，不存在"半个子树搬过去"的中间态。
  **删除 `parent_id` 让这条语句第一次真正成立**：这条 UPDATE 的 WHERE 同时覆盖"节点自己"和"整个子树"，而 `SET parent_id` 只对节点自己成立、对子树里每一行都错——旧设计若在这一条里带上 `parent_id`，就会把整棵子树的父指针全改成新父级；若拆成两条语句维护，则一旦第二条失败就永久漂移。删列之后，移目录天然是**一条自洽的语句**，没有需要额外维护的派生状态。
  **防环**：新父级的 `path` 是自己的 `path` 的前缀 → 拒绝。现网只判 `id == parent_id`（`directory.go:54-56`），A→B→A 这种两级环可以直接写进去；改成路径前缀判断后，任意深度的环都被一条规则挡住。
- **按 id 取祖先链 / 子树**：祖先链用 `path` 拆出前缀集合再 `path IN (...)`；子树用上面的前缀 range。都不需要递归 CTE，因此不要求 MySQL ≥ 8.0。
  **这句话对全库的约束不止于本表**：8 张表一律不写 8.0 专属语法——无 CTE、无窗口函数、无 `utf8mb4_0900_*` 排序规则。表级排序规则统一用 **`utf8mb4_unicode_ci`**（UCA 系、同为 `ai_ci`，是 5.7 可用的排序规则里与 `utf8mb4_0900_ai_ci` 语义最接近的；`utf8mb4_general_ci` 虽更常见但不是完整的 accent-insensitive）。**换排序规则不影响任何唯一性判定**——所有唯一键都显式声明了列级 `utf8mb4_bin` / `ascii_bin`，不依赖表级排序规则；表级排序规则只影响 `ORDER BY` 与人工写 SQL 时的等值比较。

### 路径前缀查询必须转义（`LIKE` 元字符）

子树查询用 `path LIKE '前缀/%'`，而 `_` / `%` / `\` 在 `LIKE` 里是元字符：

- `_` 匹配任意**单**字符 → 目录 `图_片` 的前缀 `'/图_片/%'` 会命中 `/图A片/...`；
- `%` 匹配任意**多**字符 → 目录 `图%片` 的前缀会命中一大片无关目录；
- `\` 是默认转义符本身。

后果是**误删 / 误移**：删目录和移目录用的是同一条前缀谓词，多匹配到的目录会被一起删掉或一起搬走。`app` 限定了范围所以不会跨应用，但同一应用内的误伤已经足够严重。

**处置**：所有子树前缀查询一律走同一个转义函数，并显式声明转义符：

```
pattern = escapeLike(path) + '/%'        -- escapeLike 依次：\ → \\，% → \%，_ → \_
WHERE app = ? AND path LIKE :pattern ESCAPE '\\'
```

- **先转 `\` 再转 `%` / `_`**：顺序不能反，否则新加的转义符会被再转一遍。
- **`ESCAPE` 必须显式写**：不写就依赖 MySQL 的默认转义符，还要看 `NO_BACKSLASH_ESCAPES` 之类的模式开关，不能靠隐式默认。
- **索引不受影响**：`LIKE` 的常量前缀（转义后仍是字面量）照常让优化器取 `uidx_directory_app_path` 的范围扫描，`ESCAPE` 不破坏它。
- **涉及位置**：**删目录**、**移目录**、**列直接子目录**（也可用内存树，见 `parent_id` 删除说明）、以及第 2 节里"按 `path LIKE '前缀%'` 批量回溯"的离线脚本。

**为什么不改用范围查询**（`path >= :prefix AND path < :prefix_next`）：理论可行——`utf8mb4_bin` 按码点比较，`'/'`(0x2F) 的下一个字符是 `'0'`(0x30)，所以 `path >= '/图片/' AND path < '/图片0'` 恰好圈住子树。但它依赖"分隔符恰好小于任何可能的段首字符"这个隐式事实，一旦分隔符或排序规则变动就**静默**出错。转义只依赖 `LIKE` 语义，更直白。

**为什么不干脆禁掉 `%` / `_`**：它们是合法文件名，禁掉是拿用户体验换实现省事；而且 `\` 作为转义符仍要处理，问题不会消失。

### 自动建目录（保留行为，五条护栏）

按路径上传是客户端的主要入口，改成强制预建目录会把设计压力推给接入方，所以保留"路径不存在则逐级建目录"，但把现网 `GetDirectoryLimitByPath`（`internal/infra/dbs/directory.go:135-152`）的逐层 `FirstOrCreate` 重写为：

1. **限深 10**，超限返回 `PathTooDeep`（这条同时也是 `path` 列长的保证）。
2. **先定位最深已存在层**：`SELECT id, path FROM directory WHERE app=? AND path IN (前缀集合)`（前缀集合在内存里按 `/` 切出来，最多 10 个值），一次查询拿到起点，**不再每层一次 SELECT**。
3. **从那一层往下逐层 INSERT**，并发靠 `uidx_directory_app_path` + `INSERT IGNORE`，忽略掉重复键后重查即可得到别人刚建好的行；不加锁、不依赖事务串行化。
4. **新建行的 `accept` / `max_size` 取自 `app_config`**，且 `app_config` 缺行时整次上传直接报"应用未配置"，**不回落到服务级默认**（`conf.DefaultAcceptTypes` / `conf.DefaultMaxSize` 这两个全局兜底随快照式一起下线）。抄之前先过一遍值域校验——`accept` 只能是 `'*'` / `''` / 后缀白名单，`max_size` 必须落在 `[1, 4294967295]`，**非法即拒绝整次上传**。快照式的含义是"抄一份当下**合法**的策略"，不是"抄一份什么进来都行"。
5. **每个段先过 `name` 值域校验**（非空、**≤64 字符**、不是 `.`/`..`、无控制字符、**无四字节字符 `rune > 0xFFFF`**，见字段 `name`）；**任一段不合法则整次请求拒绝**，不做部分建目录——半个子树落地比直接失败更难收拾。

现网这段代码还有一个隐式写入的老问题：`Attrs(entity.Directory{Accept: strings.Join(conf.DefaultAcceptTypes, ","), ...})`（`directory.go:145-146`）把**全局配置**当默认值写进目录行，所以历史上自动建出来的目录携带的是服务级默认、与所属 app 无关；本表加了 `app` 之后，这个默认值必须来自该 app 的 `app_config` 行。

### 连带删除的列

- `entity.Directory.ParentId` 随列一起删除，同时下线这些依赖它的代码：`ListDirectory`（`internal/app/directory.go`）里 `tree.BuildArrayTree` 的 `ParentId` 入参改为按 `path` 前缀在内存里建树；`CreateDirectory` / `UpdateDirectory`（含移动）不再接收、不再写入 `parent_id`；`dbs` 层 `GetDirectory` 的 `WHERE id=?` 补上 `app=?`（现网跨应用可读，见字段 `app` 说明）。对外响应里的 `parentId` 改为从 `path` 派生回填，保持接入方视角不变。
- `entity.Directory` 里的 `Accept`/`MaxSize` 语义不变，但 `GetDirectoryLimitByPath` 返回的 `DirectoryLimit` 结构体不再需要携带服务级默认值兜底——`conf.DefaultAcceptTypes` / `conf.DefaultMaxSize` 随快照式一起下线，`app_config` 缺行就是错误，不是回落。

---

## 4. `object` — 物理对象

一行 = 某个桶里真实存在的一份字节。这张表把现网 `file` 表同时承担"业务文件"和"物理对象"两件事拆开：去重、秒传、断点续传、回收全部只看 `object`，而归属、名字、目录位置全部在 `file`。

```sql
CREATE TABLE `object` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `store_keyword` CHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '存储后端标识',
  `md5`           CHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL COMMENT '内容摘要（分片摘要的摘要）',
  `ext`           VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '' COMMENT '后缀',
  `size`          BIGINT UNSIGNED NOT NULL COMMENT '对象大小',
  `created_at`    BIGINT UNSIGNED NOT NULL COMMENT '创建时间',
  `updated_at`    BIGINT UNSIGNED NOT NULL COMMENT '修改时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uidx_object_store_md5` (`store_keyword`, `md5`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='物理对象';
```

### 核心不变式

> **展示路径需要的信息全部在 URL 里，`object` 表只服务写侧（去重判定）和回收侧（GC、下线反查）。公共读一次库都不查。**

配套三处表述同一个事实，互不引用、互不复制：

| 位置 | 存什么 | 形态 |
|---|---|---|
| 桶里 | 对象名 | `{md5}.{ext}` |
| `object` 行 | 事实 | `store_keyword` + `md5` + `ext` + `size` |
| 接入方库里 | 对外标识 | **完整 URL** `{public_base_url}/{md5}.{ext}` |

**代价**：公共读意味着这一层没有访问控制——对象对**任何知道其 md5 的人**可读。这是显式接受的决策，见下方"公有读 = 无访问控制"。

### 公有读 = 无访问控制（P0-4 的显式决策）

**事实**：对象键是 `{md5}.{ext}`、URL 是 `{public_base_url}/{md5}.{ext}`，而 md5 是**公开可算**的——任何拿到过这份文件的人都能算出它的 URL。`cdn_auth_type = 'none'` 的 store 上，这个 URL 不需要任何凭据即可读。

**这是被接受的决策，不是疏漏**：内容寻址与"文件名不可猜"这层弱保护**互斥**。去重要求键由内容唯一决定；键一旦由内容决定，知道内容的人就能推出键。本表把"读路径 0 查询"当作核心收益，代价就是放弃对象级访问控制。

**边界（写清楚，避免误读）**：

| 档位 | 行为 |
|---|---|
| `cdn_auth_type = 'none'` 的 store | 对象对**任何知道其 md5 的人**可读。md5 不是口令，是可以公开计算的摘要，不能当密钥用 |
| `cdn_auth_type != 'none'` 的 store | 读仍走签名 URL（`temp_url_ttl`），**签名是这一档唯一的访问控制**，本决策不影响它 |
| 秒传路径 | 与公有读叠加后，`PrepareUpload` 的命中/未命中构成"该 md5 是否已存在于该 store"的**存在性 oracle**。已随 P0-3 显式接受。**注意它泄露的是"某份内容存不存在"，不是"能不能读到它"**——后者在 `cdn_auth_type='none'` 时本来就成立（见上两行） |

**由此产生的三条硬规则**：

1. **敏感对象不得放进公有读 store**。判断依据是"这份内容是否允许任何知道其摘要的人读取"，而不是"这份内容是否重要"。要放敏感内容 → 给该 app 配 `cdn_auth_type != 'none'` 的 store（配置点只有 `app_config.store_keyword`），读走签名。
2. **不得为了"让人猜不到"往对象键里编信息**。键里加 app / user / 随机段会让去重失效、让键不再能由 `(md5, ext)` 推导（见"对象命名规则"），却换不到任何真实的访问控制——**键的保密性不是访问控制**。
3. **URL 一旦发出即等于交付**：拿到 URL 的人可以转给第三方。撤销只有配置面手段（给该 store 打开 `cdn_auth_type`、或换 store 改 host），且受 CDN 缓存与已发外链影响——**已发出的 URL 基本不可撤回**。这与"对象名和键格式是对外承诺"（连带义务 2）是同一条约束的两面。

**不做的**：不加对象级 ACL 表、不加权限位、不强制所有读走签名。理由是本表的全部价值（去重、秒传、0 查询读）都建立在"键由内容唯一决定且可公开解析"之上；把它当作**部署期的配置责任**（选对 store），而不是运行时的鉴权。

### 说明

- **字段 `store_keyword`**：全库 store 引用统一用 keyword，不用 `store_id`（决策与代价见第 1 节 `id` 字段说明）。它回答"这份字节躺在哪个桶"，而不是"哪个 app 该写哪个桶"——后者属于 `app_config`。
- **字段 `md5`**：**内容摘要——不是整文件 MD5，而是"分片摘要的摘要"**。算法：按服务端下发的 `chunkSize` 切片 → 逐片取 MD5 → 把 N 个 32 位小写 hex **按片号升序拼成一个长串** → 再对这个串取一次 MD5（单片路径 N=1，即"对文件 md5 再取一次 md5"）。完整定义、理由与核验落点见 **`设计方案/06-upload.md` §4.2**；本节只固定与库有关的三件事：**32 位小写 hex**（`CHAR(32)` + `ascii_bin`）、**由内容唯一决定**、**客户端算、服务端在对象诞生之前核**。
  **列名为什么仍叫 `md5`**：它命名的是**算法**（MD5 的输出），不是"整文件 md5"这一种具体形态。改名（`digest` 之类）要动 5 个列 + 唯一索引 + proto 字段 + 全库散文，收益只是字面更贴切，故**保留 `md5`、把描述写准**。但**不许再把它描述成"整文件 MD5"或"文件内容 MD5"**——那才是真正的不诚实。
  **语义**：它是**去重键，不是完整性证明**（MD5 可构造碰撞，不能当防伪依据）。它原本宣称的作用是"防链路截断"，**这个作用仍然成立且更强**——按片核验能定位到**是哪一片**坏了，而不是只知道整体不一致。**"客户端声明的摘要与服务端算的一致"这句现在可执行**：服务端在 `CompleteMultipartUpload` **之前**独立重算并比对（`06-upload.md` §7.1 判据 4、§5.1），不符即中止、**不留对象**。将来真要内容寻址时新增 `sha256` 独立列，**不要改这一列的含义**。
  **命名约定**：内容摘要列一律命名 `md5`。`sha256` 是另一个算法（未来可选增强），不与 `md5` 混为一谈。
  **字符集口径（P1-9 定稿，2026-10-13）**：全库的 `md5` / `ext` 一律用 **`ascii_bin`**（`object.md5` / `object.ext` / `upload.md5` / `upload.ext` / `export_task.md5` 五处）。理由有两条，缺一不可：① **值域本来就只有 ASCII**——`md5` 是 32 位小写 hex，`ext` 归一化后只允许 `[a-z0-9]`（见"对象命名规则"细则 2），用 `utf8mb4` 存它们是纯浪费：`CHAR(32)` 从 128 字节降到 32 字节；② **`_bin` 是必须的**，因为大小写必须区分（`md5` 一律小写、`ext` 一律小写，若用 `_ci` 则 `jpg`/`JPG` 会被判成同一个值，与"归一化"这条前置约定打架）。收益落在索引上：`uidx_object_store_md5` 从 256 字节降到 160 字节。
  **这条是"值域已知为 ASCII"的一个特例，不是全库规则**：`store_keyword` / `app` / `path` / `name` 这些含中文或需与 manager 逐字对齐的列**保持 `utf8mb4_bin`**，不动。
- **字段 `ext`**：对象名的后缀段，不含点，取自首个写入该内容的上传方文件名后缀（归一化规则见"对象命名规则"）。它是**首写者的事实**，不是当前引用者的事实，因此一旦写入就不再变更（秒传命中复用行时绝不覆写）。`Content-Type` 由代码从 `ext` 生成，并在 **`CreateMultipartUpload`（所有路径，含 `chunkCount=1` 的单片）时**写进厂商对象元数据——**所有上传（含 `chunkCount=1` 单片）只有 Initiate 这一次机会**（`UploadPart` 不接受元数据、`Complete` 改不了它，见 `06-upload.md` §7）。字符集用 **`ascii_bin`** 与 `md5` 一致（口径见上方字段 `md5` 的"字符集口径"），避免 `jpg`/`JPG` 在同一路径规则下判成两个对象。**`ext` 不参与唯一索引**：去重按内容（`(store_keyword, md5)`），后缀只影响外观和厂商侧规则，不参与身份判定。
- **字段 `size`**：**单位字节，类型 `BIGINT UNSIGNED`**。程序写入的度量按规矩用字节；`INT UNSIGNED` 上限 4 GB，而这张表天生服务分片上传的大文件，会溢出。现网 `file.size` 是 KB、`entity.File.Size` 是 `uint32`，统一到字节 + BIGINT 后，现网唯一的换算点 `pkg.GetKBSize`（`internal/pkg/pkg.go:15-17`）随之删除。
  **API 侧的对应字段（proto `size`）声明为 `string`**——32 位字节只有 4 GB，会把这个上限强加给全系统，与本列用 BIGINT 的意图冲突；理由与三层换算见文首"类型分层"。（proto `chunkSize` 保持 `uint32`，它的上界推导见同一节。）
  **本列只允许写入服务端证实的字节数**（来自实际收到的字节，或厂商 `HeadObject`），**不接受客户端声明的 `size`**。原因是它同时是秒传的比对基准：基准本身若可被客户端声明污染，秒传校验就退化成"两个声明互相比对"。详见下方"size 校验"。
- **没有 `app` 列**：去重边界是 **store 而不是 app**。真正决定"字节存在哪里、谁付成本、走哪个域名、在哪个区域"的是 store，`app` 只是"该往哪个 store 写"的配置来源。所以同一张图被 5 个 app 使用、而这 5 个 app 配同一个桶时，桶里是**一份**字节、`object` 是**一行**、URL 是**同一个**——这正是去重该省的东西；不同桶各自一份是物理必然，不是重复。业务归属仍然在 `file.app`。
- **没有 `object_key` 列**：这里有一条等价式要说明白——**"不存 `object_key`" 等价于 "对象名必须能由表里的列完整推导"**。既然对象名定为 `{md5}.{ext}`，那就必须把 `ext` 存成独立列（就是上面那条），而不是存整条键：键里的桶段、日期段一旦出现，`object_key` 就从"推导结果"退化成"第二份真相"，行里的名字和桶里真实的对象对不上时没有任何自动检查能发现。
- **没有 `etag` 列**：分片上传的**对象** ETag 是 `md5-of-md5s-N` 形式（二进制摘要拼接 + `-N`），**不等于本表 `md5` 列的值**（那是 hex 文本拼接后再取一次 md5），不能拿来当身份；单片（`chunkCount=1` 的 1 段多段上传）的对象 ETag 为 `md5(file)-1` 形式，而本表存的是"对文件 md5 再取一次 md5"，仍然不同。**注意区分**：分片**级**的 ETag（`ListParts` 返回的每个 part 的 ETag）**确实等于该分片内容的 MD5**，服务端用它重算摘要（见第 5 节"内容摘要的算法"）——但它是**瞬时消费**的中间值，不落库。
- **没有 `ref_count` 列**：引用回收走**纯扫描**——删 `file` 时 `SELECT 1 FROM file WHERE object_id=? LIMIT 1`，无引用才删对象。计数器必须由 `file` 的增删在同一事务维护，一旦失真就再无可信度且无法察觉（与 `used_bytes` 被判掉是同一条理由）；而删除是低频路径，多一次索引查询不是成本（**删目录**时是 N 倍放大，这句不适用——批量回收的算法见第 6 节"删目录时的对象回收"）。为此 `file` 表需要 `KEY(object_id)`。
- **没有 `status` 列**：见下面"分片上传"。
- **字段 `created_at` / `updated_at`**：Unix 秒，代码写入，不给 DEFAULT，同规。`updated_at` 在这张表上几乎没有更新场景（对象内容不可变），保留是为了与其余表一致，以及回收任务按时间扫描时有依据。
- **唯一索引 `uidx_object_store_md5`**：`128 + 32 = 160` 字节（`store_keyword` 是 `utf8mb4_bin` 128 字节，`md5` 是 `ascii_bin` 32 字节；P1-9 定稿前这里是 256）。它同时是秒传查询的执行计划（`WHERE store_keyword=? AND md5=?` 全等值命中）和"同桶内不允许两行声明同一个对象"的结构保证。**`ext` 有意不放进这个索引**——身份是内容，不是外观；把后缀放进唯一键就等于接受"同一份字节按后缀分裂成多行、桶里多份对象"。这张表**只有这一个索引**——曾经考虑的 `KEY(md5)` 单列索引（给"外部只存裸 md5 来换 URL"用）已经不需要，因为对外标识改成了完整 URL；跨 app 的"这张图谁传过"属离线审计，扫表够用，不值得让写入热表每次插入多维护一条索引。

### 对象命名规则

`{md5}.{ext}`，一层平铺。两个"不带"各自的理由：

- **不带桶名 / store 段**。桶已由 `store_keyword` 确定，把 `bdemo` 写进 `bdemo` 桶里的对象名，就是现网双桶段 404 的同源错误。store 的定位信息由 **URL 的 host**（`public_base_url`）承担，不进键。
- **不带日期或哈希分层**（`{app}/2026/10/{md5}` 这类）。除了厂商本身按 key 哈希分区、不需要手工分层之外，更硬的理由是：**键里一旦出现时间成分，它就不再能由 `(store_keyword, md5, ext)` 推导**，那就必须把 `object_key` 存进表——"不存键"和"日期分层"这两个选择互斥。

五条细则：

1. **`md5` 一律小写**。摘要计算结果必须归一化成小写再入键，否则同一份字节以两种写法上传会得到两个对象，且 `ascii_bin` 的比较会拦不住秒传命中——表现为去重随机失效。
2. **`ext` 归一化：小写、仅 `[a-z0-9]`、不含点、写入前校验长度**（列宽 16）。现网 `pkg.GetFileType`（`internal/pkg/pkg.go:7-14`）只做 `filepath.Ext` 截取，没有任何字符清洗，后缀直接进公网 URL 等于把文件名里任意一段字节送进对象键。清洗后不合法就落空串，不允许猜。
3. **`ext` 为空串时对象名退化为 `{md5}`**（无后缀，例如服务端生成的导出流）。因此**任何解析对象名的代码都不得用"固定段数"判定**——`strings.Split(name, ".")` 取 `len==2` 是错的，必须用 `filepath.Ext(name)` 这一个入口；同时 `ObjectName(md5, ext)` 构造函数负责在 `ext` 为空时不拼点。
4. **秒传命中即复用行，绝不复写 `ext`**。后缀是首写者的事实：复写会改对象名，而对象名在公网 URL 里，改了就是打断所有已发出的历史外链。后果是同一份字节的外观由谁先传而定，这是内容寻址 + 带后缀键必须接受的代价，也是去重仍然生效的原因（唯一索引 `(store_keyword, md5)` 不含 `ext`，第二个后缀不同的上传方必然命中首写者的那一行、拿到首写者的 URL，桶里只有一份对象）。
5. **显示与下载一律以 `file.name` 为准**，`object.ext` 只是外观。业务名在 `file` 表，一份字节被 A 叫 `logo.png`、被 B 叫 `banner.jpg` 时，两边各自展示各自的名字，只有裸 URL 的末尾段是首写者的后缀。

关于"同内容、多后缀会不会分裂成两个对象"：不会。上一版这里写过"扩展名进键会让去重静默失效"，那个论证是错的——唯一索引不含 `ext`，第二次上传必然命中并复用首写者的行。**代价不在去重，在外链稳定性**（细则 4）。

**重复上传是幂等覆盖，不是冲突**：键由内容决定 ⇒ 同桶同字节的 PUT 落在同一个对象上，覆盖无害。唯一有害的情形是"同 md5 不同字节"（碰撞或客户端报错值），它会静默覆盖掉别处引用指向的对象——所以写前的唯一索引查询命中后**必须比对 `size`**，不一致直接拒绝，不允许覆盖。这是内容寻址命名下唯一的护栏。

### size 校验（两层，P0-3 的处置）

秒传的安全性完全建立在"`object.size` 可信"之上，因此校验必须分两层，缺一层就退化成"两个客户端声明互相比对"：

1. **写入层（真实上传字节的路径）**：服务端在对象落盘前后核对**实际收到的字节数 == 请求声明的 `size`**——**直传路径（含 `chunkCount=1` 的单片）一律用 `ListParts` 各片 `Size` 之和**（字节**客户端直传厂商**、服务端看不到原文，见第 5 节"字节流向"）。不符即拒绝，不入 `object` 行、不返回 Key。这一层的作用是**保证 `object.size` 永远是服务端证实过的事实**，而不是首写者的声明。
2. **秒传层（命中已有对象）**：`请求声明的 size == object.size`，不符即拒绝、不返回 Key。因为第 1 层，此处的 `object.size` 是服务端事实，所以这次比对是"**客户端声明 vs 服务端事实**"。

**残留风险（本轮显式接受）**：两层校验挡不住"**手握真实字节的人**"——他知道目标文件的 `md5` 与 `size`，声明上去就能命中秒传、拿到 URL。但这**不是新获得的权限**：对象键是 `{md5}.{ext}`、URL 公开可读（见上文"公有读 = 无访问控制"），**任何拿到过这份文件的人本来就能自己算出 URL**。所以 size 校验的实际作用是**挡住误命中与碰撞**（大小对不上的自动拒绝），**不是**阻断定向投毒；它真正在守的是"**同一 `md5` ⇒ 同一份内容**"这条等价的廉价实现。

**"同 md5 不同字节"由摘要核验封死，不由 size 封死**：`size` 管"多大"、摘要管"是哪一份"，一份 1024 字节的文件完全可以被另一份 1024 字节的冒充。服务端在对象诞生之前重算摘要（见第 5 节"内容摘要的算法"），不符即中止，**桶里不会出现"挂着别人摘要"的字节**；秒传路径一个字节都不写，不可能污染任何键。

**不引入 PoP**（2026-10-13 更正）：早先这里写"正确做法是 PoP 或取消秒传"，**方向是错的**。PoP（随机 range 挑战）只证明"你手上确实有这份字节"，**证不了"你声明的 `md5` 等于这份字节的摘要"**——攻击者手里就是有 M，你出什么 range 他都能答；而这里的攻击前提恰恰是他**已经有内容**。**唯一能真正封住的是服务端重算摘要**，已落地为第 5 节"内容摘要的算法"。

### 读写路径

**写（去重判定，唯一必须查库的路径，只发生在上传时）**：

1. `app` → `app_config` → `store_keyword`（**客户端不再传 store**，现网 `req.Store` 字段要删——让调用方指定落哪个桶是越权，既能把 A 的数据写进 B 的桶，也能绕过 `app_config` 直接挑走导出桶）。
2. `SELECT id, size, ext FROM object WHERE store_keyword=? AND md5=?`（`md5` 是"分片摘要的摘要"，客户端先经取片大小接口算出）。命中且 `size` 相符 → 秒传，复用该行（含其 `ext`，见命名规则细则 4）；命中但 `size` 不符 → 拒绝；未命中 → **先核验摘要**（服务端重算、与声明比对，见第 5 节"内容摘要的算法"），通过后才由 `(md5, ext)` 构造对象名、写入对象、INSERT，**写入前同时校验"实际收到的字节数 == 请求声明的 `size`"**（见下方"size 校验"第 1 层）。**核验必须在写入之前**——顺序反了就会先覆盖掉键上既有的合法内容。

**读（0 查询）**：

| 场景 | 怎么解析 | 查库 |
|---|---|---|
| 浏览器直接展示图片 | CDN 按域名的回源配置打到固定桶，path 就是对象名 | 0，不经过服务 |
| 生成临时签名 URL | URL host → 启动时全量加载的 store map → 该 store 的 `cdn_auth_type`/`cdn_auth_key`；需签名时再取 app 的 `temp_url_ttl`（**Redis 缓存**，见第 2 节"缓存策略"；app 来自请求上下文 `X-App-Key`） | 0 次 DB（命中 Redis 1 跳；`cdn_auth_type=none` 时连 Redis 都不需要） |
| 服务代理字节 | 同上，host 定 store，path 定对象名 | 0 |

**服务内部的 `file` 列表出 URL**：两种作用域各走各的索引——目录内 `WHERE app=? AND directory_id=? LIMIT 20`（`idx_file_app_directory`）、"我的文件" `WHERE app=? AND user_id=? LIMIT 20`（`idx_file_app_user`）；再 `SELECT id, store_keyword, md5, ext FROM object WHERE id IN (...)`（PK 批量）。一页两条 SQL，第二条走主键。不给 `file` 冗余存键——一屏的开销换"不存在两份真相"。

**一条必须守住的不变量**：内容不能落到不由上传方控制的桶里。所以去重查询的作用域必须是"上传方 app 当前配置的 store"，**禁止**全表按 md5 找。少了这一步就会出现"B 上传的图秒传成功、内容实际躺在 A 的桶和 A 的域名下"——成本、区域、外链归属全部错位，而排查时看不出任何异常。

**读路径以 `object` 为准，不以 `app_config` 为准**：`app_config.store_keyword` 是"现在写去哪"，`object.store_keyword` 是"当时写去了哪"。混用的后果是 app 一换桶，所有历史图片链接立刻断。这也是要求外部存**完整 URL**（含 host）而不是裸对象名的原因——裸对象名没有 host，就只能拿当前配置去猜历史位置。

### 分片上传

**字节流向（2026-10-13 改）**：**全部直传厂商**（含 `chunkCount=1` 的单片）。所有上传服务端只做控制面——建会话、下发**预签名分片 URL**、查 `ListParts`、发 `Complete`；**上传字节从不经过服务端**。理由与统一直传的取舍见 `设计方案/06-upload.md` §5（核心：任何文件的带宽都不该由本服务承担；统一后核验全部在 `CompleteMultipartUpload` 之前用 `ListParts` 的 ETag 完成）。

不需要 `object.status`，因为 **`object` 行只在 `CompleteMultipartUpload` 成功后才插入**，进行中的状态全部由 `upload` 承载：

| 阶段 | 谁执行 | 厂商侧 | `object` 行 | 状态在哪 |
|---|---|---|---|---|
| InitiateMultipartUpload | **服务端** | 分配 uploadId，无对象 | 无行 | `upload` 行存在 = 会话活着 |
| UploadPart × N | **客户端直传** | 分片暂存 | 无行 | 厂商侧（`ListParts` 是权威，库里不记进度） |
| CompleteMultipartUpload | **服务端**（先核验） | 对象此刻才存在 | **此刻 INSERT** | 结束，此后"有行 = 可读" |
| Abort / GC 超时 | **服务端** | 分片释放 | 从未有过行 | `upload` 行删除 |

断点续传同样归 `upload`：客户端带着 `upload_id` 回来 → 按 `uidx_upload_upload_id` 命中会话 → 返回 `ListParts` 的已传分片列表。**客户端若丢了 `upload_id` 就新建会话**，旧的交给 GC，不做"按 md5 找回"（理由见第 5 节"为什么没有 app 列"）。现网把这件事塞进 `file.status` + `file.upload_id`（`entity/file.go:15-16`），后果是 `file` 行在 `PROGRESS` 期间就存在于业务表里，于是**每个读接口都必须自己挡它**——`internal/domain/service/file.go:92-94` 那句 `if res.Status != StatusCompleted { return NotExist }` 就是给半成品打的补丁；同时僵尸 `PROGRESS` 行无人回收。拆表之后这两件事在结构上消失：`object` 里根本没有半成品可以挡。

并发竞态是内容寻址最干净的地方：两个请求同时传同一份字节到同一个桶，两次 Complete 写同一个键（`{md5}.{ext}`）→ 物理对象覆盖一次且字节相同，无害；两条 INSERT 一条成功、一条撞 `uidx_object_store_md5` → 撞的那方按"已存在"处理、复用赢家的 `object.id` 和赢家的 `ext`（**注意此时不要用请求自己的后缀重新拼一次 URL 返回给调用方，要用赢家的行**）。**不需要锁、不需要 upsert、不需要占位再回填。**

**合并时的 size 校验与摘要核验**：直传路径的 `object.size` 同样必须来自服务端事实——各片长度取自 `ListParts` 的 `Size` 之和（或合并后对对象做一次 `HeadObject`），与请求声明的 `size` 比对，不符则 `Abort` 且不建行。这是上方"size 校验"第 1 层在直传路径上的落地。

**摘要核验同样在这里、且必须在 `CompleteMultipartUpload` 之前**：服务端把 `ListParts` 的分片 ETag（按片号升序、剥引号取 hex）拼成长串再取一次 MD5，与会话声明的 `md5` 比对；不符即 `Abort`、报错。**顺序不能反**——先 Complete 就等于用"客户端声明的 `md5`"为键往桶里写了一份字节，若该键已被合法内容占用（投毒场景），这次 Complete 就是一次覆盖。完整论证见 `06-upload.md` §7.2。

### 本表决策带来的连带义务

1. **对象名由唯一的构造函数产生**：`ObjectName(md5, ext string) string`（`ext` 为空时不拼点），配单测锁死。现网缺陷的根因不是"键里编了信息"，而是两处各自拼 key（`internal/domain/service/file.go:53-55` 拼 `{store}/{md5}.{ext}`，`internal/app/file.go:377` 的 Redirect 又拼一层 `req.Store`），理解不一致就拼出双桶段。格式变更只能改这一个函数，并且必须配套迁移。
2. **对象名和键格式是对外承诺**：完整 URL 会写进第三方页面和接入方的库，所以 `md5` 段的位置、大小写规则、**后缀取首写者且永不变更**这三点与 `store.keyword` 不可修改同级。变更路径只有"新建 store + 批量拷贝 + CDN 侧 301"。
3. **解析失败必须拒绝，不得回落到默认桶**：URL 的 host 不在 store map 里、path 去掉单个后缀段之后不是 32 位小写 hex，两种情况都直接 404（解析只允许用 `filepath.Ext`，禁止按段数切分，理由见命名规则细则 3）。这与"`provider` 不在白名单就启动报错"是同一立场——回落会把配置错误变成静默读错桶。
4. **大小写精确匹配**：`store_keyword` 是 `utf8mb4_bin`，URL host → store 的 map 查找和 `WHERE store_keyword=?` 都区分大小写；`bdemo` 与 `BDemo` 不能当成同一个桶。
5. **GC 必须能处理"对象存在但无行"**：厂商侧写入无法包进 DB 事务，所以存在"`CompleteMultipartUpload` 成功但 INSERT 失败"的窗口。它不需要补偿——同样内容重传会覆盖同一个键并成功建行，自愈；真正需要主动清的是反向（有行无对象，桶被人为清过），由启动自检或周期核对按 `store_keyword` 抽样 `HeadObject` 发现。
6. **桶内碎片要回收**：未完成的 multipart upload 一直在计费，现网只在手动删除文件时 Abort、没有超时回收（`internal/infra/store/s3.go:237-244` 有实现，调用方只有 `service/file.go:413-418`）。`upload` 表必须存厂商 `uploadId`，GC 任务带 store 上下文调 Abort。这条在 `upload` 一节展开。
7. **不需要"删 store 前反查"**（2026-10-13 定）：`store` 行不可删除、只可 `status=0` 停用，因此本表（以及 `app_config` / `export_task`）都不需要为"删 store 前的引用检查"准备索引或代码路径。见第 1 节"行只增不减"。

---

## 5. `upload` — 分片上传会话

一行 = 一次进行中的分片上传，是厂商侧 multipart upload 的镜像。**表里只有活跃会话**：合并成功或中止即删行，"行存在 = 会话活着"，没有状态列。

```sql
CREATE TABLE `upload` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `store_keyword` CHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '存储后端标识',
  `md5`           CHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL COMMENT '内容摘要（分片摘要的摘要）',
  `ext`           VARCHAR(16) CHARACTER SET ascii COLLATE ascii_bin NOT NULL DEFAULT '' COMMENT '后缀',
  `user_id`       BIGINT UNSIGNED NOT NULL COMMENT '发起用户ID，0=系统',
  `directory_id`  BIGINT UNSIGNED NOT NULL COMMENT '目标目录ID',
  `name`          VARCHAR(256) NOT NULL COMMENT '业务文件名',
  `upload_id`     VARCHAR(512) CHARACTER SET ascii COLLATE ascii_bin NOT NULL COMMENT '厂商分片上传ID',
  `total_size`    BIGINT UNSIGNED NOT NULL COMMENT '声明总大小',
  `created_at`    BIGINT UNSIGNED NOT NULL COMMENT '创建时间',
  `updated_at`    BIGINT UNSIGNED NOT NULL COMMENT '修改时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uidx_upload_upload_id` (`upload_id`),
  UNIQUE KEY `uidx_upload_session` (`store_keyword`, `md5`, `user_id`, `directory_id`),
  KEY `idx_upload_updated` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='分片上传会话';
```

### 说明

- **字段 `store_keyword` / `md5` / `ext`**：会话创建时就确定的三个事实——写哪个桶、内容摘要、对象名的后缀段。Prepare 先查秒传（`SELECT ... FROM object WHERE store_keyword=? AND md5=?`），**命中根本不建会话**、直接建 file 行返回 URL，只有未命中才创建会话；这三列是 GC 现场重建调用上下文（桶 + 对象名）所需的全部信息，没有它们 GC 无法构造 `AbortMultipartUpload`。三者类型必须与 `object` 逐字一致。
- **字段 `user_id` / `directory_id` / `name`**：会话的三个**放置事实**——谁传的、传到哪、叫什么。消费点都是**完成事务**（第 6 节）：file 行诞生时必须知道它们，而 GC 补合没有请求上下文、客户端重传 Complete 也不携带服务端解析出的 `directory_id`（路径解析是服务端行为，客户端只知道自己的路径串），所以必须随会话落库。**仍然不存 `app`**——完成时 `SELECT app FROM directory WHERE id=?` 一次完成"目录还在"+派生归属（`file.app ≡ directory.app`）。**`user_id` / `directory_id` 参与召回谓词并进入唯一键 `uidx_upload_session`**（2026-10-13 定，见下方"断点续传"）：条目长度 `128 + 32 + 8 + 8 = 176` 字节，远低于 3072；**`name` 仍不进任何键**——放置事实归首调者（与 `ext` 归首写者同构），召回命中后 file 行用首调者的 name。
- **字段 `upload_id`**：厂商 `CreateMultipartUpload` 返回的 uploadId。**它是会话凭据，不是元数据**——续传、传片、合并全部只凭这一个串定位会话，所以 `utf8mb4` 之外单列 `ascii_bin`（厂商 ID 是 ASCII 串，用 7bit 字符集让索引用 512 字节而不是 2048）。**唯一索引兼作定位与鉴权**：`upload_id` 是会话的唯一命中点，不存在"拿到哪条算哪条"的歧义。现网把这条定位放在旧 `file` 表上（`GetFileByUploadId`，`internal/infra/dbs/file.go:42-45`；线上该列本就是 `UNIQUE KEY upload_id`），新设计整体移到本表。
  **写入前必须校验，超长一律拒绝、绝不截断**：厂商 uploadId 是不透明串，**没有公开的长度上限保证**，`VARCHAR(512)` 只是我们留的余量（BOS / S3 / OSS 实测都在百字符量级）。校验三项：**非空**、**字节长度 ≤ 512**、**纯 ASCII 可见字符**（与 `ascii_bin` 列配套，含非 ASCII 即拒）。
  **为什么不截断**：截断后的串在厂商侧既 `Abort` 不掉（碎片会一直计费）也续传不了——等于把一个可诊断的错误换成一个不可诊断的泄漏。直接拒绝，并把 provider 与超长值记进日志。真遇到超过 512 的 provider，扩列即可（本轮是清库重建，无迁移成本）。
- **字段 `total_size`**：客户端在 Initiate 时的声明值。三个用途：①Initiate 之前挡目录上限（不能等合并完再说）；②合并判据里"Σ 片大小 == total_size"的比对基准；③进度显示。**列宽 `BIGINT UNSIGNED`** 与 `object.size` 一致，理由同（4GB 会溢出）。
  **它不再兼任"唯一能把声明与事实对上"的角色**：早先这里写着"服务端从不校验 `md5`"，所以只有 `total_size` 能当比对基准。**2026-10-13 起服务端会核验 `md5`**（见下方"内容摘要的算法"：客户端算、服务端在对象诞生之前重算比对），`total_size` 回归纯粹的"大小基准"。
- **字段 `created_at` / `updated_at`**：Unix 秒，代码写入。`updated_at` 在这张表上有第二个职责：**它就是 GC 的滑动过期时钟**（见下）。

### 为什么不存的列

- **没有 `status`**：会话生命周期只有"活着 / 结束"，行在不在就是答案。存状态位会造出"行说完成了、`object` 里没有行"的第二份真相——和 `file.status` 上踩过的坑同源。
- **没有 `app`**：app 由完成事务从目录派生，本表存它没有寻址价值。**但"没有 app"不等于"没有身份"**——本表有 `user_id`，它承担的是**鉴权**而不是寻址：`app` 回答"哪个租户"，`user_id` 回答"谁"；完成请求要证明的是"是同一个上传者"，用 user 比用 app 精确得多（同租户内的两个用户也必须区分开）。详见下方"会话鉴权"。连带否决了我一度想要的 `UNIQUE(store_keyword, md5)`（**跨上传者**共享一个会话）：**共享会话与并发上传不兼容**——两个所有者往同一个 uploadId 传片时，"到齐"没有定义，A 的 Complete 会被 B 在飞的片干扰成"缺片"，B 的 Abort 会拆掉 A 的传输。**这个否决今天依然成立，且被唯一键完整兑现**（2026-10-13 定）：`uidx_upload_session(store_keyword, md5, user_id, directory_id)` 把"所有者"从"人"收紧为"**(用户, 目录, 内容)**"——键相同 ⇒ `md5` 相同 ⇒ 分片边界与每片内容逐位相同 ⇒ 两个并发上传对同一个 uploadId 的写是**幂等覆盖**，"到齐"仍有定义（都以会话声明的 `chunkCount` 为准）；真正危险的形态（不同内容进同一会话）在键里就不可能发生。跨用户 / 跨目录的同内容上传仍各建各的会话，撞的是对象键 `{md5}.{ext}` 上的重复 PUT（字节相同，06 §7.2 的并发竞态段照旧成立）。一次 PrepareUpload = 一次"**秒传判定 → 会话召回 → 新建**"。
- **没有 `chunkSize` / `chunkCount`**：切片粒度是**代码里的推导函数**（定义、性质与厂商下限见下方"chunkSize 的推导"）——5MB 起步（全局 5MB 下限，2026-10-13 拍板），把厂商"单个 multipart 最多 10000 片"的上限吸收进推导式。**下发点有两处、值必须逐位相同**：① **专门的取片大小接口**（客户端先用它拿到值、再据此算内容摘要，见下方"内容摘要的算法"）；② `PrepareUpload` 响应里再带一份。它是传输参数，不是会话属性、不是文件属性、不是 app 属性，所以不进任何表；`chunkCount = ceilDiv(total_size, chunkSize)` 在同一会话内是精确的派生值，合并时服务端自算（这正是"不采信前端声明"的落点）。
  **命名（本节唯一叫法）**：这个值在本文件和 API 上一律叫 **`chunkSize`**（proto `PrepareUploadFileResponse.chunkSize`、配置 `conf.ChunkSize`、HTTP 路径 `/resource/api/chunk_upload` 都是这个），片数叫 **`chunkCount`**。**不要用 `part_size` / `part_count` 这两个别名**——同一个东西两个名字，读的人会以为是两个量。厂商侧叫 part（`ListParts` / `PartNumber` / `UploadPart` / `CompleteMultipartUpload`），**只有引用厂商 API 时保留 part 的写法**。
  **上界**：见下方"chunkSize 的推导"。
- **没有已传分片进度 / 片级 ETag**：分片接收的权威记录在厂商侧 `ListParts`（单片 ETag 也在那里）。自建计数器要落在"厂商调用成功、我们写库失败"这个不确定边界上，一漂就两种坏结果：漂少了，物理完整的对象永远合不上，而客户端**无法补回这条记录**（它每片都发成功了），只能整份重传；漂多了，缺片的对象被合出来。合并时那一次 `ListParts` 就是它的全部用途，不值得常驻副本。

### chunkSize 的推导（整数、MB 对齐）

```go
const MB = 1 << 20                                            // 1048576
chunkSize  = max(5*MB, ceilDiv(totalSize, 10000*MB) * MB)       // 全整数运算、5MB 下限（MB 对齐保留）
ceilDiv(a, b) = (a + b - 1) / b                               // 整数除法，向上取整
chunkCount = ceilDiv(totalSize, chunkSize)
```

等价说法：**不小于 `totalSize / 10000` 的最小 1MB 整数倍**，且不低于 5MB（下限已从 1MB 抬到 5MB）。四条性质各有用处：

1. **结果永远是 1MB 的整数倍**（5MB 下限本身也是 1MB 的整数倍）。客户端切片不会出现"半个字节"的边界歧义，合并判据第 2 条（除末片外每片 `Size == chunkSize`）也就有了一个干净的比较基准。
2. **片数必然 ≤ 10000**：`chunkSize ≥ totalSize / 10000` ⇒ `chunkCount = ceilDiv(totalSize, chunkSize) ≤ 10000`。这是把厂商"单个 multipart 最多 10000 片"的硬上限**吸收进公式**，而不是事后再判。注意 `totalSize / 10000` 那一步**也必须是向上取整**：用整数除法向下取整，会在 `totalSize` 恰好跨过 10000 分界时算出 10001 片。
3. **50GB 以内 `chunkSize` 恒为 5MB**（`totalSize ≤ 10000 × 5MB` 时 `ceilDiv(totalSize, 10000×1MB)` 的结果乘 1MB 后仍 ≤ 5MB，被 5MB 下限兜成 5MB），与"固定 5MB"完全等价，不引入任何行为变化。
4. **全程整数运算，禁止 `math.Ceil(float64(...))`**（现网就是这么写的）。理由不是浮点会失真（`totalSize` 上限 42.94 TB，远在 `2^53` 之内），而是**这个值必须与任何一处重算逐位一致**——服务端下发它、GC 补合时服务端还要自算 `chunkCount`、客户端也可能拿它反推。整数运算的一致性可证明，浮点只是"看起来一致"，没必要冒这个险。

**全局 5MB 下限**（2026-10-13 拍板"按照5M处理"）：不再按厂商分下限。S3 要求除末片外 ≥5MB（1MB 会被 Complete 拒掉），BOS / OSS 允许更小但也接受 5MB，所以公式下限直接抬到 5MB 即可对**全体目标厂商合法**——公式与 provider 解耦，无需在接入新厂商时复核下限。末片可小于 5MB，符合各厂商规则。

**上界**：`chunkSize` 超过 `uint32` 需要 `totalSize ≈ 42.94 TB`（推导见文首"类型分层"）。因此 proto 的 `chunkSize` 保持 `uint32`，**代价是单文件上限被隐式钉在该值**——计算处加断言，超界即报"文件过大"，**不得静默截断**（截断会让服务端下发的值与客户端实际切片不一致，表现为永远合不上）。**不另造业务常量**：上界由公式与类型推导得出，人为取一个更小的值只会多一个需要同步的地方。

**这个值是协议契约，不是普通常量**（2026-10-13 定）：内容摘要按它切片（见下一小节），所以改它会改变**每一份已有内容的 `md5`**、进而改变对象键 `{md5}.{ext}`。冻结级别与"对象名和键格式是对外承诺"同级（见下文"连带义务"2）——变更路径只有"新建 store + 批量拷贝 + CDN 301"。现在是清库重建、无历史数据，**改动的成本此刻为零**；上线后就不是了。

### 内容摘要的算法（分片摘要的摘要）

**规范定义在 `设计方案/06-upload.md` §4.2，此处只留结论与库相关约束**（避免两处抄同一段论证而漂移）：

```
md5 = MD5( hex(md5(part₁)) ‖ hex(md5(part₂)) ‖ … ‖ hex(md5(partₙ)) )
```

- **拼接的是 hex 文本**、按片号升序；**不是**厂商多段对象 ETag 的"二进制摘要拼接 + `-N`"。
- **单片路径必须同规**：`total_size ≤ chunkSize` 时 N=1，`md5 = MD5(hex(md5(整个文件)))`——**对文件 md5 再取一次 md5**。否则同一份内容走单片与走分片会得到两个身份，去重与秒传同时失效。
- **客户端算、服务端核**：客户端上传前算（要拿它查秒传）；服务端在**对象诞生之前**重算比对，不符即中止（`06-upload.md` §7.1 判据 4 / §5.1）。**核验成本≈0**——直传路径复用本来就必须调的 `ListParts`（含 `chunkCount=1` 的单片）。
- **客户端怎么拿到切片边界**：**先调取片大小接口**、再自己切片算摘要，顺序不能反（`06-upload.md` §6.1）。
- **由此 `chunkSize` 成为协议契约**（见上一小节末段）。

### 会话鉴权

会话是**有主的**：`user_id` 记下创建者，**取分片上传地址**与**完成**都必须由同一个用户发起。校验点只有一个，就放在"会话命中"的那一次查询上：

```
SELECT ... FROM upload WHERE upload_id = ? AND user_id = :请求user
```

查不到即拒绝，**并且返回"会话不存在"而不是"不是你的"**——不区分这两种情况，否则 uploadId 的有效性就成了一个可探测的 oracle。

**为什么必须做**：会话行持有的是**创建者的** `user_id` / `directory_id` / `name`，而完成只需要 `upload_id`。若只凭 uploadId 放行，任何拿到它的人都能操作别人的会话。注意**摘要核验落地后，这条路已经产不出"内容替换"**（攻击者传的字节算出的摘要必然对不上会话声明的 `md5`，合并判据第 4 条会拦下）；剩下的危害是**拒绝服务**——污染分片让真正的上传者永远合不上，或直接 `Abort` 掉别人的会话。加一次 user 比对把这两件事一起关掉，代价只是一次索引比对。

**不再叠 app 校验**（2026-10-13 定）：已确认 manager 侧 `user.id` **全局唯一**，所以上面的 user 比对已经隐含了租户隔离——一个 app 的 user 不可能是另一个 app 的 user。因此 `upload` 表**不需要 `app` 列**，"没有 `app`"这个决策保持不变（它本来回答的就是"哪个租户"，而这里要回答的是"谁"）。

**GC 补合不校验**：GC 是内部可信路径，按 `updated_at` 过期扫全表、没有请求上下文，因此不受这条约束；它用的就是会话自带的 `user_id`，产出的 file 行归属仍然正确。

### 合并判定（"何时合并"与"该不该合并"）

**发起权在客户端，判定权在服务端。** 服务端无法区分"还有片在路上"和"客户端已经死了"，任何自发合并都会把没传完误判成传完——现网 `internal/domain/service/file.go:383-384` 正是用"编号最大的一片到了"当信号，猜错就静默产出短对象并把行置 `COMPLETED`。

| 时机 | 谁发起 | 行为 |
|---|---|---|
| 正常 | 客户端显式调 `CompleteUpload(upload_id)` | 服务端跑下方判据，齐则走**完成事务**（下面展开）；不齐返回缺失片号，**会话保持存活**，客户端补齐再调 |
| 兜底 | GC 扫到 `updated_at` 过期的会话 | **先跑同一套判据试合并**（客户端只是崩在最后一步，字节其实都到了，捡回来省一整份重传流量）；判据齐走同一套完成事务，不齐才 Abort |
| 否决 | 任何时机 | 判据任一不满足，永不合并，没有"尽力而为"分支 |

判据（`ListParts` 必须全量分页，每个 part 只有 PartNumber/Size/ETag）：

1. 片号集合恰好 `== {1..ceil(total_size / chunkSize)}`——缺号=没传完；号越界=客户端越界（现网 `req.Index` 无上界校验，`service/file.go:379` 直接透传，`index=99999` 也能落片）；只比数量不比编号不够（现网 `:183` 的分支）。
2. 除末片外每片 `Size == chunkSize`，末片 `0 < Size ≤ chunkSize`——**末片允许顶满**（整倍数大小的文件，末片就是满片；写成 `<` 会把这类文件判成永远合不上），保证分片形状与下发的 `chunkSize` 一致，拦住客户端乱切（厂商自己的下限校验在 Complete 时还会再拦一道）。
3. `Σ Size == total_size`。
4. **摘要核验通过**：把 `ListParts` 的 ETag 序列（按片号升序、剥引号取 hex）拼成长串、取 MD5，与会话声明的 `md5` 比对；不符即 `Abort`、报错（见上方"内容摘要的算法"）。

> 判据 1~3 只证明"片都到齐了、形状对"，**证不了"内容就是客户端声称的那一份"**。判据 4 是唯一把"客户端声明"与"服务端事实"对上的那一条。

不用"最后一片请求带 `isLast` 标记"来省掉第三个接口：标记放错时机被拒后，客户端没有第二条带标记的请求可发，重试等于重传一片字节；显式 Complete 是纯控制调用，重试免费。

**完成前的固定顺序（2026-10-13 定）**：① `ListParts`（全量分页）→ 跑上述四条判据（**含摘要核验**）；② 判据齐 ⇒ `CompleteMultipartUpload`（对象**此刻才诞生**）；③ 对象已存在 ⇒ 走下面的完成事务。**不能"先 Complete 再核验"**——Complete 一旦成功就以客户端声明的 `md5` 为键写了字节，若该键已被合法内容占用，这就是一次覆盖。详见 `06-upload.md` §7.2。

**完成事务**（客户端 `CompleteUpload` 与 GC 补合走同一条路径，一个 DB 事务）：`[确保 object 行] + [INSERT file] + [DELETE 会话]`。确保 object 行 = 按 `(store_keyword, md5)` 查询，未命中才 INSERT（`size` 取会话的 `total_size`、`ext` 取会话值）；并发下撞 `uidx_object_store_md5` 的一方复用赢家的行，`size` 不符则拒绝（碰撞护栏）。file 行的 `app` 由 `SELECT app FROM directory WHERE id=?` 派生、`directory_id`/`name` 取会话、`upload_id` 取厂商 uploadId。**file 行的出生时刻 = 会话行的死亡时刻**——这是"表里只有活跃会话"不产生残渣的结构原因。

**幂等不靠状态列，权威在厂商侧**：`ListParts` / `Complete` / `Abort` 返回 `NoSuchUpload` 一律当"会话已在厂商侧结束"处理（已完成或已中止），不许冒泡成错误。此后按现场分四种收口：

| 现场 | 动作 |
|---|---|
| 会话行还在，桶里那份字节在（`object` 行已在，或行缺但 `HeadObject` 命中） | 客户端在厂商侧已合、完成事务未落地 → 重放完成事务（并发重放由 `uidx_file_upload_id` 仲裁，只有一个生效） |
| 会话行还在，`object` 行缺且桶内无对象 | 会话确实被中止 → 删会话行（厂商侧已无可 Abort 的东西） |
| 会话行已删，按 `upload_id` 命中 `file` 行 | 上次完成事务已成交 → 返回那一行的 URL（重复 Complete 的重试落在这里） |
| 会话行已删，`file` 也未命中 | 会话过期且无产出 → 返回"需重新发起"（内容若已在桶里，客户端重新 Prepare 会秒传命中，零字节） |

曾经设想的"回落 `object (store_keyword, md5)` + `size` 比对返回成功"被否掉：它在最后一行那种情景里会误报成功，而这次上传实际什么都没产出。**任何时候不得仅凭 `object` 行存在就断定本次上传成功**——对象行可能是别的上传者早先建好的。

### 断点续传与 GC

- **断点续传 = 重调 `PrepareUploadFile`（召回式，2026-10-13 定）**：秒传未命中后按 `(store_keyword, md5, user_id, directory_id)` 查本表，命中即**召回**该会话——`ListParts` 验活后返回既有 `upload_id` 与**已传片号列表**，客户端只补缺口；`NoSuchUpload`（厂商侧已死）⇒ 删会话行、重走新建；`session.total_size` 与请求 `size` 不符 ⇒ 拒绝（与秒传层的碰撞护栏同口径）。**对象命中优先于召回**：字节已在桶里时直接秒传，任何在飞会话留给 GC，不因它改道。客户端**不需要**自己保存 `upload_id`——它是召回的结果而不是续传的入参（刷新、换设备、重装都成立）。并发同键 Prepare 被唯一键收敛：`INSERT` 撞 `uidx_upload_session` 的一方重查复用赢家（与 `object` / `file` 撞键同形态），**不存在"两个在飞会话选哪一个"的问题**。
- **滑动过期的两个 Touch 点（直传后重定，2026-10-13）**：`GetUploadPartURLs` 每批取址 + 会话召回命中。传片字节不经过服务端，"每次传片顶 `updated_at`"已不成立；一次按份上传的取址往返有十余次（每批 ≤200 片）⇒ 活跃会话持续续期——**这是不用固定 `expired_at` 列的原因**：写死的过期时间会在传输中途把 5GB/40 分钟级别的健康会话 Abort 掉。
- GC 用 `WHERE updated_at < now - ttl` 走 `idx_upload_updated`，`ttl` 是服务配置（厂商侧 multipart 超时也在这条线上）。**多数过期会话在走到 GC 前已被召回采用**（同键重调 Prepare 即复用），GC 实际处理的是"客户端彻底放弃"的那批。
- **单片也进本表**：`total_size ≤ chunkSize` 的文件走 `chunkCount=1` 的多段上传，同样建会话、走 `upload` 表；`upload` 表里从此只有真实的厂商 uploadId（含 `chunkCount=1` 的会话）。这杀掉了现网"单片直传也 `uuid.NewString()` 塞进 `file.upload_id`"的混淆（`service/file.go:292`）。
- **`CompleteMultipartUpload` / `AbortMultipartUpload` 返回 `NoSuchUpload` 一律当"已完成/已释放"处理**，不能往上抛——现网 `internal/infra/store/s3.go:237-266` 的错误直接冒泡，会把成功的上传报成 500、把重试的 Abort 报成失败。

### 本表决策带来的连带义务

1. **`ListParts` 分页必修**：现网 `internal/infra/store/s3.go:197-211` 无 `IsTruncated` / `NextPartNumberMarker` 循环，超过 1000 片时续传列表和完成提交都被截断，这是本设计的**前置条件**。
2. **`chunkSize` 是代码里的常量函数**（定义与四条性质见"chunkSize 的推导"），服务端下发。**改这个公式会打断所有在飞会话**（判据变了，客户端只能整份重传），与 `store.keyword` 不可修改同级，当作一次代码发布来对待。公式已采用**全局 5MB 下限**（2026-10-13 拍板），对全体目标厂商合法，原来"接入新厂商时要复核下限"这一条不再需要。
3. **删 `upload_part` 表**：草案里的第 7 张表作废，表清单 8 → 7。分片级事实（片号、大小、ETag）的唯一权威在厂商侧，入库只会造第二份真相。
4. **GC 是"先验尸后清理"**：过期会话先跑判据试合并（判据齐 → 完成事务，捡回一整份流量）；判据不齐 → `AbortMultipartUpload` → 删行。**判定的权威一律是厂商侧 `ListParts`，不能用 `object` 行是否存在来推断**——对象行可能是别的上传者早先建好的，它证明不了本次会话合没合过；`NoSuchUpload` 同样走上面的四种收口（可能整份已合而事务未落地），不直接删了事。GC 必须带 `store_keyword` 构造桶上下文。

---

## 6. `file` — 业务文件

**状态：定稿**（2026-10-13）。

一行 = 一个业务文件条目：某个 app 的某个目录下、有名字的一个入口。物理字节在 `object`（第 4 节），本表只存归属与位置。**文件的身份是行 id**，不是内容、也不是名字——同一份内容可以出现在任意多个位置，重传和重名都不产生冲突。

```sql
CREATE TABLE `file` (
  `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `app`          CHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '应用标识',
  `user_id`      BIGINT UNSIGNED NOT NULL COMMENT '上传用户ID，0=系统',
  `directory_id` BIGINT UNSIGNED NOT NULL COMMENT '目录ID',
  `name`         VARCHAR(256) NOT NULL COMMENT '业务文件名',
  `object_id`    BIGINT UNSIGNED NOT NULL COMMENT '物理对象ID',
  `upload_id`    VARCHAR(512) CHARACTER SET ascii COLLATE ascii_bin NULL COMMENT '产出该行的上传会话ID',
  `created_at`   BIGINT UNSIGNED NOT NULL COMMENT '创建时间',
  `updated_at`   BIGINT UNSIGNED NOT NULL COMMENT '修改时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uidx_file_upload_id` (`upload_id`),
  KEY `idx_file_app_user` (`app`, `user_id`),
  KEY `idx_file_app_directory` (`app`, `directory_id`),
  KEY `idx_file_object` (`object_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='业务文件';
```

### 说明

- **字段 `app`**：应用标识，全库同规（`CHAR(32)` + `utf8mb4_bin`）。**结构不变式：`file.app ≡ directory.app`**——file 行只能生在它自己 app 的目录里。这个值不独立存在，由完成事务 `SELECT app FROM directory WHERE id=?` 派生（会话只带 `directory_id`，不带 app，见第 5 节），所以不存在"file 的 app 与目录的 app 不一致"的形态；**完成请求带的 app 与会话/目录不一致时直接拒绝**（见字段 `user_id` 与第 5 节"会话鉴权"），归属因此不可能记错。这次查询同时承担"目录仍存在"的检查，不是纯开销。
- **字段 `user_id`**：上传者，manager 侧 `user.id`，`0` = 系统（与 `export.user_id` 同规范）。**它必须落库、不能靠目录反推**：目录是共享的，同一个目录里放的是多个用户的文件，"这份文件是谁传的"是目录表达不了的事实。三个消费方：① **"我的文件"列表**（`WHERE app=? AND user_id=?`，`idx_file_app_user` 为它而建）；② **上传会话鉴权**——会话携带创建者，完成/传片时校验请求用户与会话用户一致，否则拒绝（第 5 节"会话鉴权"）；③ 追责。
  **无 DEFAULT**：与同表的 `directory_id` 同规——每一行都有确定的上传者，漏写必须报错而不是静默落成 `0`（"系统"）。**GC 补合也填得出**：它会话还在，`user_id` 随会话落库（第 5 节），所以四条诞生路径都能给出真值，不存在"因为没请求上下文只能写 0"的行。
- **字段 `directory_id`**：文件所在目录，引用 `directory.id`，不声明 FK。**NOT NULL、无默认值**——`0 = 根`的旧约定砍掉：线上 `file` 表实测 0 行根文件，且旧代码里根上传本来就不可达（`GetDirectoryLimitById(0)` 直接 record not found；全空路径返回零值 limit 后大小检查拒绝一切真实文件）。"没有目录的文件"这种形态不存在，上面的 app 派生才有稳定来源。
- **字段 `name`**：业务文件名。**同目录同名允许**，本表对名字没有任何唯一约束——上传的语义是"产生一个新条目"而不是"抢占一个名字"，加唯一会迫使定义"同名是覆盖还是拒绝"，两者都会让并发上传出现不可预期的失败，重名如何展示由接入方决定。**显示与下载一律以它为准**，`object.ext` 只是 URL 末尾段的外观（第 4 节命名规则细则 5）。
- **字段 `object_id`**：引用 `object.id`（第 4 节），不声明 FK。同一 object 可被任意多行 file 引用——跨 app、跨目录、同目录都成立。回收侧的唯一依据是 `SELECT 1 FROM file WHERE object_id=? LIMIT 1`（`idx_file_object` 为它而建）：删到最后一个引用才允许回收桶对象，`ref_count` 计数器因此不需要。**改名/移动不改变 URL**——URL 由 `object` 决定，与目录、名字无关。
- **字段 `upload_id`**：产出该行的上传会话 ID（厂商 uploadId 原值）。**可空**——只有秒传建的行没有会话，为 NULL（单片现在也建会话、取厂商 uploadId）；唯一索引对多个 NULL 不判重，任意多行共存。类型与 `upload.upload_id` 逐字一致（`VARCHAR(512)` + `ascii_bin`），两边要等值比较。**它的职责只有一个：完成事务的幂等键**——重复到达的完成请求（客户端重试、GC 重放）靠它找回已经产出的行（见第 5 节"幂等"），并发重放由唯一索引仲裁。曾经设想的"回落 `object (store_keyword, md5)` + `size` 比对"被否掉的原因见第 5 节。不做业务溯源（厂商侧会话完成即删，链本就不完整），不参与读路径。
- **字段 `created_at` / `updated_at`**：Unix 秒，代码写入，无 DEFAULT，与全库同规。改名（改 `name`）、移动（改 `directory_id`）用 `Select` 显式列更新，防 gorm 零值静默失效（同 `directory` 那节的教训）。
- **索引 `idx_file_app_user`**："我的文件"列表（`WHERE app=? AND user_id=?` 分页）。**不必把 `id` 写进索引**：二级索引隐含主键，`ORDER BY id DESC` 直接复用它。
- **索引 `idx_file_app_directory`**：目录内列表（`WHERE app=? AND directory_id=?` 分页），列序不能反，app 是恒等前缀。
- **索引 `idx_file_object`**：回收侧反查引用（见上面 `object_id` 字段说明）。

### 生命周期：行只从四条路径诞生

| 路径 | 诞生时机 | `upload_id` |
|---|---|---|
| 秒传 | Prepare 命中 object（`WHERE store_keyword=? AND md5=?`），建行即返回 URL，不建会话 | NULL |
| 多段直传（含单片） | 所有非秒传上传建会话、走 `upload` 表 | 厂商 uploadId |
| 分片完成 | 客户端 `CompleteUpload` 的完成事务内 | 厂商 uploadId |
| GC 补合 | 会话过期但 `ListParts` 判据齐，完成事务内 | 厂商 uploadId |

**`user_id` 的来源**：秒传（不经会话）取自**请求上下文**；分片/单片完成（建会话）取请求上下文，并**与会话的 `user_id` 比对，不一致拒绝**；GC 补合取**会话的 `user_id`**（它没有请求上下文，这正是 `upload` 必须存 `user_id` 的原因）。

**file 行的出生时刻 = 会话行的死亡时刻**（同一事务），所以 `upload` 表"只有活跃会话"不留残渣；反过来，任何 file 行都对应一次明确的产出事件，没有"半成品行"需要读接口自己去挡——现网 `internal/domain/service/file.go:92-94` 那句 `if res.Status != StatusCompleted { return NotExist }` 的补丁随拆表消失。

### 同一份内容的多处引用

去重边界是 store，不是 app（第 4 节）。同 store 的 N 次上传 = 1 行 `object`、桶里 1 份字节、1 条 URL、**N 行 file**（各自的 app / directory_id / name 完全独立）；不同 store = 物理上各自一份。引用数对 `object` 不可见，目录和名字对 `object` 也不可见。

- **顺序传**：第二次起走秒传——零会话、零字节；只有第一次真正传字节的那次在 `upload` 表留下过一行。
- **并发传同内容**：两个会话各自携带自己的放置（`directory_id`/`name`），完成事务里 `INSERT object` 一方赢、另一方撞 `uidx_object_store_md5` 复用赢家的行（`size` 不符拒绝）；两行 file 各自成立。`upload` 表故意没有 `(store_keyword, md5)` 唯一性——一行 = 一次上传动作，不是一份内容（第 5 节"为什么不存的列"）。
- **删除互不误伤**：删掉一个引用行，`object_id` 上还有别的行 → 桶对象保留；删到最后一个引用才回收。
- **`ext` 归首写者**：URL 末尾段由第一个写入该内容的行决定且永不复写；各行的展示/下载名以自己的 `name` 为准。

### 删目录时的对象回收

删目录会连带删掉子树里所有 `file` 行，但**被删的行不等于可回收的对象**——去重边界是 store 不是 app（第 4 节），同一份字节可以被任意多个目录、任意多个 app 的行引用。所以判定必须基于"删完之后还有没有人引用"，而不是"它出现在被删的子树里"。

**三个集合**（`A` 必须在 `DELETE` **之前**收集，`B` 必须在**之后**查）：

| 集合 | 定义 | 取的时刻 | 处置 |
|---|---|---|---|
| `A` | 子树里的 `file` 行指向的 `object_id`，去重 | 删之前 | — |
| `A ∩ B` | 删完之后，`A` 里仍有 `file` 行指向的 | 删之后 | **保留**（`object` 行与桶对象都不动） |
| `A − B` | 删完之后，`A` 里已无任何行指向的 | 推导 | **可回收**（删 `object` 行 + 删桶对象） |

顺序不能反：`DELETE` 一执行，子树原来指向过哪些 object 就无从得知了（行没了，线索也断了）。

**落地为 2 条查询 + 1 条 join-delete**（替代"逐个 `object_id` 点查"——万文件目录下那是万次点查）：

```sql
-- 1) 删之前收集候选
SELECT DISTINCT f.object_id FROM file f JOIN directory d ON f.directory_id = d.id
 WHERE d.app=? AND (d.id=? OR d.path LIKE :pattern ESCAPE '\\');
-- 2) 一条 join-delete 删掉整棵子树的行（不必先把 id 集合取出来）
DELETE f FROM file f JOIN directory d ON f.directory_id = d.id
 WHERE d.app=? AND (d.id=? OR d.path LIKE :pattern ESCAPE '\\');
-- 3) 删完一次查出"仍被引用"的那些 —— 返回的正是 A ∩ B
SELECT DISTINCT object_id FROM file WHERE object_id IN (:A);      -- :A 按 500 分批
-- 4) A − (A ∩ B) = 可回收，批量 DeleteObject
```

`LIKE` 前缀参数同样是 `escapeLike(path) + '/%'`（见第 3 节"路径前缀查询必须转义"）。

**执行边界与顺序（2026-10-13 补齐）**：

1. **第 1~3 步必须在同一个事务内。** 跨事务时第 3 步看到的是**另一个快照**——`DELETE` 尚未提交就查不出"已消失"这个事实，于是所有候选都被判成"仍被引用"，**一个对象都回收不掉**（不是偶发，是稳定失效）。同一事务内第 3 步能看到自己刚做的 `DELETE`（"读己之写"在 RR 与 RC 下都成立），所以这套写法**不依赖隔离级别**；写明这一点是为了防止有人把它当成"必须 RR 才对"。
2. **第 4 步的 `DeleteObject` 必须在事务提交之后。** 厂商调用无法回滚，放进事务里会把一个"可以重试的 DB 失败"变成"DB 已回滚、桶对象已经删了"的不可恢复状态。
3. **顺序：先删 `object` 行，再删桶对象。** 这个方向与本节"宁可留孤儿，不可留悬空行"是同一条：删行是**所有权交还**（此后再来的同内容上传会走真实写入、重新建行），而删桶对象是不可回滚的终局动作，放最后。两种失败残留并不对称——
   - 删行成功、`DeleteObject` 失败 → **无行有对象**：桶里多一份无人引用的字节，浪费计费但**不会让任何 URL 变成 404**；
   - 反过来 → **有行无对象**：秒传会命中这一行并发出一个死链，用户看到 404。
   把顺序选反，等于用"不可见的浪费"去换"用户可见的坏掉"，正是本节明确否掉的方向。
4. **删桶对象前要再查一次 `(store_keyword, md5)` 是否存在。** 第 3 步与第 4 步之间，同一份内容完全可能被重新上传：它走真实写入路径、`PUT` 覆盖同一个键、并 `INSERT` 出一行新的 `object`。此时若把键删掉，就等于删了新行的字节——新行立刻变悬空。命中即**放弃这次删除**（那份字节已经被新行接管）。
5. **残留（显式接受）**：第 3 步删行成功、第 4 步 `DeleteObject` 失败，就是第 3 条里那种"无行有对象"的孤儿，**库里没有任何线索能发现它**——`HeadObject` 抽样核对（第 4 节连带义务 5）只能发现反向的"有行无对象"。它只浪费计费、不影响任何 URL，**显式接受**；要发现它只能靠"枚举桶 × `object` 表"离线比对，本设计不提供。**别把这一条与"零引用 `object` 行扫描"（下文「离线对象核对」）混起来**——那一类残留的 `object` 行还在，这里行已经没了。`DeleteObject` 本身是幂等的（S3 系的 DELETE 对不存在的键返回 204），所以重跑这一段是安全的。

**这与"判掉 `ref_count`"不冲突**：判掉计数器的理由是"它会失真且失真后无从察觉"，不是"多一次查询"。这里消掉的是 N 倍放大，立场没变。

**并发下是保守的，这是故意的**：两个事务并发删不同引用、而它们共享同一个 `object` 时，双方都会看到对方那一行还没消失 → **双方都判定"仍被引用"→ 双方都保留** → 结果是"`object` 行与桶对象都留着、但零引用"的**孤儿**。方向刻意选成"**宁可留孤儿，不可留悬空行**"：孤儿只是浪费字节、且能被零引用 `object` 行扫描发现（下文「离线对象核对」）；悬空行是文件直接 404，属于"数据看起来坏了"。彻底消除要加锁或串行化，与本设计不加锁的立场冲突，不值得。

（备选写法：删之前用一条 `NOT EXISTS` 直接算出"只被子树引用"的 `object_id`，省掉两段式与顺序约束；但子树谓词要写两遍，可读性更差，且并发的保守性一模一样。）

### 离线对象核对（零引用行扫描）

单文件删除与删目录回收都只保证"**删到最后一个引用才回收**"；回收动作本身是"删 `object` 行 → 删桶对象"两步，第二步是厂商调用、无法回滚，**失败即留残留**。残留分两类，能被发现的手段完全不同：

| | 形态 | 成因 | 能否用 SQL 发现 |
|---|---|---|---|
| **A 类** | `object` 行**还在** + 桶对象还在，零 `file` 引用 | ① 删行失败；② 两个事务并发删共享同一 `object` 的不同引用 ⇒ 双方都看到对方那行还在 ⇒ 都保留（上文「并发下是保守的」段） | **能**，一条反连接，`idx_file_object` 支撑 |
| **B 类** | `object` 行**已删**、桶对象还在 | 删行成功、`DeleteObject` 失败 | **不能**，库里无行可查；只能"枚举桶 × `object` 表"，**本设计不提供**（执行边界第 5 条） |

本小节只兜 **A 类**，落地为一条周期性任务，纯 SQL、按批：

```sql
-- 1) 找零引用的 object 行（分批）
SELECT id, store_keyword, md5, ext FROM object o
 WHERE NOT EXISTS (SELECT 1 FROM file f WHERE f.object_id = o.id) LIMIT 500;
-- 2) 逐行先删行；CAS 再查一次，防"扫描之后、删除之前"被新 file 行引用
DELETE FROM object
 WHERE id = ? AND NOT EXISTS (SELECT 1 FROM file f WHERE f.object_id = ?);
-- 3) 再删桶对象（键已在第 1 步读出）
```

三条约束与「删目录时的对象回收」逐条对应：

1. **顺序仍是"先删行、再删桶对象"**，不能反——反过来的话，第 2、3 步之间那条 `object` 行会被**秒传**命中并发出死链，正是"宁可留孤儿，不可留悬空行"否掉的方向。
2. **第 2 步必须带 `NOT EXISTS` 再查一次（CAS）**，不能裸删。第 1 步到第 2 步之间同一份内容可能被重新上传并建出新 `file` 行；裸删就删掉了一条**有引用**的行，紧接着第 3 步会删掉别人正在用的字节。CAS 不命中即跳过。
3. **第 3 步失败要能重试**：键在第 1 步已读出、不依赖行存在，把 `(store_keyword, md5, ext)` 记进一张重试队列即可。这条把 B 类残留从"每次厂商调用抖动"收窄到"进程在第 2、3 步之间崩溃"。`DeleteObject` 幂等，重试安全。

**成本**：反连接由 `idx_file_object` 支撑，但 `object` 是亿级表 ⇒ 按主键区间**分批**推进，不做单条全量 SQL。

### 本表不存的列（及去向）

- **`store` / `md5` / `size` / `key`** → 全在 `object`。列表接口要大小、要 URL 时按 `object_id IN (...)` 主键批量查一次（第 4 节"读写路径"），不给本表冗余——冗余就是两份真相。
- **`type`** → 从 `name` 推导（`filepath.Ext`），零存储。
- **`status` / `chunk_count`** → 上传过程状态归 `upload`；本表行诞生即"可读"。
- **`updated_by`** → **不加**。`user_id` 已经表达"谁传的"；改名与移动不改变归属，不另记操作人；本表是亿级，不为审计加列。这与 `store` / `app_config` / `directory` 有 `updated_by` 是**有理由的差异，不是遗漏**——那三张表分别因为"`sk` 进库要追责""配置变更要追责""目录的改名与移动是跨用户管理动作且目录行没有任何别的'人'字段"。
- **`UNIQUE(md5)`** → 本表对内容摘要没有任何唯一约束。现网 `file` 的内容摘要列全局唯一正是"跨 store 撞 1062"的根因；去重的唯一仲裁在 `object.uidx_object_store_md5`。

### 本表决策带来的连带义务

1. **删目录要连带删子树里的 file 行，并按"三个集合"批量回收对象**：算法、顺序约束与并发语义见上文「删目录时的对象回收」。**不要退化成**"逐行 `SELECT 1 FROM file WHERE object_id=?`"——万文件目录就是万次点查。
2. **列表查询固定两条 SQL**：`WHERE app=? AND directory_id=?` 分页一条 + `object WHERE id IN (...)` 主键批量一条（第 4 节已定），不允许退化成每行一次点查。
3. **必须有"零引用 `object` 行"的周期扫描线**：单文件删除与删目录回收的失败残留（A 类）只由它兜住，SQL、CAS 与重试队列见上文「离线对象核对」。B 类残留（行已删、桶对象还在）无库内线索，已显式接受。

---

## 7. `export_task` + `export` — 导出产出物与请求记录

**状态：定稿**（2026-10-13）。
**⏸ 首版不实现（2026-10-13 定）**：导出功能整体推迟到首版之后（`设计方案/README.md` §1.1 范围声明），但**两张表的 DDL 首版就要建**（与其余 6 张表同批、同批自检），service / worker / 三条清理线 / 容量调度一律不实现。容量感知调度与本地目录的草稿见 `.prd/草案-导出本地目录与容量调度.md`，随导出一并延后。

### 为什么拆成两张表

现网 `export` 表把**两件生命周期完全不同的事**压在一行里：

| 概念 | 本质 | 生命周期 | 现网承载方式 |
|---|---|---|---|
| **产出物**（artifact） | 一份落在桶里的 zip 字节 | 生产 → 成功/失败 → 过期删除 | 行上的 `key`/`size`/`status`/`expired_at` |
| **用户请求**（request） | "某用户要了一份名叫 X 的导出" | 创建 → 一直可查（只要产出物还在） | 行上的 `user_id`/`name` |

压在一张表里直接导致两个现网缺陷：

1. **同一份内容被 N 个用户请求 → N 行，每行各带一份 `status`/`size`**，于是"这份产出物成功了没有"有 N 个答案。`CopyExport`（`dbs/export.go:31-44`）就是靠复制行来模拟"共享产出物"，复制出来的每一行都能独立漂移。
2. **引用计数靠按内容摘要 `COUNT(*)`**（`dbs/export.go:136-142`）——产出物的生死由"请求行的条数"隐式决定，与本文件判掉 `ref_count` / `used_bytes` 是同一条理由（派生值不能当真相）。

**拆法**：

- `export_task` = 产出物。身份 = `(app, md5)`，**重试的全部字段都在这里**。
- `export` = 用户请求记录。指向 `task_id`，只有 `user_id` + `name`。

拆完之后"重试"天然落在正确的层级：重试的是**产出物的生产**，与"有几个用户要它"无关。

### 7.1 `export_task` — 导出产出物与生产状态

```sql
CREATE TABLE `export_task` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `app`           CHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '应用标识',
  `md5`           CHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL COMMENT '导出内容MD5',
  `store_keyword` CHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '产出物落地的存储后端',
  `size`          BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '产出物大小',
  `status`        TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态 0待执行 1执行中 2成功 3失败 4已过期',
  `retry`         TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '已尝试次数',
  `reason`        VARCHAR(512) NULL COMMENT '最后一次失败原因',
  `expired_at`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '产出物过期时间',
  `leased_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'worker 最后一次领取或心跳时间（Unix 秒），0=从未领取',
  `failed_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '本轮退避的计时起点（最后一次失败时间，Unix 秒），0=本轮未失败过',
  `created_at`    BIGINT UNSIGNED NOT NULL COMMENT '创建时间',
  `updated_at`    BIGINT UNSIGNED NOT NULL COMMENT '修改时间（纯审计，由 GORM 自动维护）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uidx_export_task_app_md5` (`app`, `md5`),
  KEY `idx_export_task_claim` (`status`, `failed_at`),
  KEY `idx_export_task_expire` (`status`, `expired_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='导出产出物';
```

#### 说明

- **字段 `app` + `md5`**：产出物身份。`md5` = **产出规格**的 MD5（见 7.3），`app` 限定去重范围——**跨 app 不共享**。现网的导出查询按内容摘要全局进行、不带 app 限定（`dbs/export.go:111-114`），A 应用导出的东西 B 应用请求同内容会直接命中，属跨租户泄漏，本次修掉。
  选 `app` 字符串而非 `app_id`，理由与 `directory` 一致（全库同规，`CHAR(32)` + `utf8mb4_bin`）。
- **`md5` 是"规格摘要"而不是"zip 字节摘要"**：这一点必须说清，否则会和 `object.md5` 混淆。请求到达时 zip 还不存在，而**去重必须在那一刻完成**，所以只能对"产出规格"取摘要；规格确定 ⇒ 产出物确定，因此它仍是产出物的内容身份。反面做法（拿产出后的 zip 字节算 md5 当身份）不可行：zip 内含文件时间戳，同一规格两次生产的字节可以不同，身份会漂。
- **字段 `md5` 用 `ascii_bin`**：它是 32 位小写 hex，`ascii_bin` 让唯一索引每行 32 字节而不是 128。**P1-9 已定稿（2026-10-13）：全库 `md5` / `ext` 统一 `ascii_bin`**——本表与 `object` / `upload` 一致，不再有"两种字符集并存"的中间状态，口径见第 4 节字段 `md5` 的"字符集口径"。`app` 仍为 `utf8mb4_bin`（要与 `app_config.app` 逐字一致）。
- **字段 `store_keyword`**：产出物**实际落地**的桶，在**创建任务时**由 `app_config.export_store_keyword`（NULL 时回落 `store_keyword`）解析后写死。此后 app_config 改配置**不影响历史产出物**——与 `object.store_keyword` 同一条立场（"现在写去哪" vs "当时写去了哪"）。
- **字段 `size`**：**单位字节**，`BIGINT UNSIGNED`。现网是 KB + `uint32`（`entity/export.go:10`），统一到字节后与 `object.size` 同规。
- **字段 `status`**：`0` 待执行 / `1` 执行中 / `2` 成功 / `3` 失败 / `4` 已过期。
  **这里存状态列不违反本文件"不存状态"的原则**——`file`/`object` 的状态是**派生**的（行在即活、字节在即活），而导出任务是**真异步**的："这次生产成功了没有"无法由任何其他列推出，它是事实而非派生值。
  `3`（失败）**同时表示"会重试"和"重试耗尽"**，两者由 `retry < maxRetry` 区分——不加 `DEAD` 之类的状态值，避免一个状态位表达两件事。
- **字段 `retry`**：**已尝试次数（含首次）**，初值 0，**每次领取 +1**（不是每次失败 +1——崩溃在途的尝试也必须计数，否则一份必然失败的任务会无限重跑）。因此 `retry < 3` 的语义是"**总共最多执行 3 次**"，不是"首次之外再重试 3 次"。
  **上限写在代码里**：`const maxRetry = 3`。**不进库**的理由：它是"厂商/协议级常量"性质（与 `chunkSize` 的 5MB 起步、`provider` 白名单同类），不随 app 变化；放行上只会多一份可漂移的配置，且历史行上的值在常量改动后无法复盘，反而制造"为什么这行是 2 那行是 5"的困惑。
- **字段 `reason`**：最后一次失败原因，供人工排查与前端展示。**成功时置 NULL**。
- **字段 `expired_at`**：产出物过期时间 = 生产成功时刻 + 该 app 的 `app_config.export_ttl`。`0` = 未成功（还没有产物）。**按 app 取 TTL** 正是第 2 节要求的"导出清理任务必须按 app 分别取 TTL"——现在 `app` 就在行上，清理任务可以按 app 分组。
- **字段 `created_at` / `updated_at`**：Unix 秒，代码写入，不给 DEFAULT，全库同规。**`updated_at` 是纯审计列**——由 GORM 在每次 `Updates(map)` 时自动填（见 7.4"实现注记"），**没有任何谓词或排序读它**。
- **字段 `leased_at` / `failed_at`**（2026-10-13 定，替换原先由 `updated_at` 兼任的两个计时职责）：两个**计时列**，`0` = 未发生，`DEFAULT 0` 与 `expired_at` / `size` / `status` / `retry` 同规。
  - **`leased_at`**：worker 领取时写、执行中周期性心跳时顶动。**只被租约判定读**——`WHERE status=1 AND leased_at < :now - :lease`。
  - **`failed_at`**：失败跃迁（`1→3`）时写。**只被退避谓词读**——`failed_at + backoff(retry) <= :now`。
  - **为什么不共用一个列**：它们在**两个不同的跃迁**上写、被**两个不同的谓词**读，本来就是两个量。原先靠"按 `status` 不相交"才敢共用一个名字（`status=1` 时读作心跳、`status=3` 时读作失败时刻），但那份"不相交"是**约定**而非**结构**——任何无关 UPDATE 顶了 `updated_at`，计时就被静默破坏，且没有任何自动检查能发现。拆成两列后这条约束由**列名**承担，不再依赖文档纪律。
  - **不变量**（两条，2026-10-13 定）：`status=0 ⇒ failed_at=0`、`status=3 ⇒ failed_at>0`。前者由**手动重试**（`retry=0` / `failed_at=0` / `expired_at=0` 同步复位，2026-10-13 起是唯一维护者——请求路径的自动复活已撤销，见 7.5）维持；后者由**失败跃迁**与**租约回收**两条路径写入正数时刻保证。合起来给出两个**可直接当告警条件**的不可能状态：`status=0 AND failed_at>0`、`status=3 AND failed_at=0`。这两条不是文档纪律，而是"`status` 与 `failed_at` 只在少数几条 CAS 语句里成对写入"的结构性推论——**漏写一处就会在这里现形**，不会变成安静的调度错位。

#### 不存的列（及理由）

- **没有 `max_attempts`**：常量，属代码（见上）。
- **没有 `next_retry_at`**：**可推导**——"下次可执行时间" = `failed_at + backoff(retry)`，而 `failed_at` 在失败跃迁上被顶，`backoff` 是 `retry` 的纯函数。存它等于存第二份真相（两列必须逐字同步，一旦不一致无人察觉）。代价见 7.4。
- **没有 `started_at` / `finished_at`**：租约判定用的是 `leased_at` 心跳（见 7.4），不需要"本次开始时间"；"最后一次结束时间"不单独存——终态跃迁必然伴随一次 `updated_at` 更新，需要展示时读这个审计列即可（**只用于展示，不参与任何判定**）。
- **没有 `object_id` / 复用 `object` 表**：导出产物**不进 `object`**。理由是本文件给 `object` 定死的不变式——"**回收只看 `file` 引用**"（第 4 节）。导出产物是 **TTL 驱动**、不是引用驱动，塞进 `object` 就要给回收侧加第二个引用来源，破坏那条不变式；而共享性已由 `(app, md5)` 在任务层解决，`object` 的按内容去重在这里没有增量收益。
- **没有 `ref_count`**：`export` 行的数量是派生值，回收走扫描（`SELECT 1 FROM export WHERE task_id=? LIMIT 1`），与 `object` 同一立场。
- **没有 `ext`**：产出物恒为 `.zip`（现网 `exportKey` 写死 `.zip`，`export.go:78-80`）。本文件对"没实现过的取值不预留"（如 `cdn_b`/`cdn_c`）是同一态度——真要支持 csv/直出 xlsx 时再加列。
- **没有 `object_key`**：键由 `md5` 推导（见下），不存第二份真相。

#### 唯一索引与索引

- **`uidx_export_task_app_md5`**：`128 + 32 = 160` 字节。同时是①去重判定的执行计划（`WHERE app=? AND md5=?` 全等值命中）、②并发创建的仲裁（`INSERT` 撞键的一方重查复用赢家）。
- **`idx_export_task_claim`**：worker 领取任务的入口（`WHERE status IN (0,3)`）。第二列 `failed_at` 既服务退避的残差过滤，又服务 `ORDER BY failed_at ASC` 免排序。**选 `failed_at` 而不是 `leased_at`**：候选排序要的是"等得最久的最先做"，而 `failed_at` 正是"这一轮从什么时候开始等"；`status=0` 的行 `failed_at=0`，天然排在最前（没做过的最先做）。
- **租约恢复复用同一个索引的 `status` 前缀，不另建 `leased_at` 索引**（2026-10-13 定）：`WHERE status=1 AND leased_at < :now - :lease` 用 `status=1` 定位，`leased_at` 作为残差过滤。之所以够用，是因为 `status=1` 的行数**上界就是在飞导出数**（≤ `workerCount`，默认 20）——这是一个由 `workerCount` 天然钉住的小集合，为它单建索引（每行 8 字节 + 每次领取/心跳都要维护）不划算。若将来 `workerCount` 上调到千级且恢复扫描变慢，再补 `(status, leased_at)` 也不迟。
- **`idx_export_task_expire`**：过期回收扫描（`WHERE status=2 AND expired_at<=?`）。

#### 对象键

**`export/{md5}.zip`**，一层平铺。

- **`export/` 前缀是刻意的**：本文件的"不带前缀"规则针对的是 **store 段**和**日期/哈希分层**（第 4 节），理由是"键必须能由列推导"。`export/` 是**功能命名空间**，仍是 `(md5)` 的纯函数，不违反该规则。
  它解决一个真实冲突：若导出产物与用户上传共用同一个桶（`export_store_keyword` 为 NULL 时就是），键空间会重叠——用户上传一份同名 zip 就可能与导出产物撞同一个 key，删任一边会删掉对方。加前缀把两者隔开。
- **键必须由唯一的构造函数产生**：`ExportObjectName(md5 string) string`，与 `ObjectName(md5, ext)` 同级对待，配单测锁死。

### 7.2 `export` — 用户导出请求记录

```sql
CREATE TABLE `export` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `app`        CHAR(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '应用标识',
  `task_id`    BIGINT UNSIGNED NOT NULL COMMENT '导出产出物ID',
  `user_id`    BIGINT UNSIGNED NOT NULL COMMENT '发起用户ID',
  `name`       VARCHAR(256) NOT NULL COMMENT '用户可见名称',
  `created_at` BIGINT UNSIGNED NOT NULL COMMENT '创建时间',
  `updated_at` BIGINT UNSIGNED NOT NULL COMMENT '修改时间',
  PRIMARY KEY (`id`),
  KEY `idx_export_app_user` (`app`, `user_id`),
  KEY `idx_export_task` (`task_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='导出请求记录';
```

#### 说明

- **字段 `app`**：结构不变式 **`export.app ≡ export_task.app`**，在创建请求的同一事务内从任务行派生。存它是为了 `idx_export_app_user` 这个"我的导出列表"主查询不必 join（与 `file.app ≡ directory.app` 同一条理由）。
- **字段 `task_id`**：引用 `export_task.id`，不声明 FK。**可被任意多行引用**（N 个用户要同一份内容）。回收侧反查 `SELECT 1 FROM export WHERE task_id=? LIMIT 1` 由 `idx_export_task` 承担。
- **字段 `user_id`**：manager 侧 `user.id`，`0` = 系统。**它只挂在本表（请求记录）上，不挂产出物**——产出物与用户无关，这是拆表后"跨用户共享产出物"能成立的前提。**`NOT NULL` 且无 DEFAULT，与 `file.user_id` 逐字同规**（2026-10-13 定）：每一行都有确定的发起者，漏写必须报错而不是静默落成 `0`（"系统"）。原稿这里是 `DEFAULT 0`，与 `file` 不一致，本次统一到无 DEFAULT。
- **字段 `name`**：用户可见名称。**同 app 内允许任意多行同名**，本表对名字没有唯一约束（与 `file.name` 同一立场）。
  **`name` 不进 `md5`、不进 zip**（见 7.3），它只在**下载时**作为 `Content-Disposition` 的文件名应用。这是修正现网缺陷的关键：现网把 `name` 写进 zip（`export.go:221`）而摘要又不含 `name`，导致第二个请求者拿到的 zip 内部文件名是第一个人的。
- **`idx_export_app_user`**：列表主查询（app + 用户内分页），列序不能反。
- **`idx_export_task`**：回收反查引用。

#### 本表不存的列（及去向）

- **`status` / `size` / `md5` / `key` / `reason` / `expired_at`** → 全在 `export_task`。列表接口要状态和大小时按 `task_id IN (...)` 批量查一次（见 7.5），不给本表冗余——冗余就是两份真相。
- **`ref_count`** → 派生值，走扫描。

### 7.3 摘要口径（`md5` 到底覆盖什么）

**`md5` 必须覆盖"决定 zip 字节的全部输入"，且只覆盖它。** 现网口径是错的：

| 入口 | 现网摘要输入 | 问题 |
|---|---|---|
| `ExportExcel` | 仅 `req.Rows`（`export.go:94-95`） | **漏了 `headers` 与 `files`**：同 rows、不同 files 的两个请求被判成同一份，后者拿到前者的附件 |
| `ExportFile` | `files` + `keys`（`export.go:435-437`） | 口径与 Excel 入口不一致 |
| 两处共同 | 都不含 `name`，但 `name` 被写进 zip（`export.go:221`） | 第二请求者拿到第一个人的内部文件名 |

**新口径**：

```
md5 = MD5( canonical( headers, rows, files(value+rename), keys ) )
```

- **纳入**：`headers`、`rows`、`files`（含 `rename`）、`keys` —— 凡是会改变 zip 内容的。
- **排除**：`name`（展示名）、`user_id`、请求时间 —— 凡是**不改变 zip 内容**的。
- **要求 canonical 序列化**：键序、浮点/空值表示必须稳定（Go 的 `json.Marshal` 对 struct 是稳定的，对 `map` 不是——现网用的是 struct，安全；但新实现必须保持用 struct 或显式排序）。

**配套改动**：`name` 从 zip 内部移出。xlsx 在 zip 内的固定名改为中性名（如 `data.xlsx`），`Content-Disposition` 用请求行的 `name`。这样"同内容不同名"才能真正共享产出物——**这正是导出作为重任务最该省的那部分算力**。

### 7.4 重试机制

#### 状态机

```
        创建                领取(claim)              生产成功
  (0) 待执行 ─────────────► (1) 执行中 ─────────────► (2) 成功 ──[过期]──► (4) 已过期
        ▲                       │
        │ 退避到期(failed_at     │ 生产失败
        │  + backoff(retry))    ▼
        └─────────────────── (3) 失败 ──[retry>=maxRetry]──► 终态(仍为 3，不再被领取)
                                │                                 └─ 由 7.6 任务行回收
     手动重试(重置 retry=0,     ─────┘   CAS: WHERE id=? AND status IN (2,3,4)
              failed_at=0, expired_at=0)
```

**终态 = {2 成功, 3 失败(重试耗尽), 4 已过期}**，手动重试只从这三个状态出发；`0` / `1` 不接受手动重试（见下"手动重试"）。

#### 常量（写在代码里）

```go
const maxRetry = 3                                                    // 最多尝试 3 次（含首次）
var backoff = [3]time.Duration{0, 30 * time.Second, 120 * time.Second} // 索引 = retry
```

`retry` 上限 3 时，退避表只有两个非零值（`retry=1 → 30s`、`retry=2 → 120s`），所以它不是"参数"，是一张两行的表——**这正是把它写进代码而不是做成配置的依据**。

#### 领取（claim）—— 乐观 UPDATE，无锁、无分布式锁

```sql
-- 轮询候选
SELECT id FROM export_task
 WHERE ( status = 0
         OR ( status = 3 AND retry < 3
              AND failed_at + CASE retry WHEN 1 THEN 30 WHEN 2 THEN 120 ELSE 0 END <= :now ) )
 ORDER BY failed_at ASC, id ASC
 LIMIT :n;

-- 领取（原子）
UPDATE export_task
   SET status = 1, retry = retry + 1, leased_at = :now
 WHERE id = :id AND status IN (0,3) AND retry < 3
   AND ( status = 0
         OR failed_at + CASE retry WHEN 1 THEN 30 WHEN 2 THEN 120 ELSE 0 END <= :now );
```

`RowsAffected == 1` 才算领到。多实例并发时只有一个能改到，**不需要锁、不需要 upsert**——与 `object` 完成事务的并发处理同一套思路。

> **实现注记（GORM，2026-10-13 定）**：本节五条 CAS UPDATE 一律用 `Updates(map[string]any{...})`，**不用 `UpdateColumns`**。原因是 `UpdateColumn` / `UpdateColumns` 会设 `Statement.SkipHooks = true`（`gorm@v1.31.1/finisher_api.go:416-428`），而 GORM 自动补 `updated_at` 的回调恰好被 `!stmt.SkipHooks` 门控（`callbacks/update.go:236`）——走这两个方法的话 `updated_at` **永远不会**被自动填，等于静默丢掉审计时间。两个硬条件：① 必须带 `Model(&entity.ExportTask{})`，否则 `stmt.Schema == nil`，自动填同样不生效；② map 里**不要**出现 `updated_at`（出现则按你给的值写）。map 本来就不受"零值跳过"影响，所以放弃 `UpdateColumns` **不损失任何东西**。`retry = retry + 1` 这类表达式用 `gorm.Expr("retry + 1")`。

#### 为什么不存 `next_retry_at`

- **它可推导**：`next_retry_at ≡ failed_at + backoff(retry)`。`failed_at` 在失败跃迁（`status→3`）上被顶，那正是"本轮计时起点"的唯一来源；`backoff` 是 `retry` 的纯函数。存它就是第二份真相。
- **两列必须逐字同步**：一旦 `failed_at` 被别的路径改动，两列立刻不一致，且没有任何自动检查能发现——与本文件判掉 `object_key` / `etag` 是同一条理由。
- **代价（明确接受）**：退避谓词 `failed_at + CASE retry ... <= now` 是**非 sargable** 的，不能在索引上做 range。但候选集是 `status IN (0,3)`（待执行 + 失败待重试），**在健康系统里这个集合天然很小**（任务被 worker 快速领走），索引 `idx_export_task_claim(status, failed_at)` 先把它缩到这一小撮，残差过滤在内存里做，成本可忽略。若将来这个集合变大到影响领取延迟，再引入 `next_retry_at` 并加索引也不迟——但那时它是**性能列**，需要写明"由代码维护、与 `failed_at` 同步"。

#### 成功

```sql
UPDATE export_task
   SET status = 2, size = :size, expired_at = :now + :export_ttl,
       reason = NULL
 WHERE id = :id AND status = 1;
```

`export_ttl` 取自该 app 的 `app_config.export_ttl`（第 2 节）。`status = 1` 作为 CAS 条件：若租约已过期被别的 worker 接管，这次 UPDATE 不生效，**不会用旧结果覆盖新状态**。

#### 失败

```sql
UPDATE export_task
   SET status = 3, reason = :err, failed_at = :now
 WHERE id = :id AND status = 1;
```

退避不写库——它是 `failed_at` 与 `retry` 的函数（见上）。`retry >= maxRetry` 时**不需要任何终态标记**：领取查询的 `retry < 3` 条件自然让它再也取不到。这就是**终态失败**（2026-10-13 定）：不自动复活、不会自动重跑，同规格的新请求只会拿到"失败"，只有手动重试（见上）能救。终态行的清理走 7.6 的任务行回收线（`status IN (2,3,4)`），不在这里处理。

#### 崩溃恢复（租约）

worker 中途死掉会留下 `status=1` 的僵行。恢复靠**租约超时**：

```sql
UPDATE export_task
   SET status = 3, reason = 'worker lease expired', failed_at = :now
 WHERE status = 1 AND leased_at < :now - :lease;
```

- **执行中的 worker 必须周期性顶 `leased_at`**（心跳），`lease` 取"单次导出最长耗时的 2~3 倍"（服务配置）。否则一份大导出会在传输途中被误判成僵死。
- **心跳只动 `leased_at`，不动别的列**：这正是把它从 `updated_at` 里拆出来的收益——心跳不再与"退避计时起点"共用一列，于是不再需要"改这张表时不许顺手 `Updates` 整行"这类**文档纪律**，约束改由列名承担。
- 注意：租约超时后原 worker 可能还活着并最终写成功。此时它的 `WHERE status=1` CAS 会失败（行已被重置为 3 或已被别人领成 1）——**幂等由 CAS 保证，不靠互斥**。而物理产出物键是 `export/{md5}.zip`，重复生产是**覆盖同一份字节**，无害。

#### 手动重试

前端"重试"按钮 = 一条 UPDATE，无需新列、无需新表：

```sql
UPDATE export_task
   SET status = 0, retry = 0, reason = NULL, failed_at = 0, expired_at = 0
 WHERE id = :id AND status IN (2, 3, 4);
```

**`expired_at = 0` / `failed_at = 0` 必须与 `retry = 0` 一起重置**（`expired_at` 2026-10-13 补）：手动重试的语义是"**整轮重来**"——计时起点归零（否则破坏 7.1 的 `status=0 ⇒ failed_at=0` 不变量、`ORDER BY failed_at` 排序失真）；产物时间也归零，否则从 `status=2/4` 唤醒的行带着过去的 `expired_at` 排队，在重新生产成功前一直显示"已过期"（`0` = 未成功、还没有产物）。

**`status IN (2,3,4)` 这个 CAS 条件不能省**（2026-10-13 定）：只允许从**终态**重试。少了它，一次点击就会把正在跑的 `status=1` 打回待执行——worker 随后的 `WHERE status=1` 成功 CAS 失败、结果被丢弃，而任务被重跑一遍，等于白烧一份算力还多产出一份字节。`RowsAffected = 0` 时返回"当前状态不可重试"，前端据此禁用按钮并提示。

`status=0` 的行也**不**接受手动重试：按不变量它已经是 `retry=0 ∧ failed_at=0`，重置是个纯 no-op；语义上"已在队列里的任务"也没有"重试"可言。**不要**用"重置能让退避立即到期"当理由——`status=0` 的行在候选查询里走的是独立的 OR 分支，本来就无条件可领，`failed_at` 取什么值都不参与判断。

#### 为什么不做"按 md5 找回会话"式的重试

导出是**无状态纯函数**（输入 = 规格，输出 = zip），没有厂商侧会话、没有部分进度可续。所以重试 = **整份重跑**，`upload` 表那套 `ListParts` 续传逻辑在这里没有对应物。这也意味着**重试成本高**，所以 `maxRetry` 只给 3，且**去重命中要尽可能前置**（7.3 的口径修正是提高命中率的关键）。

### 7.5 读写路径

#### 写（请求导出）

1. 按 7.3 算 `md5`。
2. `SELECT id, status, retry, expired_at FROM export_task WHERE app=? AND md5=?`。
3. 按命中情况处理：

| 命中情况 | 动作 |
|---|---|
| 未命中 | `INSERT export_task`（`status=0`, `retry=0`, `store_keyword` 从 app_config 解析）+ `INSERT export`。撞 `uidx_export_task_app_md5` 则重查复用赢家 |
| `status=2` 且 `expired_at > now` | 直接 `INSERT export` 返回——**秒传，零算力** |
| `status=2` 且 `expired_at > now` | 直接 `INSERT export` 返回——**秒传，零算力**（有效期内的唯一快路径） |
| `status=2` 且 `expired_at <= now`，或 `status=4` | **已过期**：只 `INSERT export` 返回，**不改任务行状态**（2026-10-13 定：自动复活撤销）。判据是行的 `expired_at`，不是桶里文件是否还在；清理线随后 `2→4` 删桶对象；重跑走手动重试（7.4，`expired_at=0` 一并复位） |
| `status IN (0,1)` | `INSERT export` 返回（排队/生产中，前端轮询） |
| `status=3` 且 `retry < maxRetry` | `INSERT export` 返回（重试中）；可选把 `failed_at` 提前到 `:now - backoff(retry)` 让退避立即到期（见下） |
| `status=3` 且 `retry >= maxRetry` | `INSERT export` 返回（**终态失败**，不自动重跑）；前端提供"重试"按钮 |

**为什么过期产物不自动复活**（2026-10-13 定，三条）：① **消除争抢**——清理线的谓词是 `status=2 AND expired_at<=now`（写行），原复活分支谓词与它逐字相同（也写行）；改为只读后写侧谓词不相交，不再需要 `AND status IN (2,4)` 之类补丁。② **原复活窗口没有价值**——清理线 5 分钟一轮，自动复活只在"已过期未扫到"的几分钟内有效，之后任务行以 `status=4` 长眠（等请求行散尽，最长 `retention` 30 天），期间 100% 失效。③ **"文件还在"不是事实判据**——桶里有没有字节是清理线的过程状态，行的 `expired_at` 才是"产物作废了没有"的唯一事实（与读路径"以 `object` 行为准"同立场）。**显式接受的代价**：用户再次请求同一份导出需多点一次重试（`export_ttl` 默认 7 天，低频显式行为）。完整论证见 `设计方案/08-导出.md` §5。

**"可选把 `failed_at` 提前"这个动作的写法**：若希望"新请求把已失败的任务提到队首"，**不能**把 `failed_at` 写成 `0`——那会违反 `status=3 ⇒ failed_at>0`（失败的行必须有退避起点），同样污染告警哨兵。正确写法是让它"假装已经等够了"：

```sql
UPDATE export_task
   SET failed_at = :now - CASE retry WHEN 1 THEN 30 WHEN 2 THEN 120 ELSE 0 END
 WHERE id = :id AND status = 3 AND retry < 3;
```

这样 `failed_at + backoff(retry) = :now`，谓词立即成立，而 `failed_at` 仍是正数。**注意这是一个请求路径上的副作用写**（多一条 CAS UPDATE、多一处并发失败要处理），若不需要"优先重试"语义，最省事的做法是**不做这个可选动作**——把"插队"留给显式的手动重试。

**`status=3 且 retry >= maxRetry` 是终态**：不会自动重跑，同规格的新请求只会拿到"失败"，只有手动重试能救。这条与 7.6 的回收线配套——终态失败行没有产出物可删，直接并入任务行回收。

**`store_keyword` 解析必须校验 `app_config` 存在**——缺行即报"应用未配置"，不回落到服务级默认（与第 3 节同立场）。

#### 读（列表）—— 固定两条 SQL

```sql
SELECT ... FROM export WHERE app=? AND user_id=? ORDER BY id DESC LIMIT 20 OFFSET ?;
SELECT id, status, size, expired_at, reason, retry, failed_at FROM export_task WHERE id IN (...);
```

第二条走主键。**不允许退化成每行一次点查**（与第 6 节 `file` 列表同规）。
前端若要展示"下次重试时间"，由服务端按 `failed_at + backoff(retry)` 现算返回，不落库——`retry` / `failed_at` 与其余列同在一行，多选两列不增加 IO，但**省掉了一条专为推导而写的 SQL**（与"不存 `next_retry_at`"同一条立场：策略不落库，用时现算）。

#### 下载

下载 URL 由 `store_keyword` + `ExportObjectName(md5)` 拼出。**下载时把请求行的 `name` 作为 `Content-Disposition` 文件名**，不依赖 zip 内部命名。

### 7.6 回收（三条独立的清理线）

| 清理对象 | 触发条件 | 动作 |
|---|---|---|
| **产出物（物理）** | `status=2 AND expired_at>0 AND expired_at<=now` | 删桶对象 → `status=4`（标记已删，避免重复删） |
| **任务行（记录）** | `status IN (2,3,4)` 且 `SELECT 1 FROM export WHERE task_id=? LIMIT 1` 无引用 | `DELETE FROM export_task` |
| **请求行（记录）** | `created_at < now - :retention`（默认 **30 天**，服务配置） | `DELETE FROM export` |

- 与 `object`→`file` 的回收模式一致：**引用判定走扫描、不走计数器**。
- **任务行回收集合含 `status=3`**（2026-10-13 定）：`status=3` 且 `retry >= maxRetry` 是**终态失败**，它没有产出物可删（产出物那条线只看 `status=2`），所以直接并入任务行回收。少了这一档，失败行会只增不减。
- **请求行也要过期**（2026-10-13 定）：`export` 行按 `created_at` 保留 `:retention`，默认 **30 天**。**这是让前两条线真正收敛的那一步**——任务行的"无 `export` 引用"条件只有在请求行被清掉之后才可能成立，否则任务行会一直被钉住、永远删不掉。三条线合起来才构成闭环：产出物过期 → 任务行等请求行散尽 → 请求行到期删除 → 任务行随后被回收。缺第三条，前两条的产出会永久堆在库里。
- **"仍然带无 `export` 引用"这个条件不能省**：还有请求行指着任务时不删。请求行是用户的历史，而任务行是它唯一的状态来源；删早了，用户的列表会从"失败"变成"已删除"。失败行的存活期因此与请求行绑定——这正是上一条必须存在的原因。
- **保留期是服务配置、不是 app 配置**：它是一条**数据保留政策**（"我们留多久历史"），与 `export_ttl`（产物保留期，"这份产物多久后作废"，按 app 配）不是同一层级。取值只有一个硬约束：**`retention` 应当 ≥ `export_ttl` 的最大值**，否则会出现"产物还在、请求行已经没了"的空窗——用户看不到自己的导出记录，字节却还占着桶。
- `status=4` 作为"物理已删"的标记，是因为**删桶对象这个动作本身需要去重**（重复 DELETE 虽幂等但会产生无效调用与日志噪音），这与 `file`/`object` 里"状态可派生"的情形不同。
- 任务行删除后，请求行仍可查（历史记录），只是指向一个已不存在的 task——**这是刻意的**：请求行是用户的历史，不该因为产物过期而消失。此时列表查询 `task_id IN (...)` 查不到，前端显示"已过期/已删除"。**请求行自己到期后整行删除、从列表里消失，这是保留期的正常结果，不是缺陷。**

### 7.7 本表决策带来的连带义务

1. **`md5` 口径统一**：`ExportExcel` 与 `ExportFile` 共用同一个 `exportMd5(spec)` 函数，必须覆盖 headers/rows/files/keys 且**排除 name**，配单测锁死（现网两处口径不一致，`export.go:94` vs `:435`）。
2. **`name` 移出 zip**：xlsx 在 zip 内改用中性名，`Content-Disposition` 用请求行 `name`。这是让去重真正生效的前提。
3. **对象键唯一构造函数** `ExportObjectName(md5) string`，配单测。
4. **`maxRetry` / `backoff` 是代码常量**，与 `ExportObjectName` 同级对待；改动退避表会改变在飞任务的时序，按代码发布处理。
5. **`app_config.export_store_keyword` / `export_ttl` 落地**：`export_store_keyword` 为 NULL 时回落 `store_keyword`（第 2 节）；`export_ttl` 取代 `conf.Export.Expire`（第 2 节）。
6. **worker 取代进程内 goroutine**：现网在 `NewExport` 里 `go func(){ for { clear... } }()`（`service/export.go:52-63`），改为轮询 `export_task` 的 worker 池，池大小是服务配置。生产执行也走同一套 claim/成功/失败路径。
7. **数据面引用不进启动自检**：启动自检只覆盖**配置面**（`store` 可构造——**含 `status=0` 的停用行，见第 1 节"行只增不减"**、`provider` 白名单、`app_config→store` 存在）与配置自洽性（寻址推导、四元组一致、`public_base_url` 前缀互斥）。`file.directory_id` / `file.object_id` / `export.task_id` 这类**数据面**引用由写入路径在事务内保证，不做启动全量扫描——这些表是亿级，扫不动，而它们的写入全部经过服务层。
   （本项原为"删 store 前反查扩到 `export_task`"，随"`store` 行不可删除"作废：既没有删除动作，本表就不需要为反查准备 `store_keyword` 索引，`upload` 同理。）
8. **API 变更**：`GetExportResponse.key` 改为完整 `url`；新增手动重试接口；`ListExportRequest.status` 的值域改为新枚举；`GetExportRequest` 的内容摘要字段统一命名为 `md5`，查询需带 `app` 限定（现网的全局按摘要查询删除）。
9. **删除 `CopyExport` / `GetExportFileCount`**：前者是"压在一张表"的产物，后者是隐式引用计数，均由新模型取代。

### 7.8 服务配置（原"待定项"，2026-10-13 全部定稿）

三条原本的待定项都定了，落成**服务配置**——它们都不随 app 变化，所以不进 `app_config`：

| 配置 | 值 | 说明 |
|---|---|---|
| 租约 `lease` | 单次导出最长耗时的 **2~3 倍**（以 P99 为基准） | worker 心跳周期须显著小于它（建议 `lease/3`）。偏小会把大导出误判成僵死，偏大则僵行一直占着不放 |
| worker 并发上限 | **20** | 池大小。导出占的是本机资源（xlsx 流式写 + 本地 zip + 上传），20 是首版取值，压测后可调 |
| 请求行保留期 `retention` | **30 天** | 7.6 第三条清理线。按 `created_at` 计算，不是按产物过期；须 ≥ `export_ttl` 的最大值 |

