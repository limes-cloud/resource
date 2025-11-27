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

type Storage struct {
	Keyword         string        // 存储器标识
	Type            string        // 存储类型
	AntiTheft       bool          // 开启防盗链
	Endpoint        string        // oss连接地址
	AK              string        // AK
	Secret          string        // SK
	Bucket          string        // OSS 存储路径
	Region          string        // OSS 地域
	LocalDir        string        // 本地路径，仅local用
	ServerURL       string        // server地址
	TemporaryExpire time.Duration // 过期时间
	IsExporter      bool          // 是否为导出器
}

type Export struct {
	LocalDir string
	Expire   time.Duration
}

type Conf struct {
	DefaultMaxSize     uint32
	DefaultAcceptTypes []string
	ChunkSize          uint32
	Storage            []*Storage
	Export             *Export
}

func configSource() kconfig.Source {
	host := os.Getenv("CONF_HOST")
	token := os.Getenv("CONF_TOKEN")
	name := os.Getenv("APP_NAME")
	if host != "" && token != "" && name != "" {
		return client.New(host, name, token)
	}
	return file.NewSource("conf/")
}

// configScanWatch 初始化
func configScanWatch(watch config.Watcher) {
	watch("business", func(value config.Value) {
		if err := value.Scan(&conf); err != nil {
			panic(err)
		}
	})
}
