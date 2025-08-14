# 电烤炉OTA迁移说明

## 📋 迁移概述

电烤炉示例已从使用自己的OTA实现迁移到使用框架层的OTA插件。

## 🔄 主要变更

### 之前（应用层OTA）
- ✅ 在`main.go`中创建`OTAManager`实例
- ✅ 使用`ota.go`文件中的自定义OTA实现
- ✅ 手动管理MQTT客户端和OTA生命周期
- ✅ 版本信息存储在纯文本`version.txt`

### 现在（框架层OTA）
- ✅ 加载框架的OTA插件（`pkg/framework/plugins/ota`）
- ✅ OTA功能由框架自动管理
- ✅ 支持JSON格式的`version.txt`，包含版本和模块信息
- ✅ 设备通过属性`firmware_version`和`firmware_module`暴露版本信息

## 📁 文件变更

### 1. `main.go`
```go
// 添加导入
import "github.com/iot-go-sdk/pkg/framework/plugins/ota"

// 加载OTA插件
otaPlugin := ota.NewOTAPlugin()
otaPlugin.SetCheckInterval(5 * time.Minute)
framework.LoadPlugin(otaPlugin)

// 移除旧的OTA管理器代码
// otaManager := NewOTAManager(...) - 已删除
```

### 2. `electric_oven.go`
```go
// 新增属性
firmwareModule string // 固件模块 (x86, arm64等)

// 注册新属性
o.framework.RegisterProperty("firmware_module", o.getFirmwareModule, o.setFirmwareModule)

// 新增getter/setter
func (o *ElectricOven) getFirmwareModule() interface{} { ... }
func (o *ElectricOven) setFirmwareModule(value interface{}) error { ... }
```

### 3. `version.txt`
```json
// 从纯文本
1.0.12

// 升级为JSON
{
  "version": "1.0.12",
  "module": "x86"
}
```

## 🎯 优势

### 使用框架OTA插件的好处

1. **代码复用**
   - 不需要维护独立的OTA实现
   - 所有设备共享同一个OTA系统

2. **功能完整**
   - 自动版本检测
   - 多设备管理
   - 统一状态上报
   - 完整的错误处理

3. **配置灵活**
   - 可配置检查间隔
   - 可开关自动更新
   - 支持多种下载策略

4. **维护简单**
   - OTA逻辑集中在框架层
   - 易于测试和调试
   - 统一的更新流程

## 🔧 配置说明

### OTA插件配置
```go
// 设置检查间隔
otaPlugin.SetCheckInterval(5 * time.Minute)

// 启用/禁用自动更新
otaPlugin.SetAutoUpdate(true)

// 通过Configure方法配置
otaPlugin.Configure(map[string]interface{}{
    "auto_update": true,
    "check_interval": 10 * time.Minute,
})
```

### 版本文件格式
```json
{
  "version": "1.0.12",  // 固件版本
  "module": "x86"       // 模块名称
}
```

支持的模块名：
- `x86` - x86架构
- `arm64` - ARM64架构
- `mips` - MIPS架构
- `default` - 默认模块

## 📊 对比表

| 功能 | 应用层OTA | 框架层OTA |
|------|-----------|-----------|
| 代码位置 | `examples/framework/simple/ota.go` | `pkg/framework/plugins/ota/` |
| MQTT客户端 | 手动管理 | 自动获取 |
| 版本管理 | 纯文本 | JSON格式 |
| 模块支持 | ❌ | ✅ |
| 多设备支持 | ❌ | ✅ |
| 自动更新 | ✅ | ✅ |
| 下载策略 | 简单下载 | 简单/分块下载 |
| 错误恢复 | 基础 | 完整 |
| 状态上报 | 手动 | 自动 |
| 测试覆盖 | 无 | 有单元测试 |

## 🚀 迁移步骤

如果要将其他设备从应用层OTA迁移到框架层OTA：

1. **加载OTA插件**
   ```go
   otaPlugin := ota.NewOTAPlugin()
   framework.LoadPlugin(otaPlugin)
   ```

2. **注册版本属性**
   ```go
   device.RegisterProperty("firmware_version", ...)
   device.RegisterProperty("firmware_module", ...)
   ```

3. **更新version.txt格式**
   ```json
   {
     "version": "x.x.x",
     "module": "your_module"
   }
   ```

4. **移除旧的OTA代码**
   - 删除自定义OTA管理器
   - 删除手动版本上报代码

## ⚠️ 注意事项

1. **向后兼容**：框架OTA插件支持读取旧的纯文本version.txt
2. **自动转换**：首次更新时会自动将纯文本格式转换为JSON
3. **默认值**：如果没有指定模块，默认使用"default"
4. **依赖关系**：OTA插件依赖MQTT插件，必须先加载MQTT插件

## 🐛 已知问题与解决方案

### 问题1: Ctrl+C无法终止程序
**现象**: 集成OTA功能后，使用Ctrl+C无法正常终止程序，程序出现挂起

**根本原因**: OTA插件的同步事件处理器在框架启动期间造成死锁
- 设备注册事件触发OTA初始化
- OTA初始化过程阻塞了框架的启动流程
- 信号处理器无法正常工作

**解决方案**:
```go
// 将事件处理器改为异步处理
p.framework.On("device.registered", func(evt *event.Event) error {
    go func() {
        // 异步处理设备注册，避免阻塞框架
        // ... OTA设备注册逻辑
    }()
    return nil
})
```

### 问题2: OTA版本检查不执行
**现象**: "这个 oven 启动的时候，会用 ota 的接口检查是否有新的版本，但是我没看到日志有输出他在做检查啊"

**根本原因**: 设备注册时机问题
- 设备在框架启动**之前**注册
- OTA插件的事件处理器在框架启动**之后**注册
- `device.registered`事件在OTA插件准备好之前就已经发出

**解决方案**:
```go
// main.go中调整注册顺序
// 1. 先启动框架
if err := framework.Start(); err != nil {
    log.Fatalf("Failed to start framework: %v", err)
}

// 2. 再注册设备（确保OTA插件事件处理器已准备就绪）
if err := framework.RegisterDevice(oven); err != nil {
    log.Fatalf("Failed to register device: %v", err)
}
```

### 问题3: MQTT客户端获取死锁
**现象**: OTA管理器创建时挂起，程序无响应

**根本原因**: 互斥锁嵌套死锁
```go
// 错误的实现 - 造成死锁
func (p *OTAPlugin) createManagerForDevice(dev core.Device) error {
    p.mu.Lock()          // 第一次加锁
    defer p.mu.Unlock()
    
    // ... 其他代码
    
    mqttClient := p.getMQTTClient()  // 调用getMQTTClient
}

func (p *OTAPlugin) getMQTTClient() *mqtt.Client {
    p.mu.Lock()          // 第二次加锁 - 死锁！
    defer p.mu.Unlock()
    // ...
}
```

**解决方案**: 重构锁结构，避免嵌套加锁
```go
func (p *OTAPlugin) createManagerForDevice(dev core.Device) error {
    // 先获取MQTT客户端（无锁状态）
    mqttClient := p.getMQTTClient()
    if mqttClient == nil {
        return fmt.Errorf("MQTT client not available")
    }
    
    // 再加锁进行管理器操作
    p.mu.Lock()
    defer p.mu.Unlock()
    
    // ... 创建管理器的其他逻辑
}
```

### 问题4: 框架插件系统访问死锁
**现象**: `p.framework.GetPlugin("mqtt")`调用挂起

**根本原因**: 在设备初始化期间访问框架插件系统可能造成循环依赖

**解决方案**: 直接设置MQTT客户端，避开框架插件系统
```go
// main.go中直接设置MQTT客户端
mqttPlugin := mqtt.NewMQTTPlugin(sdkConfig)
framework.LoadPlugin(mqttPlugin)

otaPlugin := ota.NewOTAPlugin()
framework.LoadPlugin(otaPlugin)

// 框架启动后直接设置MQTT客户端
framework.Start()
otaPlugin.SetMQTTClient(mqttPlugin.GetMQTTClient())
```

### 问题5: 固件查询缺少模块参数
**现象**: 固件查询时params为空，平台返回空数据
```
Published message to topic: /sys/.../thing/ota/firmware/get
{"id":"...","version":"1.0","params":{}}  // params为空
Received OTA message: {"code":200,"data":{}}  // 无固件更新
```

**根本原因**: `QueryFirmware()`方法没有携带模块参数，平台无法识别设备类型

**解决方案**: 修改OTA查询流程，添加模块参数支持
```go
// pkg/ota/ota.go - 添加带模块参数的查询方法
func (c *Client) QueryFirmwareWithModule(module string) error {
    params := map[string]interface{}{}
    if module != "" {
        params["module"] = module
    }
    
    payload := map[string]interface{}{
        "id":      fmt.Sprintf("%d", time.Now().UnixNano()),
        "version": "1.0",
        "params":  params,  // 包含模块参数
    }
    // ...
}

// pkg/framework/plugins/ota/manager.go - 在查询时传递模块参数
func (m *ManagerImpl) CheckUpdate() (*UpdateInfo, error) {
    // 获取模块名称
    module := "default"
    if m.versionProvider != nil {
        module = m.versionProvider.GetModule()
    }
    
    // 使用带模块参数的查询方法
    if err := m.otaClient.QueryFirmwareWithModule(module); err != nil {
        return nil, err
    }
    // ...
}
```

**修复效果**:
```
Published message to topic: /sys/.../thing/ota/firmware/get
{"id":"...","version":"1.0","params":{"module":"arm"}}  // 包含模块参数
Received OTA message: {"code":200,"data":{"version":"1.0.13","module":"arm",...}}  // 返回对应固件
```

### 问题6: 进度上报格式不正确
**现象**: 进度上报缺少模块信息，格式不规范

**根本原因**: `ReportProgress`调用时缺少模块参数，进度信息不完整

**解决方案**: 修复进度上报格式，包含完整模块信息
```go
// pkg/framework/plugins/ota/manager.go - 修复进度上报
func (m *ManagerImpl) notifyStatus(status Status, progress int32, message string) {
    // 获取模块名称
    module := "default"
    if m.versionProvider != nil {
        module = m.versionProvider.GetModule()
    }
    
    // 上报进度时包含模块信息
    if status == StatusDownloading || status == StatusVerifying || status == StatusUpdating {
        m.otaClient.ReportProgress("download", message, int(progress), module)
        m.logger.Printf("Reported progress: %d%% (%s) - %s", progress, module, message)
    }
}
```

**修复效果**:
```
[OTA-S4Wj7RZ5TO] Reported progress: 0% (arm) - Starting download
[OTA-S4Wj7RZ5TO] Reported progress: 1% (arm) - Downloading: 101919/9818066 bytes
[OTA-S4Wj7RZ5TO] Reported progress: 25% (arm) - Downloading: 2454516/9818066 bytes
```

### 问题7: 版本文件更新缺失
**现象**: OTA升级成功后，version.txt文件未更新，重启后仍报告旧版本

**根本原因**: 只更新了设备属性，没有持久化到version.txt文件

**解决方案**: 添加版本文件更新机制
```go
// pkg/framework/plugins/ota/manager.go - 添加版本文件更新
func (m *ManagerImpl) updateVersionFile(newVersion string) error {
    // 获取当前模块名
    module := "default"
    if m.versionProvider != nil {
        module = m.versionProvider.GetModule()
    }
    
    // 创建版本信息结构
    versionInfo := VersionInfo{
        Version: newVersion,
        Module:  module,
    }
    
    // 尝试多个可能的位置写入version.txt
    possiblePaths := []string{
        "version.txt",      // 当前工作目录
        "./version.txt",    // 显式当前目录
    }
    
    // 添加可执行文件目录路径
    if execPath, err := os.Executable(); err == nil {
        execPath, _ = filepath.EvalSymlinks(execPath)
        dir := filepath.Dir(execPath)
        possiblePaths = append(possiblePaths, filepath.Join(dir, "version.txt"))
    }
    
    // 保存为JSON格式
    data, err := json.MarshalIndent(versionInfo, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal version info: %v", err)
    }
    
    // 写入文件
    for _, path := range possiblePaths {
        if err := os.WriteFile(path, data, 0644); err == nil {
            m.logger.Printf("Updated version file at %s with version %s (module: %s)", 
                path, newVersion, module)
            return nil
        }
    }
    
    return fmt.Errorf("failed to write version file to any location")
}

// 在OTA升级成功后调用
func (m *ManagerImpl) PerformUpdate(info *UpdateInfo) (*UpdateResult, error) {
    // ... 下载和验证逻辑
    
    // 更新设备属性
    if err := m.versionProvider.SetVersion(info.Version); err != nil {
        m.logger.Printf("Failed to save version to device: %v", err)
    }
    
    // 更新版本文件
    if err := m.updateVersionFile(info.Version); err != nil {
        m.logger.Printf("Failed to update version.txt file: %v", err)
    } else {
        m.logger.Printf("Successfully updated version.txt to version %s", info.Version)
    }
    
    // ...
}
```

**修复效果**:
```
[OTA-S4Wj7RZ5TO] Successfully updated version.txt to version 1.0.13
// 重启后
[S4Wj7RZ5TO] Loaded version info from version.txt: version=1.0.13, module=arm
[OTA-S4Wj7RZ5TO] Starting OTA manager, current version: 1.0.13
```

## 🔧 调试技巧

### 1. 检查后台进程冲突
```bash
# 查看占用IoT端口的进程
lsof -i -P | grep -E "(8883|1883|121\.41\.43\.80|121\.40\.253\.224)" | grep -v LISTEN

# 清理冲突进程
lsof -i -P | grep -E "(1883|121\.40\.253\.224)" | grep -v LISTEN | awk '{print $2}' | xargs -r kill -9 2>/dev/null || true
```

### 2. 增加调试日志
```go
// 在关键位置添加调试日志
p.logger.Printf("Creating OTA manager for device %s", deviceID)
p.logger.Printf("Getting MQTT client for device %s...", deviceID)
p.logger.Printf("Successfully created OTA manager for device %s", deviceID)
```

### 3. 超时保护
```go
// 为可能挂起的操作添加超时
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
```

## 📈 性能优化

### 1. 延迟初始化
```go
// 添加适当延迟，避免竞争条件
time.Sleep(2 * time.Second)  // 让框架完全初始化
```

### 2. 重试机制
```go
// 对可能失败的操作增加重试
maxRetries := 3
for i := 0; i < maxRetries; i++ {
    if err := p.RegisterDevice(dev); err != nil {
        if i < maxRetries-1 {
            time.Sleep(1 * time.Second)
            continue
        }
        return err
    }
    break
}
```

## 📝 总结

迁移到框架层OTA插件后，电烤炉示例的代码更简洁，功能更完整，维护更方便。这是推荐的OTA实现方式。

通过解决上述已知问题，现在系统可以：
- ✅ **正常启动和停止**（Ctrl+C响应）
- ✅ **自动进行OTA版本检查**（带模块参数）
- ✅ **稳定的MQTT客户端管理**（避免死锁）
- ✅ **正确的进度上报**（包含百分比和模块信息）
- ✅ **版本文件持久化**（自动更新version.txt）
- ✅ **完整的OTA生命周期支持**（从查询到升级完成）

## 🚀 完整的OTA工作流程

修复后的OTA系统完整工作流程：

### 1. 启动阶段
```
[S4Wj7RZ5TO] Loaded version info from version.txt: version=1.0.12, module=arm
[OTA-S4Wj7RZ5TO] Starting OTA manager, current version: 1.0.12
[OTA-S4Wj7RZ5TO] Reporting version to platform: 1.0.12 (module: arm)
```

### 2. 查询阶段
```
[OTA-S4Wj7RZ5TO] Checking for updates...
Published message to topic: /sys/QLTMkOfW/S4Wj7RZ5TO/thing/ota/firmware/get
Payload: {"id":"...","version":"1.0","params":{"module":"arm"}}
Queried for firmware updates (module: arm)
```

### 3. 响应阶段
```
Received OTA message: {"code":200,"data":{"version":"1.0.13","module":"arm","size":9818066,...}}
[OTA-S4Wj7RZ5TO] === OTA Update Available ===
[OTA-S4Wj7RZ5TO] Current version: 1.0.12
[OTA-S4Wj7RZ5TO] New version: 1.0.13
[OTA-S4Wj7RZ5TO] Size: 9818066 bytes
```

### 4. 下载阶段
```
[OTA-S4Wj7RZ5TO] Reported progress: 0% (arm) - Starting download
[OTA-S4Wj7RZ5TO] Reported progress: 1% (arm) - Downloading: 101919/9818066 bytes
[OTA-S4Wj7RZ5TO] Reported progress: 25% (arm) - Downloading: 2454516/9818066 bytes
[OTA-S4Wj7RZ5TO] Reported progress: 50% (arm) - Verifying firmware
[OTA-S4Wj7RZ5TO] Reported progress: 75% (arm) - Preparing update
[OTA-S4Wj7RZ5TO] Reported progress: 100% (arm) - Update prepared
```

### 5. 升级阶段
```
[OTA-S4Wj7RZ5TO] Successfully updated version.txt to version 1.0.13
[OTA-S4Wj7RZ5TO] Reported progress: 100% (arm) - Restarting with new version
[OTA-S4Wj7RZ5TO] Update completed successfully
```

### 6. 重启验证
```
// 设备重启后
[S4Wj7RZ5TO] Loaded version info from version.txt: version=1.0.13, module=arm
[OTA-S4Wj7RZ5TO] Starting OTA manager, current version: 1.0.13
[OTA-S4Wj7RZ5TO] Reporting version to platform: 1.0.13 (module: arm)
```

现在OTA系统完全按照IoT标准流程工作，支持多模块、进度上报、版本持久化等完整功能！