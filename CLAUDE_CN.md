# CLAUDE.md (中文版)

此文件为 Claude Code (claude.ai/code) 在处理此代码库时提供指导。

## 项目概述

这是一个基于 Go 语言的中国物联网设备 SDK，旨在将物联网设备连接到物联网平台，**以 C SDK 兼容性为核心设计原则**。SDK 提供三元组认证、MQTT 连接、TLS 安全、动态注册和 RRPC 功能。

## 开发命令

本项目使用标准 Go 工具链：

```bash
# 构建整个 SDK
go build ./...

# 运行测试
go test ./...

# 安装/更新依赖
go mod download && go mod tidy

# 运行特定示例
cd examples/basic_mqtt && go run main.go
cd examples/tls_mqtt && go run main.go
cd examples/dynreg_http && go run main.go
cd examples/dynreg_mqtt && go run main.go
cd examples/rrpc && go run main.go
```

## 架构概述

SDK 采用分层架构，关注点分离清晰：

**配置层** (`pkg/config/`)：集中式配置，支持环境变量（`IOT_*` 前缀）。自动检测安全模式（TLS → securemode=2，非 TLS → securemode=3）。

**认证层** (`pkg/auth/`)：与 C SDK 兼容的凭证生成，使用 HMAC-SHA256。生成精确的 C SDK 格式 ClientID：`ProductKey.DeviceName|timestamp=xxx,_ss=1,_v=xxx,securemode=x,signmethod=hmacsha256,ext=3,|`

**MQTT 层** (`pkg/mqtt/`)：基于 Paho 的线程安全 MQTT 客户端，具有自动重连、TLS 支持和特定主题的消息路由。

**动态注册** (`pkg/dynreg/`)：支持基于 HTTP 和 MQTT 的设备注册，用于自动获取凭证。

**RRPC 层** (`pkg/rrpc/`)：远程过程调用实现，包含方法处理器、请求/响应关联和超时处理。

**TLS 层** (`pkg/tls/`)：证书管理，内置 CA 证书和自定义证书支持。

**OTA 层** (`pkg/ota/`)：固件升级功能，支持分段下载、进度上报和版本管理。

## 关键设计模式

**C SDK 兼容性**：所有认证算法、ClientID 格式和消息结构与 C SDK 实现完全匹配。使用固定时间戳（`2524608000000`）以保持一致性。

**线程安全**：在所有组件中广泛使用 `sync.RWMutex`，配合基于通道的异步通信。

**配置驱动**：单一配置结构体流经所有组件，具有全面的环境变量覆盖支持。

**回调架构**：基于函数的处理器（`MessageHandler`、`RequestHandler`）用于灵活的消息处理和 RPC 方法注册。

## 必需配置

设备凭证是必需的：
- `ProductKey` 和 `DeviceName`（始终必需）
- `DeviceSecret` 或 `ProductSecret`（用于动态注册）

支持的环境变量：
```
IOT_PRODUCT_KEY, IOT_DEVICE_NAME, IOT_DEVICE_SECRET
IOT_MQTT_HOST, IOT_MQTT_PORT, IOT_MQTT_USE_TLS
IOT_MQTT_SECURE_MODE, IOT_TLS_SKIP_VERIFY
```

## 依赖项

- Go 1.21+
- `github.com/eclipse/paho.mqtt.golang v1.4.3`（核心 MQTT 功能）

## TLS 配置

SDK 支持 TLS 连接，对生产环境有特殊处理：

**重要的网络配置**：
- 端口 1883（非 TLS）：使用 IP `121.40.253.224`
- 端口 8883（TLS）：使用 IP `121.41.43.80`（SSL 卸载配置）

**TLS 证书问题**：
1. 服务器使用**自签名证书**，颁发给 CN="IoT"，而不是 IP 地址
2. Go 的 TLS 验证比 C SDK 更严格
3. 通过 IP 连接到为域名颁发的证书时，证书验证将失败

**推荐的 TLS 配置**：

1. **用于测试/开发**（使用自签名证书）：
```go
cfg.TLS.SkipVerify = true  // 跳过证书验证
```

2. **用于生产**（使用正确的证书）：
```go
cfg.TLS.SkipVerify = false
cfg.TLS.ServerName = "IoT"  // 必须匹配证书 CN
```

**已知问题**：
- SDK 在 `pkg/mqtt/client.go` 中的 TLS 逻辑尝试通过在设置 `ServerName` 时设置 `InsecureSkipVerify=true` 来处理带域名证书的 IP 连接，但这可能无法正确处理自签名证书
- 对于自签名证书，始终使用 `SkipVerify = true`

## MQTT 主题通配符支持

**关键问题已修复**：MQTT 客户端现在支持通配符主题订阅（`+` 和 `#`）。没有这个功能，RRPC 和其他基于通配符的订阅将失败。

**实现**：客户端实现了 `topicMatches` 函数，处理 MQTT 通配符匹配，将消息路由到正确的处理器。

## 调试和进程管理

### 查找后台 IoT 连接

当关闭程序后 IoT 设备仍显示在线时，使用这些命令查找并终止后台进程：

**1. 通过 IoT 平台端口查找进程：**
```bash
# 检查 IoT 平台的活动连接（最有用）
lsof -i -P | grep -E "(8883|1883|121\.41\.43\.80|121\.40\.253\.224)" | grep -v LISTEN

# 按端口检查网络连接
netstat -tulpn | grep -E "(:8883|:1883)"

# 使用 ss 命令的替代方法
ss -tulpn | grep -E "(:8883|:1883)"
```

**2. 通过设备标识符查找 Go 进程：**
```bash
# 搜索包含设备凭证的进程
ps aux | grep -E "(QLTMkOfW|WjJjXbP0X1|THYYENG5wd)" | grep -v grep

# 查找所有 Go 相关进程
ps aux | grep "go" | grep -v grep | grep -v gopls
```

**3. 查找所有运行的 Go 程序：**
```bash
# 查找 go run 进程
ps aux | grep -E "(go run|main\.go)" | grep -v grep

# 查找缓存中编译的 Go 二进制文件
ps aux | grep "/go-build/" | grep -v grep
```

**4. 安全地终止进程：**
```bash
# 通过 PID 终止特定进程
kill -15 <PID>  # 首先尝试优雅终止
kill -9 <PID>   # 如果需要强制终止

# 终止所有 Go 进程（谨慎使用）
pkill -f "go run"
```

**5. 综合 IoT 连接检查：**
```bash
# 一行命令查找所有 IoT 相关连接
lsof -i -P | grep -E "(8883|1883)" | grep -v LISTEN && ps aux | grep -E "(go run|/go-build/)" | grep -v grep
```

### 常见问题

- 从编辑器（VSCode、Cursor）启动的程序可能在后台继续运行
- 编译到 ~`~/.cache/go-build/` 的 Go 程序从进程名称可能不明显
- 始终使用 `Ctrl+C` 正确退出程序，而不是关闭终端窗口

## 动态注册（MQTT）

**重要实现细节**：

MQTT 动态注册过程与常规 MQTT 连接不同，有特定要求：

### 认证格式
1. **ClientID**：`deviceName.productKey|random={random},authType={authType},securemode=2,signmethod=hmacsha256|`
   - 注意：deviceName 在前，然后是 productKey
   - authType："register" 用于白名单模式，"regnwl" 用于免白名单模式
   - random：使用随机数，而不是时间戳

2. **Username**：`deviceName&productKey`

3. **Password**：HMAC-SHA256 签名，使用**大写**十六进制编码
   - 签名内容：`deviceName{deviceName}productKey{productKey}random{random}`
   - 使用 ProductSecret 签名（不是 DeviceSecret）

### 消息流程
**关键**：动态注册不需要手动主题订阅！

1. 客户端使用特殊凭证连接到 MQTT broker
2. 服务器自动为客户端订阅 `/ext/register/{productKey}/{deviceName}`
3. 服务器在连接后立即推送注册结果
4. 响应格式：`{"deviceSecret":"xxx"}`（直接 JSON，无包装）

### 常见陷阱
- **错误的 auth type**：确保 skipPreRegist 标志与平台配置匹配
- **大小写敏感性**：密码必须是大写十六进制（C SDK 兼容性）
- **无需手动订阅**：服务器自动处理订阅
- **ProductSecret vs DeviceSecret**：动态注册认证使用 ProductSecret

### 成功示例日志
```
Dynamic registration connecting with ClientID: deviceName.productKey|random=xxx,authType=register,securemode=2,signmethod=hmacsha256|
Connected to MQTT broker for dynamic registration: ssl://121.41.43.80:8883
Received message on topic /ext/register: {"deviceSecret":"xxx"}
```

## 当前限制

- 没有自动化测试套件
- 没有配置 CI/CD 管道
- 文档主要为中文
- 没有自定义构建脚本或 Makefile
- 后台进程管理需要手动检查

## 重要指令提醒

做所要求的事情；不多不少。
除非绝对必要，否则不要创建文件。
始终优先编辑现有文件而不是创建新文件。
除非用户明确要求，否则不要主动创建文档文件（*.md）或 README 文件。