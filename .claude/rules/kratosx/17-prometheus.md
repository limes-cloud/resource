# Prometheus 监控 (library/prometheus)

> 包路径：`github.com/limes-cloud/kratosx/library/prometheus`

## 概述

Prometheus 指标管理，支持 Counter、Gauge、Histogram、Summary 四种指标类型，配置驱动。

## 配置

```yaml
metrics: true    # 启用 /metrics 端点

prometheus:
  - name: task_total
    type: counter
    namespace: app
    subsystem: worker
    help: "任务执行总数"
    labels: ["type", "status"]

  - name: request_latency
    type: histogram
    namespace: app
    subsystem: server
    help: "请求延迟"
    labels: ["name"]
    buckets: [0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0]

  - name: active_connections
    type: gauge
    namespace: app
    subsystem: pool
    help: "活跃连接数"
    labels: ["type"]

  - name: call_duration
    type: summary
    namespace: app
    subsystem: client
    help: "下游调用耗时"
    labels: ["name"]
    objectives:
      0.5: 0.05
      0.9: 0.01
      0.99: 0.001
```

## Prometheus 接口

```go
type Prometheus interface {
    CounterVec(name string) *prometheus.CounterVec
    GaugeVec(name string) *prometheus.GaugeVec
    HistogramVec(name string) *prometheus.HistogramVec
    SummaryVec(name string) *prometheus.SummaryVec
}
```

## 使用方式

```go
prom := prometheus.Instance()

// Counter — 计数
prom.CounterVec("task_total").
    WithLabelValues("job", "success").Inc()

// Histogram — 延迟
prom.HistogramVec("request_latency").
    WithLabelValues("list").Observe(0.123)

// Gauge — 当前值
prom.GaugeVec("active_connections").
    WithLabelValues("db").Set(42)

// Summary — 分位数
prom.SummaryVec("call_duration").
    WithLabelValues("remote").Observe(0.5)
```

## 辅助类型

框架提供了更简洁的封装类型：

```go
// Counter
c := prometheus.NewCounter(prom.CounterVec("task_total"))
c.With("job", "success").Inc()
c.With("job", "failed").Add(3)

// Gauge
g := prometheus.NewGauge(prom.GaugeVec("active_connections"))
g.With("db").Set(42)
g.With("db").Add(1)
g.With("db").Sub(1)

// Histogram / Summary (统一 Observer 接口)
h := prometheus.NewHistogram(prom.HistogramVec("request_latency"))
h.With("list").Observe(0.123)
```

## /metrics 端点

启用 `metrics: true` 后，HTTP Server 自动注册 `/metrics` 路由，暴露 Prometheus 指标。
