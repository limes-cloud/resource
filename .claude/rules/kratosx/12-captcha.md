# 验证码 (library/captcha)

> 包路径：`github.com/limes-cloud/kratosx/library/captcha`

## 概述

图形验证码服务，支持多场景配置、刷新限制、设备唯一性校验。基于 Redis 存储验证码状态。

## 配置

```yaml
captcha:
  login:                    # 场景名称
    length: 4               # 验证码长度
    expire: 5m              # 过期时间
    redis: "cache"          # 使用的 Redis 实例
    refreshTime: 60s        # 刷新冷却时间
    limit: 5                # 最大尝试次数
    uniqueDevice: false     # 是否绑定设备
  register:
    length: 6
    expire: 10m
    redis: "cache"
```

## PCaptcha 接口

```go
type PCaptcha interface {
    // Image 生成图形验证码
    Image(scene string, height, width int) (id string, content string, err error)

    // Verify 验证验证码
    Verify(scene string, id string, answer string) bool
}
```

## 使用方式

```go
captcha := ctx.Captcha()

// 生成验证码（返回 base64 图片）
id, base64Img, err := captcha.Image("login", 40, 120)

// 验证
ok := captcha.Verify("login", id, userInput)
```
