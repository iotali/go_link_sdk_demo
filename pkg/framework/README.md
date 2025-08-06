# IoT Framework

基于事件驱动的 Go IoT 框架，提供了更高层次的抽象，让开发者可以专注于业务逻辑而不必关心底层的连接、协议等细节。

## 核心特性

- **事件驱动架构**：基于事件总线的异步消息处理
- **插件化设计**：核心功能模块化，支持按需加载
- **并发处理**：充分利用 Go 的 goroutine 特性
- **生命周期管理**：完整的设备和框架生命周期管理
- **业务分离**：框架处理基础设施，业务专注逻辑

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

## 下一步计划

- [ ] 实现 MQTT 插件（迁移现有 MQTT 功能）
- [ ] 实现 OTA 插件（迁移现有 OTA 功能）
- [ ] 实现设备影子插件
- [ ] 实现规则引擎插件
- [ ] 添加更多示例

## 迁移指南

从 SDK 迁移到框架：

1. **设备定义**：创建设备结构体，继承 `BaseDevice`
2. **回调实现**：实现需要的生命周期和业务回调
3. **属性注册**：使用 `RegisterProperty` 替代直接 MQTT 发布
4. **服务注册**：使用 `RegisterService` 替代消息处理
5. **事件处理**：使用事件总线替代直接的消息订阅

## 注意事项

1. 框架还在早期开发阶段，API 可能会有变化
2. 当前示例是模拟的，还没有集成真实的 MQTT 连接
3. 插件系统已实现但还没有实际的插件
4. 需要根据实际需求继续完善功能