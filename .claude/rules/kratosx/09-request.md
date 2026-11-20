# HTTP 请求客户端 (library/request)

> 包路径：`github.com/limes-cloud/kratosx/library/request`

## 概述

基于 [go-resty/resty](https://github.com/go-resty/resty/v2) 封装的 HTTP 请求客户端，支持重试、超时、日志记录。

## 配置

```yaml
request:
  enableLog: true         # 是否记录请求日志
  retryCount: 3           # 最大重试次数
  retryWaitTime: 1s       # 重试等待时间
  maxRetryWaitTime: 5s    # 最大重试等待时间
  timeout: 60s            # 请求超时
  userAgent: "resource-service"  # User-Agent
```

## Request 接口

```go
type Request interface {
    // Option 设置 resty 请求参数
    Option(fn Func) Request

    // Get 发送 GET 请求
    Get(url string) (*response, error)

    // Post 发送 POST 请求
    Post(url string, data any) (*response, error)

    // PostJson 发送 JSON POST 请求
    PostJson(url string, data any) (*response, error)

    // Put 发送 PUT 请求
    Put(url string, data any) (*response, error)

    // PutJson 发送 JSON PUT 请求
    PutJson(url string, data any) (*response, error)

    // Delete 发送 DELETE 请求
    Delete(url string) (*response, error)

    // Do 执行已配置的请求
    Do() (*response, error)
}
```

## 使用方式

### 基本请求

```go
// 通过 context 获取
req := ctx.Request()

// GET 请求
resp, err := req.Get("https://api.example.com/data")
if err != nil {
    return err
}

// 解析 JSON 响应
var result MyResponse
if err := resp.Result(&result); err != nil {
    return err
}

// 获取原始字节
body := resp.Body()
```

### POST 请求

```go
// JSON POST
resp, err := ctx.Request().PostJson("https://api.example.com/create", map[string]any{
    "name": "test",
    "value": 123,
})

// Form POST
resp, err := ctx.Request().Post("https://api.example.com/upload", formData)
```

### 自定义选项

```go
resp, err := ctx.Request().Option(func(r *resty.Request) {
    r.SetHeader("X-Custom", "value")
    r.SetQueryParam("page", "1")
    r.SetAuthToken("bearer-token")
}).Get("https://api.example.com/data")
```

### 覆盖默认配置

```go
// 临时覆盖超时和重试
resp, err := ctx.Request(
    request.WithTimeout(5 * time.Second),
    request.WithRetryCount(1),
    request.WithEnableLog(false),
).PostJson(url, data)
```

## 日志输出

启用日志时，每次请求会记录：
- 请求次数（attempt）
- HTTP 方法
- URL
- Header
- Body
- 耗时（ms）
- 响应内容
