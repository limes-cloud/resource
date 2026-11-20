# 邮件服务 (library/email)

> 包路径：`github.com/limes-cloud/kratosx/library/email`

## 概述

基于 gomail 封装的邮件发送服务，支持模板渲染。

## 配置

```yaml
email:
  user: "noreply@example.com"    # 发件人邮箱
  name: "Resource Service"       # 发件人名称
  password: "smtp-password"      # SMTP 密码
  host: "smtp.example.com"       # SMTP 服务器
  port: 465                      # SMTP 端口
  template:                      # 邮件模板
    verify:
      subject: "验证码"
      path: "templates/verify.html"
      enable: true
      from: "noreply@example.com"
      type: "text/html"
    alert:
      subject: "系统告警"
      path: "templates/alert.html"
```

## Email 接口

```go
type Email interface {
    // Send 发送邮件
    Send(to string, subject string, body string) error

    // SendTemplate 使用模板发送邮件
    SendTemplate(to string, templateName string, data map[string]any) error
}
```

## 使用方式

```go
// 通过 context 获取
email := ctx.Email()

// 发送纯文本邮件
err := email.Send("user@example.com", "测试主题", "邮件正文")

// 使用模板发送
err := email.SendTemplate("user@example.com", "verify", map[string]any{
    "code": "123456",
    "expire": "5分钟",
})
```
