# IoT Framework 设计文档

## 背景与动机

当前的 Go SDK 是从 C SDK 移植而来，主要提供了基础的功能封装。但这种方式没有充分利用 Go 语言的特性：

**C SDK 的限制**：
- 单进程事件循环（资源受限）
- 回调函数处理（代码分散）
- 状态机驱动（复杂度高）
- 同步阻塞模式

**Go 的优势未被利用**：
- 原生并发支持（goroutine）
- Channel 通信机制
- 接口抽象能力
- 内置的并发安全机制

## 设计目标

1. **业务与框架分离**：业务代码只关注业务逻辑，不处理连接、重连、协议等底层细节
2. **事件驱动架构**：基于事件的异步处理，充分利用 Go 的并发特性
3. **插件化扩展**：核心功能模块化，支持按需加载和扩展
4. **声明式编程**：通过配置和注解简化开发
5. **生产级别可靠性**：内置重连、错误处理、监控等机制

## 核心架构

### 分层设计

```
┌─────────────────────────────────────┐
│       Application Layer              │  <- 业务逻辑层
│   (User Business Logic)              │
├─────────────────────────────────────┤
│       Framework Layer                │  <- 框架层
│   (Event Bus, Plugin System)         │
├─────────────────────────────────────┤
│       Component Layer                │  <- 组件层
│   (OTA, Shadow, Rules)               │
├─────────────────────────────────────┤
│       Protocol Layer                 │  <- 协议层
│   (MQTT, HTTP, CoAP)                 │
├─────────────────────────────────────┤
│       Transport Layer                │  <- 传输层
│   (TCP, TLS, WebSocket)              │
└─────────────────────────────────────┘
```

### 核心组件

```
┌──────────────────────────────────────────────────┐
│                  IoT Framework                    │
│                                                   │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐ │
│  │   Event    │  │   Plugin   │  │  Lifecycle │ │
│  │    Bus     │  │   Manager  │  │   Manager  │ │
│  └────────────┘  └────────────┘  └────────────┘ │
│                                                   │
│  ┌────────────────────────────────────────────┐ │
│  │              Core Components                │ │
│  │                                              │ │
│  │  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐   │ │
│  │  │ MQTT │  │ OTA  │  │Shadow│  │ Rule │   │ │
│  │  └──────┘  └──────┘  └──────┘  └──────┘   │ │
│  └────────────────────────────────────────────┘ │
│                                                   │
│  ┌────────────────────────────────────────────┐ │
│  │            Business Interface               │ │
│  │                                              │ │
│  │   OnConnect()  OnMessage()  OnProperty()    │ │
│  │   OnService()  OnEvent()    OnError()       │ │
│  └────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────┘
```

## 实现方案

### Phase 1: 事件驱动核心

#### 1.1 事件总线实现

```go
// pkg/framework/event/event.go
package event

type EventType string

const (
    // 系统事件
    EventConnected    EventType = "system.connected"
    EventDisconnected EventType = "system.disconnected"
    EventError        EventType = "system.error"
    
    // 业务事件
    EventPropertySet  EventType = "property.set"
    EventPropertyGet  EventType = "property.get"
    EventServiceCall  EventType = "service.call"
    EventEventEmit    EventType = "event.emit"
    
    // OTA事件
    EventOTANotify    EventType = "ota.notify"
    EventOTAProgress  EventType = "ota.progress"
    EventOTAComplete  EventType = "ota.complete"
)

type Event struct {
    Type      EventType
    Source    string
    Timestamp time.Time
    Data      interface{}
    Context   context.Context
}

type Handler func(event Event) error

type EventBus interface {
    Subscribe(eventType EventType, handler Handler) error
    Unsubscribe(eventType EventType, handler Handler) error
    Publish(event Event) error
    Start() error
    Stop() error
}
```

#### 1.2 框架核心接口

```go
// pkg/framework/core/framework.go
package core

type Framework interface {
    // 生命周期管理
    Initialize(config Config) error
    Start() error
    Stop() error
    
    // 设备管理
    RegisterDevice(device Device) error
    UnregisterDevice(deviceID string) error
    
    // 插件管理
    LoadPlugin(plugin Plugin) error
    UnloadPlugin(name string) error
    
    // 事件管理
    On(eventType EventType, handler Handler) error
    Emit(event Event) error
}

type Device interface {
    // 设备信息
    GetDeviceInfo() DeviceInfo
    
    // 生命周期回调
    OnInitialize(ctx context.Context) error
    OnConnect(ctx context.Context) error
    OnDisconnect(ctx context.Context) error
    OnDestroy(ctx context.Context) error
    
    // 业务回调
    OnPropertyChange(property Property) error
    OnServiceInvoke(service Service) (interface{}, error)
    OnEventReceive(event DeviceEvent) error
}
```

### Phase 2: 插件系统

#### 2.1 插件接口定义

```go
// pkg/framework/plugin/plugin.go
package plugin

type Plugin interface {
    // 插件信息
    Name() string
    Version() string
    Description() string
    
    // 生命周期
    Init(ctx context.Context, framework Framework) error
    Start() error
    Stop() error
    
    // 依赖管理
    Dependencies() []string
    
    // 配置管理
    Configure(config map[string]interface{}) error
}

type PluginManager interface {
    Register(plugin Plugin) error
    Unregister(name string) error
    Get(name string) (Plugin, error)
    List() []Plugin
    
    // 生命周期管理
    InitAll() error
    StartAll() error
    StopAll() error
}
```

#### 2.2 内置插件

```go
// 1. MQTT插件 - 处理MQTT连接和消息
// 2. OTA插件 - 处理固件升级
// 3. Shadow插件 - 设备影子同步
// 4. Rules插件 - 规则引擎
// 5. Metrics插件 - 监控指标
// 6. Storage插件 - 数据持久化
```

### Phase 3: 声明式设备定义

#### 3.1 设备模型定义

```go
// pkg/framework/model/device.go
package model

// 使用结构体标签定义设备模型
type SmartLamp struct {
    // 属性定义
    Power      bool    `iot:"property,rw" json:"power"`
    Brightness int     `iot:"property,rw,min=0,max=100" json:"brightness"`
    Color      string  `iot:"property,rw" json:"color"`
    
    // 只读属性
    Temperature float64 `iot:"property,r" json:"temperature"`
    
    // 服务定义
    Reset      func() error                     `iot:"service"`
    SetTimer   func(minutes int) error          `iot:"service"`
    GetStatus  func() (map[string]interface{}, error) `iot:"service"`
}

// 事件定义
func (l *SmartLamp) EmitOverheat() {
    framework.Emit(Event{
        Type: "overheat",
        Data: map[string]interface{}{
            "temperature": l.Temperature,
        },
    })
}
```

#### 3.2 自动注册机制

```go
// 框架自动解析标签并注册
func RegisterModel(device interface{}) error {
    // 1. 解析结构体标签
    // 2. 自动生成属性列表
    // 3. 自动生成服务列表
    // 4. 注册到框架
}
```

### Phase 4: 业务开发示例

#### 4.1 最简示例

```go
package main

import (
    "github.com/iot-go-sdk/framework"
)

type MyDevice struct {
    temperature float64
}

func (d *MyDevice) OnPropertyChange(prop Property) error {
    switch prop.Name {
    case "temperature":
        d.temperature = prop.Value.(float64)
    }
    return nil
}

func main() {
    // 创建框架实例
    fw := framework.New(framework.Config{
        ProductKey:   "xxx",
        DeviceName:   "xxx",
        DeviceSecret: "xxx",
    })
    
    // 注册设备
    fw.RegisterDevice(&MyDevice{})
    
    // 运行框架
    fw.Run() // 阻塞运行
}
```

#### 4.2 完整示例

```go
package main

import (
    "context"
    "log"
    
    "github.com/iot-go-sdk/framework"
    "github.com/iot-go-sdk/framework/plugins/ota"
    "github.com/iot-go-sdk/framework/plugins/shadow"
)

type SmartSensor struct {
    // 设备状态
    temperature float64
    humidity    float64
    online      bool
    
    // 框架引用
    framework framework.Framework
}

// 实现 Device 接口
func (s *SmartSensor) OnInitialize(ctx context.Context) error {
    log.Println("Device initializing...")
    
    // 注册属性
    s.framework.RegisterProperty("temperature", s.GetTemperature, nil)
    s.framework.RegisterProperty("humidity", s.GetHumidity, nil)
    
    // 注册服务
    s.framework.RegisterService("calibrate", s.Calibrate)
    
    // 注册事件处理
    s.framework.On(event.EventOTANotify, s.HandleOTA)
    
    return nil
}

func (s *SmartSensor) OnConnect(ctx context.Context) error {
    s.online = true
    
    // 上报初始状态
    s.framework.ReportProperties(map[string]interface{}{
        "temperature": s.temperature,
        "humidity":    s.humidity,
        "online":      s.online,
    })
    
    return nil
}

func (s *SmartSensor) HandleOTA(event Event) error {
    task := event.Data.(OTATask)
    
    // 业务判断是否接受升级
    if s.ShouldUpgrade(task.Version) {
        return s.framework.AcceptOTA(task)
    }
    
    return s.framework.RejectOTA(task, "Version not compatible")
}

func main() {
    // 配置
    config := framework.Config{
        Device: framework.DeviceConfig{
            ProductKey:   "xxx",
            DeviceName:   "xxx",
            DeviceSecret: "xxx",
        },
        MQTT: framework.MQTTConfig{
            Host:      "iot.example.com",
            Port:      8883,
            UseTLS:    true,
            KeepAlive: 60,
        },
    }
    
    // 创建框架
    fw := framework.New(config)
    
    // 加载插件
    fw.LoadPlugin(ota.NewPlugin())
    fw.LoadPlugin(shadow.NewPlugin())
    
    // 注册设备
    sensor := &SmartSensor{framework: fw}
    fw.RegisterDevice(sensor)
    
    // 启动框架
    if err := fw.Start(); err != nil {
        log.Fatal(err)
    }
    
    // 等待退出
    fw.WaitForShutdown()
}
```

## 实施计划

### 第一阶段：核心框架（Week 1-2）
- [ ] 实现事件总线
- [ ] 实现框架核心接口
- [ ] 实现生命周期管理
- [ ] 创建基础示例

### 第二阶段：插件系统（Week 3-4）
- [ ] 实现插件管理器
- [ ] 迁移MQTT功能为插件
- [ ] 迁移OTA功能为插件
- [ ] 实现插件依赖管理

### 第三阶段：设备模型（Week 5）
- [ ] 实现声明式模型解析
- [ ] 实现属性自动注册
- [ ] 实现服务自动注册
- [ ] 添加验证机制

### 第四阶段：高级特性（Week 6）
- [ ] 实现设备影子插件
- [ ] 实现规则引擎插件
- [ ] 实现监控指标插件
- [ ] 添加更多示例

### 第五阶段：优化与文档（Week 7）
- [ ] 性能优化
- [ ] 完善错误处理
- [ ] 编写用户文档
- [ ] 编写迁移指南

## 兼容性策略

为了不破坏现有SDK的使用，我们采用以下策略：

1. **并行开发**：框架代码放在 `pkg/framework/` 目录下，不影响现有 `pkg/` 下的SDK代码
2. **示例分离**：框架示例放在 `examples/framework/` 目录下，保留原有 `examples/` 
3. **版本标记**：使用 git tag 标记SDK版本（v1.x.x-sdk）和框架版本（v2.x.x-framework）
4. **渐进迁移**：提供迁移工具和指南，帮助用户从SDK迁移到框架

## 性能考虑

1. **并发模型**：每个插件运行在独立的 goroutine 中
2. **背压控制**：使用带缓冲的 channel 和限流机制
3. **内存管理**：使用对象池减少 GC 压力
4. **批量处理**：支持批量上报和批量处理

## 监控与调试

1. **内置Metrics**：暴露 Prometheus 格式的指标
2. **分布式追踪**：支持 OpenTelemetry
3. **日志分级**：结构化日志with context
4. **调试模式**：详细的事件流日志

## 下一步行动

1. 根据本文档创建框架的基础结构
2. 实现核心的事件总线
3. 创建第一个可运行的框架示例
4. 逐步迁移现有功能为插件

## 参考资料

- [Go Concurrency Patterns](https://go.dev/blog/pipelines)
- [Actor Model in Go](https://github.com/asynkron/protoactor-go)
- [Event-Driven Architecture](https://martinfowler.com/articles/201701-event-driven.html)
- [Plugin Architecture](https://eli.thegreenplace.net/2021/plugins-in-go/)