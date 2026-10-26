package core

import (
	"os"
	"time"

	kconfig "github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/limes-cloud/configure/api/configure/client"
	"github.com/limes-cloud/kratosx/config"
)

var conf = &Conf{}

// Upload 上传与分片配置
type Upload struct {
	SessionTtl time.Duration // 分片会话过期；GC 按 updated_at 扫
	GcInterval time.Duration // GC 扫描周期
	PartUrlTtl time.Duration // 预签名分片 URL 有效期；覆盖"一个批次从下发到传完"，非整份文件
}

// Export 导出（⏸ 首版不实现，配置保留为设计占位，导出功能启动时才读取）
type Export struct {
	LocalDir             string        // 本地工作目录；容量调度与本地清理见 .prd/草案-导出本地目录与容量调度.md
	Lease                time.Duration // 租约时长；单次导出最长耗时的 2~3 倍（P99 基准），同时复用为导出亲和锁 TTL
	Heartbeat            time.Duration // worker 心跳周期（顶 leased_at）；须 < lease，建议 lease/3
	WorkerCount          int           // worker 池大小
	Retention            time.Duration // 请求行保留期（30 天）；须 >= export_ttl 最大值
	CleanupInterval      time.Duration // 产出物清理周期（expired_at 到期 → 删桶对象 → status=4）
	LeaseReclaimInterval time.Duration // 租约恢复周期（leased_at 超租约 → status=3）
	TaskReclaimInterval  time.Duration // 任务行回收周期（status IN (2,3,4) 且无 export 引用 → DELETE）
}

// ObjectAudit 离线对象核对（回收 A 类残留）
type ObjectAudit struct {
	Enable    bool          // 是否启用
	Interval  time.Duration // 扫描周期
	BatchSize int           // 单批扫描行数
}

// AppConfigCache app_config 的 Redis 缓存口径
type AppConfigCache struct {
	Ttl         time.Duration // 正缓存兜底 TTL（整行快照）
	NegativeTtl time.Duration // 空结果负缓存 TTL（与正缓存同 key，空串即哨兵）
}

// Conf business 段服务配置
type Conf struct {
	Upload         *Upload         // 上传与分片
	Export         *Export         // 导出（首版不读取）
	ObjectAudit    *ObjectAudit    // 离线对象核对
	AppConfigCache *AppConfigCache // app_config 缓存
}

// configSource 配置源：三个环境变量齐全时走远程配置中心，否则回落 conf/ 目录
func configSource() kconfig.Source {
	host := os.Getenv("CONF_HOST")
	token := os.Getenv("CONF_TOKEN")
	name := os.Getenv("APP_NAME")
	if host != "" && token != "" && name != "" {
		return client.New(host, name, token)
	}
	return file.NewSource("conf/")
}

// configScanWatch 监听 business 段热更新，刷新全局单例 conf
func configScanWatch(watch config.Watcher) {
	watch("business", func(value config.Value) {
		if err := value.Scan(&conf); err != nil {
			panic(err)
		}
	})
}
