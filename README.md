# IoT Go SDK

基于 Go 语言的物联网设备 SDK，提供与 IoT 平台对接的完整功能，包括 MQTT 连接、动态注册、RRPC 远程调用等核心功能。

## 功能特性

- ✅ **三元组认证**: 支持 ProductKey、DeviceName、DeviceSecret 认证
- ✅ **MQTT 连接**: 基础 MQTT 连接和消息收发
- ✅ **TLS 安全连接**: 支持 MQTT over TLS 安全传输
- ✅ **一机一密**: HTTP 协议动态注册获取设备密钥
- ✅ **动态一机一密**: MQTT 协议动态注册，支持免白名单模式
- ✅ **RRPC 功能**: 远程过程调用，支持同步请求响应
- ✅ **自动重连**: 网络断开自动重连机制
- ✅ **证书管理**: 自定义 CA 证书支持
- ✅ **安全模式**: 支持 securemode=2/3，自动适配 TLS/非TLS 连接
- ✅ **IoT Framework**: 事件驱动框架，提供更高层次的抽象
- ✅ **物模型支持**: 完整的属性、服务、事件支持
- ✅ **插件化架构**: 模块化设计，支持按需扩展

## 快速开始

### 安装

```bash
go mod init your-project
go get github.com/iot-go-sdk
```

### 基础 MQTT 连接

```go
package main

import (
    "log"
    "time"
    
    "github.com/iot-go-sdk/pkg/config"
    "github.com/iot-go-sdk/pkg/mqtt"
)

func main() {
    // 创建配置
    cfg := config.NewConfig()
    cfg.Device.ProductKey = "your_product_key"
    cfg.Device.DeviceName = "your_device_name"
    cfg.Device.DeviceSecret = "your_device_secret"
    cfg.MQTT.Host = "your_mqtt_host"
    cfg.MQTT.Port = 1883
    
    // 创建 MQTT 客户端
    client := mqtt.NewClient(cfg)
    
    // 连接
    if err := client.Connect(); err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect()
    
    // 订阅消息
    topic := "/your_product_key/your_device_name/user/update"
    client.Subscribe(topic, 0, func(topic string, payload []byte) {
        log.Printf("Received: %s", payload)
    })
    
    // 发布消息
    client.Publish(topic, []byte("Hello IoT"), 0, false)
}
```

### TLS 安全连接

```go
cfg.MQTT.UseTLS = true
cfg.MQTT.Port = 8883
cfg.TLS.SkipVerify = false  // 验证服务器证书

// 可选：手动指定安全模式（默认会自动判断）
cfg.MQTT.SecureMode = "2"   // TLS 连接使用 securemode=2
```

### 安全模式说明

SDK 支持两种安全模式，与 C SDK 完全兼容：

- **securemode=2**: 用于 TLS 加密连接，默认端口 8883
- **securemode=3**: 用于非 TLS 连接，默认端口 1883

SDK 会根据 `UseTLS` 配置自动选择合适的安全模式，也可以手动指定：

```go
// 自动判断（推荐）
cfg.MQTT.UseTLS = true   // 自动使用 securemode=2
cfg.MQTT.UseTLS = false  // 自动使用 securemode=3

// 手动指定
cfg.MQTT.SecureMode = "2"  // 强制使用 securemode=2
cfg.MQTT.SecureMode = "3"  // 强制使用 securemode=3
```

### HTTP 动态注册

```go
import "github.com/iot-go-sdk/pkg/dynreg"

cfg.Device.ProductSecret = "your_product_secret"
client := dynreg.NewHTTPDynRegClient(cfg)

deviceSecret, err := client.Register()
if err != nil {
    log.Fatal(err)
}
log.Printf("Device Secret: %s", deviceSecret)
```

### MQTT 动态注册

```go
client := dynreg.NewMQTTDynRegClient(cfg)
responseData, err := client.Register(true, 60*time.Second)
if err != nil {
    log.Fatal(err)
}
// 使用返回的连接凭据
```

### RRPC 远程调用

```go
import "github.com/iot-go-sdk/pkg/rrpc"

rrpcClient := rrpc.NewRRPCClient(mqttClient, productKey, deviceName)

// 注册处理器
rrpcClient.RegisterHandler("LightSwitch", func(requestId string, payload []byte) ([]byte, error) {
    response := map[string]interface{}{"LightSwitch": 0}
    return json.Marshal(response)
})

// 启动 RRPC 服务
rrpcClient.Start()
```

## IoT Framework（新增）

基于事件驱动的 IoT 框架，提供更高层次的抽象，让开发者可以专注于业务逻辑而不必关心底层连接、协议等细节。

### 框架特性

- **事件驱动架构**: 基于事件总线的异步消息处理
- **插件化设计**: 核心功能模块化，支持按需加载
- **物模型支持**: 完整的属性、服务、事件处理
- **生命周期管理**: 完整的设备和框架生命周期管理
- **并发控制**: 基于 worker pool 的高效事件处理

### 快速上手

```go
import (
    "github.com/iot-go-sdk/pkg/framework/core"
    "github.com/iot-go-sdk/pkg/framework/event"
    "github.com/iot-go-sdk/pkg/framework/plugins/mqtt"
)

// 创建框架配置
frameworkConfig := core.Config{
    Device: core.DeviceConfig{
        ProductKey:   "YourProductKey",
        DeviceName:   "YourDeviceName",
        DeviceSecret: "YourDeviceSecret",
    },
    MQTT: core.MQTTConfig{
        Host: "your_mqtt_host",
        Port: 1883,
    },
}

// 创建框架实例
framework := core.New(frameworkConfig)

// 初始化并加载 MQTT 插件
framework.Initialize(frameworkConfig)
mqttPlugin := mqtt.NewMQTTPlugin(sdkConfig)
framework.LoadPlugin(mqttPlugin)

// 注册设备
device := NewYourDevice()
framework.RegisterDevice(device)

// 注册事件处理器
framework.On(event.EventConnected, func(evt *event.Event) error {
    log.Println("Connected to IoT platform")
    return nil
})

// 启动框架
framework.Start()
framework.WaitForShutdown()
```

### 设备实现

```go
type MyDevice struct {
    core.BaseDevice
    temperature float64
}

// 实现设备接口
func (d *MyDevice) OnPropertySet(property core.Property) error {
    // 处理属性设置
    return nil
}

func (d *MyDevice) OnServiceInvoke(service string, params map[string]interface{}) (interface{}, error) {
    // 处理服务调用
    return nil, nil
}
```

### 框架示例 - 智能电烤炉

`examples/framework/simple/` 目录包含一个功能完整的智能电烤炉实现：

- **属性管理**: 温度、加热器状态、门状态等
- **服务实现**: 设置温度、启动定时器、切换门状态
- **事件上报**: 定时器完成、过热警告等
- **动态行为**: 根据状态智能调整上报频率（正常30秒，活跃2秒）

```bash
cd examples/framework/simple
go build -o oven .
./oven
```

## 项目结构

```
iot-go-sdk/
├── pkg/
│   ├── config/          # 配置管理
│   ├── auth/            # 认证模块
│   ├── mqtt/            # MQTT 客户端
│   ├── dynreg/          # 动态注册
│   ├── rrpc/            # RRPC 功能
│   ├── tls/             # TLS 证书管理
│   └── framework/       # IoT 框架
│       ├── core/        # 框架核心
│       ├── event/       # 事件系统
│       ├── device/      # 设备抽象
│       └── plugins/     # 插件系统
│           └── mqtt/    # MQTT 插件
├── examples/            # 示例代码
│   ├── basic_mqtt/      # 基础 MQTT 连接示例
│   ├── tls_mqtt/        # TLS MQTT 连接示例
│   ├── dynreg_http/     # HTTP 动态注册示例
│   ├── dynreg_mqtt/     # MQTT 动态注册示例
│   ├── rrpc/            # RRPC 示例
│   └── framework/       # 框架示例
│       └── simple/      # 智能设备示例
└── README.md
```

## 示例运行

### 基础 MQTT 连接

```bash
cd examples/basic_mqtt
go run main.go
```

### TLS 连接

```bash
cd examples/tls_mqtt
go run main.go
```

### HTTP 动态注册

```bash
cd examples/dynreg_http
go run main.go
```

### MQTT 动态注册

```bash
cd examples/dynreg_mqtt
go run main.go
```

### RRPC 功能

```bash
cd examples/rrpc
go run main.go
```

### 框架示例

```bash
cd examples/framework/simple
go build -o oven .
./oven
```

## 配置参数

### 设备配置

| 参数 | 说明 | 必填 |
|------|------|------|
| ProductKey | 产品密钥 | ✅ |
| DeviceName | 设备名称 | ✅ |
| DeviceSecret | 设备密钥 | ✅* |
| ProductSecret | 产品密钥 | ✅* |

*注：DeviceSecret 和 ProductSecret 至少需要一个

### MQTT 配置

| 参数 | 说明 | 默认值 |
|------|------|-------|
| Host | MQTT 服务器地址 | localhost |
| Port | MQTT 服务器端口 | 1883 |
| UseTLS | 是否使用 TLS | false |
| SecureMode | 安全模式 (2/3) | 自动判断 |
| KeepAlive | 心跳间隔 | 60s |
| CleanSession | 清除会话 | true |

### TLS 配置

| 参数 | 说明 | 默认值 |
|------|------|-------|
| CACert | CA 证书内容 | 内置证书 |
| SkipVerify | 跳过证书验证 | false |

## 环境变量支持

SDK 支持通过环境变量进行配置：

```bash
export IOT_PRODUCT_KEY="your_product_key"
export IOT_DEVICE_NAME="your_device_name"
export IOT_DEVICE_SECRET="your_device_secret"
export IOT_MQTT_HOST="your_mqtt_host"
export IOT_MQTT_PORT="1883"
export IOT_MQTT_USE_TLS="false"
export IOT_MQTT_SECURE_MODE="3"
```

然后在代码中：

```go
cfg := config.NewConfig()
cfg.LoadFromEnv()
```

## 错误处理

SDK 提供完整的错误处理机制：

```go
if err := client.Connect(); err != nil {
    log.Printf("Connection failed: %v", err)
    // 处理连接失败
}

if err := client.Publish(topic, payload, 0, false); err != nil {
    log.Printf("Publish failed: %v", err)
    // 处理发布失败
}
```

## 日志配置

```go
import "log"

logger := log.New(os.Stdout, "[IoT-SDK] ", log.LstdFlags)
client.SetLogger(logger)
```

## 对比 C SDK

| 功能 | C SDK | Go SDK | 状态 |
|------|-------|--------|------|
| 三元组认证 | ✅ | ✅ | 完成 |
| MQTT 基础连接 | ✅ | ✅ | 完成 |
| MQTT TLS 连接 | ✅ | ✅ | 完成 |
| HTTP 动态注册 | ✅ | ✅ | 完成 |
| MQTT 动态注册 | ✅ | ✅ | 完成 |
| RRPC 功能 | ✅ | ✅ | 完成 |
| 二进制数据传输 | ✅ | ✅ | 完成 |
| 自动重连 | ✅ | ✅ | 完成 |
| 安全模式支持 | ✅ | ✅ | 完成 |
| 物模型支持 | ✅ | ✅ | 完成 |
| 事件驱动框架 | ❌ | ✅ | 完成 |
| 插件化架构 | ❌ | ✅ | 完成 |
| OTA 固件升级 | ✅ | ✅ | 完成 |

## 开发进展

### v1.0.0 - 完整功能实现

#### ✅ 核心功能
- **MQTT 连接**: 支持基础和 TLS 加密连接
- **三元组认证**: 完全兼容 C SDK 的认证算法
- **动态注册**: HTTP 和 MQTT 两种动态注册方式
- **RRPC 功能**: 远程过程调用，支持多种处理器
- **安全模式**: 自动适配 securemode=2/3

#### ✅ IoT Framework 实现

**事件驱动框架**
- **核心组件**: EventBus、PluginManager、DeviceManager
- **MQTT 插件**: 完整集成，支持物模型通信
- **服务路由**: 修复了服务调用路由机制（topic 解析问题）
- **OTA 插件**: 完整的固件升级功能，支持多模块和自动更新
- **事件上报**: 支持自定义事件类型上报
- **动态频率**: 根据设备状态智能调整上报频率

#### ✅ 重要发现和修复

**1. HTTP 动态注册签名算法**
- **问题**: 初始实现使用了错误的签名格式（参数排序+URL编码）
- **修复**: 改为 C SDK 兼容格式：`deviceName%sproductKey%srandom%s`
- **影响**: HTTP 动态注册现在完全正常工作

**2. MQTT 认证 ClientID 格式**
- **问题**: 原始 ClientID 格式不完整，缺少关键参数
- **修复**: 采用 C SDK 完整格式：`ProductKey.DeviceName|timestamp=xxx,_ss=1,_v=xxx,securemode=x,signmethod=hmacsha256,ext=3,|`
- **影响**: 认证兼容性大幅提升

**3. 安全模式自动判断**
- **新增**: 根据 TLS 使用情况自动设置 securemode
- **规则**: TLS 连接使用 securemode=2，非 TLS 使用 securemode=3
- **兼容**: 支持手动指定覆盖自动判断

**4. 服务器连接配置**
- **发现**: 不同服务器支持不同的端口和协议组合
- **规律**: TLS 和非 TLS 连接可能需要使用不同的服务器地址
- **建议**: 生产环境中需要根据实际部署情况配置服务器地址

#### 🧪 测试验证

**连接测试**
- ✅ 非 TLS 连接（securemode=3，端口 1883）
- ✅ TLS 连接（securemode=2，端口 8883）
- ✅ 消息发布和订阅
- ✅ 自动重连机制

**动态注册测试**
- ✅ HTTP 动态注册（获取 DeviceSecret）
- ✅ MQTT 动态注册（支持免白名单模式）
- ✅ 签名算法验证

**RRPC 测试**
- ✅ 请求/响应机制
- ✅ 多处理器支持
- ✅ 错误处理

**OTA 测试**
- ✅ 固件版本查询
- ✅ 固件下载和验证
- ✅ 多模块支持（x86、arm64等）
- ✅ 进度上报机制
- ✅ 版本文件持久化

#### 📈 性能和稳定性
- **内存管理**: 无内存泄漏
- **并发安全**: 所有操作线程安全
- **错误处理**: 完整的错误处理机制
- **日志系统**: 可配置的分级日志
- **框架集成**: 事件驱动框架与 SDK 无缝集成
- **OTA 功能**: 完整的固件升级功能，支持多模块架构

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！

## 技术支持

如有问题，请通过以下方式联系：

1. 提交 GitHub Issue
2. 查看示例代码
3. 阅读 API 文档