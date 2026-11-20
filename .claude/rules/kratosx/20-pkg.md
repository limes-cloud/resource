# 工具包 (pkg)

> 包路径：`github.com/limes-cloud/kratosx/pkg`

## 概述

kratosx 提供了一组常用的工具包，供业务代码使用。

---

## value — 指针与类型转换

> `github.com/limes-cloud/kratosx/pkg/value`

### Pointer 创建指针

```go
import "github.com/limes-cloud/kratosx/pkg/value"

// 泛型指针创建
p := value.Pointer(true)      // *bool
p := value.Pointer("hello")   // *string
p := value.Pointer(42)        // *int

// 类型特化
p := value.Bool(true)         // *bool
p := value.String("hello")    // *string
p := value.Int32(42)          // *int32
p := value.Uint32(1)          // *uint32
p := value.Float64(3.14)      // *float64
```

### Value 安全解引用

```go
// 泛型解引用（nil 安全，返回零值）
v := value.Value(p)   // 如果 p 为 nil，返回类型零值
```

### Transform 结构体转换

```go
import "github.com/limes-cloud/kratosx/pkg/value"

// Proto → Types DTO
var in types.CreateExampleRequest
if err := value.Transform(protoReq, &in); err != nil {
    return errors.TransformError()
}

// Entity → Proto Response
var reply pb.GetExampleResponse
if err := value.Transform(entity, &reply); err != nil {
    return errors.TransformError()
}

// Entity 列表 → Proto 列表
var replyList []*pb.Example
if err := value.Transform(entities, &replyList); err != nil {
    return errors.TransformError()
}
```

Transform 基于 JSON 序列化/反序列化实现结构体之间的字段映射。

---

## cache — 缓存辅助

> `github.com/limes-cloud/kratosx/pkg/cache`

提供内存缓存工具，适合短期数据缓存场景。

---

## crypto — 加密工具

> `github.com/limes-cloud/kratosx/pkg/crypto`

```go
import "github.com/limes-cloud/kratosx/pkg/crypto"

// AES 加密/解密
encrypted, err := crypto.AesEncrypt([]byte(plainText), key)
decrypted, err := crypto.AesDecrypt(encrypted, key)

// MD5
hash := crypto.Md5([]byte("content"))

// SHA256
hash := crypto.Sha256([]byte("content"))

// 密码哈希
hashed, err := crypto.HashPassword("password")
ok := crypto.ComparePassword(hashed, "password")
```

---

## tree — 树形结构

> `github.com/limes-cloud/kratosx/pkg/tree`

将扁平列表构建为树形结构：

```go
import "github.com/limes-cloud/kratosx/pkg/tree"

// 节点需要实现 Tree 接口
type Menu struct {
    ID       uint32
    ParentID uint32
    Children []*Menu
}

func (m *Menu) GetID() uint32       { return m.ID }
func (m *Menu) GetParentID() uint32 { return m.ParentID }
func (m *Menu) SetChildren(c any)   { m.Children = c.([]*Menu) }

// 构建树
menuTree := tree.Build(menuList)
```

---

## ua — User-Agent 解析

> `github.com/limes-cloud/kratosx/pkg/ua`

```go
// 通过 context 获取
agent := ctx.UserAgent()

agent.Platform   // "windows" / "macOS" / "linux" / "iOS" / "android"
agent.OS         // "Windows 10" / "macOS 14.0"
agent.Browser    // "Chrome" / "Firefox" / "Safari"
agent.Version    // "120.0.0"
agent.Mobile     // true / false
agent.Bot        // true / false
```

---

## pkg 根工具

> `github.com/limes-cloud/kratosx/pkg`

```go
import "github.com/limes-cloud/kratosx/pkg"

// 文件路径追加后缀（在扩展名前插入）
newPath := pkg.AppendFileSuffix("image.png", "_thumb")
// 结果: "image_thumb.png"
```
