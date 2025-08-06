# OTA (Over-The-Air) 升级技术文档

## 概述

OTA升级是IoT设备远程固件更新的核心功能。本文档总结了在实现Go SDK的OTA功能时遇到的技术细节、与IoT平台的交互方式以及需要注意的陷阱。

## 与IoT平台的交互流程

### 1. 完整的OTA升级流程

```
设备                                    IoT平台
 |                                        |
 |---(1) 上报当前版本------------------->|
 |   /ota/device/inform/{pk}/{dn}        |
 |   {"params":{"version":"1.0.0",       |
 |    "module":"default"}}               |
 |                                        |
 |---(2) 查询固件更新------------------->|
 |   /sys/{pk}/{dn}/thing/ota/firmware/get|
 |                                        |
 |<--(3) 接收固件信息--------------------|
 |   /sys/{pk}/{dn}/thing/ota/firmware/  |
 |   get_reply                            |
 |   {"data":{"version":"1.0.2",...}}    |
 |                                        |
 |---(4) 下载固件 (HTTPS)--------------->|
 |   从URL下载固件文件                    |
 |                                        |
 |---(5) 上报下载进度------------------->|
 |   /ota/device/progress/{pk}/{dn}      |
 |   {"params":{"step":"50",...}}        |
 |                                        |
 |---(6) 设备重启 & 使用新固件----------->|
 |                                        |
 |---(7) 设备上线 (MQTT连接)------------>|
 |                                        |
 |---(8) 上报新版本 (2秒内!)------------>|
 |   /ota/device/inform/{pk}/{dn}        |
 |   {"params":{"version":"1.0.2",...}}  |
 |                                        |
```

### ⚠️ 关键时序要求

**重要**：设备升级完成后的版本上报有严格的时序要求：

1. **立即重启**：固件烧写完成后立即重启设备
2. **快速上线**：重启后使用新固件启动，建立MQTT连接
3. **2秒规则**：设备上线后必须在2秒内上报新版本号

```
重启 -> MQTT连接 -> 上报版本 (≤2秒)
```

**失败条件**：如果超过2秒未上报新版本，平台可能判定升级失败，即使进度已上报100%。

### 2. MQTT主题订阅

必须订阅的主题：
- `/ota/device/upgrade/{productKey}/{deviceName}` - 接收主动推送的升级任务
- `/sys/{productKey}/{deviceName}/thing/ota/firmware/get_reply` - 接收查询响应

### 3. 消息格式

#### 版本上报
```json
{
  "id": "时间戳",
  "params": {
    "version": "1.0.0",
    "module": "default"  // 可选，但建议包含
  }
}
```

#### 进度上报
```json
{
  "id": "时间戳",
  "params": {
    "step": "100",      // 进度百分比或错误码
    "desc": "描述",
    "progress": 100,    // 进度值
    "module": "default" // 可选
  }
}
```

#### 固件查询响应
```json
{
  "code": 200,
  "data": {
    "version": "1.0.2",
    "size": 1552,
    "url": "https://...",
    "md5": "c7277e3c77151d3447c2ce47405612cc",
    "sign": "c7277e3c77151d3447c2ce47405612cc",
    "signMethod": "Md5",
    "module": "default"
  }
}
```

## 技术实现要点

### 1. Module字段处理

**重要**：module字段在整个OTA流程中需要保持一致。

```go
// 版本上报时包含module
if module != "" {
    params["module"] = module
}

// 进度上报时也要包含相同的module
if task.Module != "" {
    params["module"] = task.Module
}
```

### 2. 分段下载与MD5验证

**陷阱**：分段下载时不能对每个分段单独计算MD5。

错误做法：
```go
// ❌ 每个分段独立验证MD5
func Download(rangeStart, rangeEnd) {
    // 下载分段
    hasher := md5.New()
    hasher.Write(segmentData)
    if hasher.Sum() != expectedMD5 { // 这会失败！
        return error
    }
}
```

正确做法：
```go
// ✅ 累积所有数据后验证完整文件的MD5
var firmwareData []byte

// 下载所有分段
firmwareData = append(firmwareData, segment1...)
firmwareData = append(firmwareData, segment2...)

// 最后验证完整数据
hasher := md5.New()
hasher.Write(firmwareData)
actualMD5 := fmt.Sprintf("%x", hasher.Sum(nil))
if actualMD5 != task.ExpectDigest {
    return error
}
```

### 3. 空响应处理

当没有可用更新时，平台返回空data对象：
```json
{"code":200,"data":{},"id":"xxx"}
```

需要正确处理：
```go
if len(data) == 0 {
    log.Printf("No firmware update available")
    return nil
}
```

### 4. 错误码规范

遵循C SDK定义的标准错误码：
- `-1`: 升级失败
- `-2`: 下载失败  
- `-3`: MD5校验失败
- `-4`: 固件烧写失败

```go
// 报告错误
client.ReportProgress("-2", "Download failed", -2, module)
```

## 常见陷阱与解决方案

### ⚠️ 陷阱1：版本上报时序错误 (最严重)

**问题**：升级后未及时上报新版本，或上报时机不对。

**现象**：
- 进度显示100%但平台显示升级失败
- 超过升级超时时间后任务失败

**解决方案**：
```go
// ❌ 错误：在demo中立即上报（生产环境不要这样）
client.ReportVersionWithModule(newVersion, module)

// ✅ 正确：重启后立即上报
func main() {
    // 连接MQTT
    mqttClient.Connect()
    
    // 立即上报当前版本（2秒内！）
    otaClient.ReportVersion(getCurrentFirmwareVersion())
    
    // 然后才处理其他逻辑
}
```

**关键点**：
- 升级成功的**唯一判定标准**是版本号匹配
- 设备上线后2秒内必须上报版本
- 即使进度100%，不上报版本仍算失败

### 陷阱2：忽略订阅查询响应主题

**问题**：只订阅了升级推送主题，没有订阅查询响应主题。

**现象**：
```
Queried for firmware updates
DEFAULT HANDLER - Received message on topic /sys/.../thing/ota/firmware/get_reply
```

**解决**：必须订阅 `/sys/+/+/thing/ota/firmware/get_reply`

### 陷阱3：通配符订阅失败

**问题**：MQTT客户端不支持通配符匹配。

**现象**：订阅了 `/ota/device/upgrade/+/+` 但收不到消息。

**解决**：
1. 实现正确的通配符匹配函数
2. 或使用具体的productKey/deviceName订阅

```go
// 使用具体路径而非通配符
fotaTopic := fmt.Sprintf("/ota/device/upgrade/%s/%s", productKey, deviceName)
```

### 陷阱4：连接不稳定

**问题**：订阅过多主题或频繁操作导致连接断开。

**现象**：
```
Connection lost: EOF
Attempting to reconnect to MQTT broker...
```

**解决**：
1. 减少同时订阅的主题数量
2. 使用具体路径而非通配符
3. 实现自动重连机制

### 陷阱5：进度上报时机

**问题**：在下载完成handler中报告100%进度。

**建议**：
- 下载过程中：每5%报告一次进度
- 下载完成后：在主流程中报告100%
- 升级成功后：重启并上报新版本号

### 陷阱6：TLS端口配置

**重要**：不同端口需要使用不同的IP地址：
- Port 1883 (非TLS): 使用 `121.40.253.224`
- Port 8883 (TLS): 使用 `121.41.43.80`

## 完整示例代码结构

```go
// 1. 创建OTA客户端
otaClient := ota.NewClient(mqttClient, productKey, deviceName)

// 2. 设置接收处理器
otaClient.SetRecvHandler(func(client *ota.Client, recvType ota.RecvType, task *ota.TaskDesc) {
    // 处理FOTA任务
    if recvType == ota.RecvTypeFOTA {
        // 下载固件
        client.Download(ctx, task, 0, 0)
        // 报告进度
        client.ReportProgress("100", "Download completed", 100, task.Module)
        // 升级后报告新版本
        client.ReportVersionWithModule(task.Version, task.Module)
    }
})

// 3. 启动客户端
otaClient.Start()

// 4. 上报当前版本
otaClient.ReportVersion(currentVersion)

// 5. 查询更新
otaClient.QueryFirmware()
```

## 调试技巧

### 查找后台IoT连接

```bash
# 查看活跃的IoT连接
lsof -i -P | grep -E "(8883|1883|121\.41\.43\.80|121\.40\.253\.224)" | grep -v LISTEN

# 查找Go进程
ps aux | grep -E "(go run|/go-build/)" | grep -v grep

# 终止进程
pkill -f "go run main.go"
```

### 日志分析要点

1. 检查主题订阅是否成功
2. 确认消息是否被正确路由到handler
3. 验证module字段在整个流程中的一致性
4. 检查进度上报的时序

## 最佳实践

### 生产环境关键要求

1. **版本上报时序**：这是最重要的！
   ```go
   func main() {
       // 1. 建立MQTT连接
       mqttClient.Connect()
       
       // 2. 立即上报当前固件版本（2秒内完成！）
       currentVersion := readFirmwareVersion() // 从固件中读取版本
       otaClient.ReportVersionWithModule(currentVersion, "default")
       
       // 3. 然后处理其他业务逻辑
       // 4. 设置OTA处理器等
   }
   ```

2. **升级流程**：
   - 下载 → 验证 → 烧写 → **立即重启** → 上线 → **2秒内上报新版本**
   - 不要在下载完成后立即上报新版本（demo可以，生产不行）

3. **技术实现**：
   - **始终包含module字段**：即使是"default"，保持整个流程的一致性
   - **实现完整的错误处理**：使用标准错误码上报失败状态
   - **下载后验证**：完整下载后再验证MD5，而不是分段验证
   - **进度上报**：合理控制上报频率，避免过于频繁
   - **连接管理**：实现自动重连，处理网络不稳定情况

### 判定标准

**升级成功的唯一条件**：设备上报的版本号与OTA目标版本号一致

- ✅ 版本号匹配 = 升级成功
- ❌ 版本号不匹配 = 升级失败
- ❌ 不上报版本号 = 升级失败（即使进度100%）
- ❌ 超时未上报 = 升级失败

## 与C SDK的兼容性

Go SDK的OTA实现完全兼容C SDK的协议和行为：
- 相同的主题格式
- 相同的消息结构
- 相同的错误码定义
- 相同的module处理逻辑
- 相同的进度上报机制