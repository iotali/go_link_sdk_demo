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

## 📝 总结

迁移到框架层OTA插件后，电烤炉示例的代码更简洁，功能更完整，维护更方便。这是推荐的OTA实现方式。