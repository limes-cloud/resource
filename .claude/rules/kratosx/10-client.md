# gRPC 客户端 (library/client)

> 包路径：`github.com/limes-cloud/kratosx/library/client`

## 概述

gRPC 客户端连接管理，支持服务发现、负载均衡、签名、TLS 等特性。

## 配置

```yaml
client:
  - server: user-service    # 服务名称
    type: direct            # 连接类型：direct（直连）/ discovery（服务发现）
    timeout: 5s             # 连接超时
    maxRecvSize: 4194304    # 最大接收消息大小（4MB）
    maxSendSize: 4194304    # 最大发送消息大小
    metadata:               # 透传元数据
      x-app-id: "resource"
    backends:               # 后端地址列表（direct 模式）
      - target: "127.0.0.1:8010"
        weight: 10
      - target: "127.0.0.1:8011"
        weight: 5
    signature:              # 客户端签名配置
      enable: true
      ak: "app-key"
      sk: "secret-key"
    tls:                    # TLS 配置
      pem: "..."
      key: "..."
```

## 使用方式

### 获取 gRPC 连接

```go
// 通过 context 获取 gRPC 连接
conn, err := ctx.GrpcConn("user-service")
if err != nil {
    return err
}

// 创建 gRPC 客户端
client := pb.NewUserServiceClient(conn)
resp, err := client.GetUser(ctx.Ctx(), &pb.GetUserRequest{Id: userId})
```

### 连接类型

| 类型 | 说明 |
|------|------|
| `direct` | 直连模式，使用 `backends` 中的地址列表，支持加权负载均衡 |
| `discovery` | 服务发现模式，通过 Registry 自动发现服务实例 |

### 负载均衡

direct 模式支持加权轮询负载均衡：

```yaml
backends:
  - target: "10.0.0.1:8010"
    weight: 10    # 权重高，分配更多流量
  - target: "10.0.0.2:8010"
    weight: 5
```

## 注意事项

- 客户端连接在首次调用时建立，之后复用
- 配置 `signature` 后，每次请求会自动附加签名 Header
- 元数据通过 kratos metadata 中间件自动透传
