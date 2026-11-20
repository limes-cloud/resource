# 服务器工具 (server)

> 包路径：`github.com/limes-cloud/kratosx/server`

## 概述

HTTP 服务器的辅助工具，包括 CORS 跨域和响应格式化。

---

## server/cors — 跨域配置

### 配置

```yaml
server:
  http:
    cors:
      allowCredentials: true
      allowOrigins: ["*"]
      allowMethods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
      allowHeaders: ["*"]
      exposeHeaders: ["Content-Length"]
      maxAge: 12h
      allowPrivateNetwork: false
```

### 效果

自动处理 OPTIONS 预检请求，在响应中添加 CORS 相关 Header。

---

## server/response — 响应格式化

### 配置

```yaml
server:
  http:
    formatResponse: true
```

### 效果

启用后，HTTP 响应统一格式化为：

```json
// 成功
{
  "code": 200,
  "msg": "success",
  "data": { ... }
}

// 错误
{
  "code": 400,
  "msg": "参数错误",
  "reason": "ParamsError"
}
```

---

## 静态 Web 服务 (library/web)

### 配置

```yaml
server:
  http:
    webServerDir: "./dist"   # 静态文件目录
```

### 功能

- 自动预加载常用静态资源到内存（html/css/js/json/图片/字体）
- SPA 支持（非文件路由自动返回 index.html）
- 路径安全检查（防止目录遍历）
- 内存缓存加速

### 预加载的文件类型

`.html` `.css` `.js` `.json` `.svg` `.ico` `.png` `.jpg` `.jpeg` `.gif` `.woff` `.woff2` `.ttf` `.eot` `.map`
