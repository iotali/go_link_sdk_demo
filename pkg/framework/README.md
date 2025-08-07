# IoT Framework

基于事件驱动的 Go IoT 框架，提供了更高层次的抽象，让开发者可以专注于业务逻辑而不必关心底层的连接、协议等细节。

## 核心特性

- **事件驱动架构**：基于事件总线的异步消息处理
- **插件化设计**：核心功能模块化，支持按需加载
- **并发处理**：充分利用 Go 的 goroutine 特性
- **生命周期管理**：完整的设备和框架生命周期管理
- **业务分离**：框架处理基础设施，业务专注逻辑

## 已实现功能

### ✅ 核心框架
- **事件总线**: 完整的发布/订阅机制，支持优先级和异步处理
- **插件管理器**: 动态加载和管理插件，支持生命周期控制
- **设备抽象**: 标准化的设备接口和基础实现
- **并发控制**: 基于 worker pool 的事件处理机制

### ✅ MQTT 插件
- **连接管理**: 自动连接、断线重连、心跳保活
- **主题订阅**: 自动订阅物模型相关主题
- **消息路由**: 智能路由属性设置、服务调用等消息
- **TLS 支持**: 完整的 TLS/SSL 加密连接支持

### ✅ 物模型支持
- **属性上报**: 单个或批量属性上报，支持时间戳
- **属性设置**: 接收并处理云端属性设置请求
- **服务调用**: 完整的服务注册、路由和执行机制（已修复）
- **事件上报**: 支持自定义事件上报（如 timer_complete）

### ✅ 设备功能
- **属性注册**: 灵活的属性定义（读写权限、数据类型）
- **服务注册**: 动态注册服务处理器
- **状态管理**: 设备状态同步和变更通知
- **生命周期**: 完整的初始化、连接、断开、销毁流程

## 快速开始

### 1. 创建设备

```go
type MyDevice struct {
    core.BaseDevice
    // 设备状态
    temperature float64
}

// 实现设备接口
func (d *MyDevice) OnInitialize(ctx context.Context) error {
    // 初始化设备
    return nil
}

func (d *MyDevice) OnConnect(ctx context.Context) error {
    // 连接成功回调
    return nil
}

func (d *MyDevice) OnPropertySet(property core.Property) error {
    // 处理属性设置
    return nil
}
```

### 2. 使用框架

```go
// 创建配置
config := core.Config{
    Device: core.DeviceConfig{
        ProductKey:   "YourProductKey",
        DeviceName:   "YourDeviceName",
        DeviceSecret: "YourDeviceSecret",
    },
    // 其他配置...
}

// 创建框架
framework := core.New(config)

// 初始化
framework.Initialize(config)

// 注册设备
device := &MyDevice{}
framework.RegisterDevice(device)

// 注册事件处理
framework.On(event.EventConnected, func(evt *event.Event) error {
    log.Println("Connected!")
    return nil
})

// 启动框架
framework.Start()

// 等待退出
framework.WaitForShutdown()
```

## 框架架构

### 事件总线 (Event Bus)

事件总线是框架的核心，负责事件的发布和订阅：

```go
// 订阅事件
framework.On(event.EventPropertySet, handler)

// 发布事件
framework.Emit(event.NewEvent(event.EventPropertyReport, "source", data))
```

支持的系统事件：
- `EventConnected` - 连接成功
- `EventDisconnected` - 断开连接
- `EventError` - 错误事件
- `EventReady` - 系统就绪

### 插件系统 (Plugin System)

插件提供了框架的扩展能力：

```go
// 创建插件
type MyPlugin struct {
    plugin.BasePlugin
}

func (p *MyPlugin) Init(ctx context.Context, framework interface{}) error {
    // 初始化插件
    return nil
}

// 加载插件
framework.LoadPlugin(&MyPlugin{})
```

### 设备模型 (Device Model)

设备接口定义了设备的标准行为：

- **生命周期**：`OnInitialize`, `OnConnect`, `OnDisconnect`, `OnDestroy`
- **属性处理**：`OnPropertySet`, `OnPropertyGet`
- **服务处理**：`OnServiceInvoke`
- **事件处理**：`OnEventReceive`
- **OTA处理**：`OnOTANotify`

## 与 SDK 的区别

| 特性 | SDK | Framework |
|-----|-----|-----------|
| 抽象层次 | 低，直接操作 MQTT/HTTP | 高，事件驱动 |
| 使用难度 | 需要了解协议细节 | 专注业务逻辑 |
| 灵活性 | 高，完全控制 | 中，框架约束 |
| 代码量 | 多 | 少 |
| 适用场景 | 需要精细控制 | 快速开发 |

## 完整示例

查看 `examples/framework/simple/main.go` 获取完整的示例代码。

## 待实现功能

### 高优先级 (P1)
- [ ] **服务响应机制**: 服务调用后返回执行结果到云端
- [ ] **错误处理**: 统一的错误处理和重试机制
- [ ] **连接状态管理**: 更智能的连接状态监控和恢复

### 中优先级 (P2)
- [ ] **批量属性设置**: 一次设置多个属性
- [ ] **设备影子**: 本地设备状态缓存和同步
- [ ] **OTA 升级**: 固件和配置的远程升级

### 低优先级 (P3)
- [ ] **设备分组**: 批量管理多个设备
- [ ] **规则引擎**: 本地规则处理
- [ ] **数据持久化**: 离线消息存储和重发
- [ ] **WebSocket 支持**: 除 MQTT 外的连接方式

## 迁移指南

从 SDK 迁移到框架：

1. **设备定义**：创建设备结构体，继承 `BaseDevice`
2. **回调实现**：实现需要的生命周期和业务回调
3. **属性注册**：使用 `RegisterProperty` 替代直接 MQTT 发布
4. **服务注册**：使用 `RegisterService` 替代消息处理
5. **事件处理**：使用事件总线替代直接的消息订阅

## 技术亮点

### 服务路由机制（已修复）
框架现在能正确解析 MQTT 主题并路由服务调用：
```go
// 主题格式: $SYS/{ProductKey}/{DeviceName}/service/{ServiceName}/invoke
// 正确提取服务名: parts[4] (之前错误使用 parts[5])
serviceName := parts[4]  // 例如: "start_timer"
```

### 动态报告频率
根据设备状态智能调整上报频率：
- 正常模式: 30秒上报一次
- 活跃模式: 2秒上报一次（如定时器运行时）

### 事件驱动架构
```go
// 发布事件
framework.Emit(event.NewEvent(eventType, source, data))

// 订阅事件
framework.On(event.EventPropertySet, handler)
```

## 使用示例

### 完整的电烤炉示例
`examples/framework/simple/` 目录包含一个功能完整的智能电烤炉实现：

- **属性管理**: 温度、加热器状态、门状态等
- **服务实现**: 设置温度、启动定时器、切换门状态
- **事件上报**: 定时器完成、过热警告等
- **动态行为**: 根据状态调整上报频率

运行示例：
```bash
cd examples/framework/simple
go build -o oven .
./oven
```

## 注意事项

1. 框架已成功集成真实 IoT 平台，支持完整的设备管理
2. 服务路由机制已修复，能正确处理云端服务调用
3. MQTT 插件完全功能正常，支持 TLS 和非 TLS 连接
4. 事件上报功能已实现，支持自定义事件类型