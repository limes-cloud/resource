# 服务注册与发现 (library/registry)

> 包路径：`github.com/limes-cloud/kratosx/library/registry`

## 概述

服务注册与发现抽象层，目前内置 Consul 实现。通过 DSN 格式配置。

## 配置

```yaml
server:
  registry: "consul://127.0.0.1:8500?token=xxx&datacenter=dc1"
```

## DSN 格式

```
scheme://host:port?param1=value1&param2=value2
```

| 参数 | 说明 |
|------|------|
| `token` | Consul ACL Token |
| `datacenter` | 数据中心名称 |

## 注册新的 Registry 实现

```go
import "github.com/limes-cloud/kratosx/library/registry"

func init() {
    registry.Register("etcd", func(dsn *url.URL) (registry.Interface, error) {
        // 创建 etcd 注册中心实例
        return etcdRegistry, nil
    })
}
```

使用时配置 DSN：

```yaml
server:
  registry: "etcd://127.0.0.1:2379"
```

## Interface 接口

```go
type Interface interface {
    kregistry.Registrar   // 服务注册
    kregistry.Discovery   // 服务发现
}
```
