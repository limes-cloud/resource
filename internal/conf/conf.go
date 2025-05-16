package conf

import "time"

const (
	STORE_ALIYUN  = "aliyun"
	STORE_TENCENT = "tencent"
	STORE_BAIDU   = "baidu"
	STORE_LOCAL   = "local"
)

type Storage struct {
	Keyword         string        // 存储关键字
	IsExporter      bool          // 是否为导出器
	Type            string        // 存储类型
	AntiTheft       bool          // 开启防盗链
	Endpoint        string        // oss连接地址
	Id              string        // AK
	Secret          string        // SK
	Bucket          string        // OSS 存储路径
	Region          string        // OSS 地域
	LocalDir        string        // 本地路径，仅local用
	ServerURL       string        // server地址仅local用
	TemporaryExpire time.Duration // 过期时间
}

type Config struct {
	// Secret             string
	// Expire             time.Duration
	DefaultMaxSize uint32

	DefaultAcceptTypes []string
	ChunkSize          uint32
	Export             struct {
		ServerURL string
		LocalDir  string
		Expire    time.Duration
	}
	Storages []*Storage
}

func (c *Config) GetLocalStorage() *Storage {
	for _, storage := range c.Storages {
		if storage.Type == STORE_LOCAL {
			return storage
		}
	}
	return nil
}
