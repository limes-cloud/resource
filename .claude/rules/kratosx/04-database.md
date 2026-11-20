# 数据库 (library/db)

> 包路径：`github.com/limes-cloud/kratosx/library/db`

## 概述

封装 GORM 数据库管理，支持多数据库实例、自动建库、事务传播、慢查询日志、错误转换和数据初始化。

## 配置

```yaml
database:
  - enable: true
    name: resource        # 实例名称（ctx.DB("resource")）
    drive: postgres       # postgres / mysql / sqlserver
    autoCreate: false     # 自动创建数据库
    connect:
      host: 127.0.0.1
      port: 5432
      dbName: resource
      username: postgres
      password: postgres
    config:
      logLevel: 1             # 0=Silent 1=Error 2=Warn 3=Info
      slowThreshold: 200ms    # 慢查询阈值
      maxLifetime: 2h         # 连接最大生命周期
      maxOpenConn: 20         # 最大打开连接数
      maxIdleConn: 10         # 最大空闲连接数
      prepareStmt: false      # 预编译模式
      dryRun: false           # DryRun 模式
      tablePrefix: ""         # 表名前缀
      transformError:         # 错误转换
        enable: true
        format:
          duplicated: "数据已存在"
          addForeign: "关联数据不存在"
          delForeign: "请先删除关联数据"
      initializer:            # 数据初始化器
        enable: false
        force: false          # 强制重新初始化
        path: deploy/data.sql
```

## 使用方式

### 获取数据库实例

```go
// 获取默认（第一个）数据库
db := ctx.DB()

// 获取指定名称的数据库
db := ctx.DB("resource")
```

### 事务

```go
// 开启事务（支持嵌套事务传播）
err := ctx.Transaction(func(ctx kratosx.Context) error {
    // 闭包内的 ctx.DB() 自动使用事务连接
    if err := ctx.DB().Create(&entity1).Error; err != nil {
        return err  // 返回 error 自动回滚
    }
    if err := ctx.DB().Create(&entity2).Error; err != nil {
        return err
    }
    return nil  // 返回 nil 自动提交
})

// 指定数据库的事务
err := ctx.Transaction(func(ctx kratosx.Context) error {
    return ctx.DB("other").Create(&entity).Error
}, "other")
```

### 事务传播

嵌套调用 `Transaction` 时，如果当前 context 已在事务中，不会创建新事务：

```go
err := ctx.Transaction(func(ctx kratosx.Context) error {
    // 内部再次调用 Transaction，复用外层事务
    return ctx.Transaction(func(ctx kratosx.Context) error {
        return ctx.DB().Create(&entity).Error
    })
})
```

## DB 接口

```go
type DB interface {
    // Get 获取指定名称的 gorm.DB 实例
    Get(name ...string) *gorm.DB

    // TxKey 获取事务在 context 中的 key
    TxKey(name ...string) string
}
```

## 错误转换 (gormtranserror)

将 GORM 底层错误转换为用户友好的错误信息：

- **Duplicated Key** → 自定义 "数据已存在" 消息
- **Foreign Key (Add)** → "关联数据不存在"
- **Foreign Key (Delete)** → "请先删除关联数据"

启用方式：在配置中设置 `transformError.enable: true`。

## 数据初始化器 (initializer)

支持从 SQL 文件自动初始化数据：

```yaml
config:
  initializer:
    enable: true
    force: false          # true = 每次启动都重新执行
    path: deploy/data.sql
```

初始化器会在首次启动时执行 SQL 文件，通过内部记录避免重复执行。

## 跳过 DB 钩子

某些场景需要跳过 DB 层的自定义钩子：

```go
ctx := kratosx.MustContext(c, kratosx.WithSkipDBHook())
ctx.DB().Create(&entity)
```

## 日志

GORM 操作日志通过 kratosx logger 输出，包含：
- SQL 语句
- 执行耗时
- 影响行数
- 慢查询标记
