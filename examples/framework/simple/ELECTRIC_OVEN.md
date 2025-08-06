# 电烤炉物模型模拟器

基于IoT框架的智能电烤炉模拟器，完整实现了物模型定义的所有功能。

## 功能特性

### 属性上报（每30秒自动上报）
- **current_temperature** - 当前温度（0-300°C）
- **target_temperature** - 目标温度（可设置）
- **heater_status** - 加热器状态
- **timer_setting** - 定时器设置（分钟）
- **remaining_time** - 剩余时间（分钟）
- **door_status** - 门状态（开/关）
- **power_consumption** - 功耗（瓦）
- **operation_mode** - 操作模式（待机/加热中/暂停等）
- **internal_light** - 内部照明（可设置）
- **fan_status** - 风扇状态

### 服务调用
1. **set_temperature** - 设定目标温度
   ```json
   {
     "temperature": 180.0
   }
   ```

2. **start_timer** - 启动定时器
   ```json
   {
     "time": 30
   }
   ```

3. **toggle_door** - 切换门状态
   ```json
   {}
   ```

### 事件告警
- **overheat_alarm** - 温度超过250°C时自动触发
- **timer_complete** - 定时器结束时触发

## 模拟逻辑

### 温度控制
- 每2秒更新一次温度
- 加热速率随温度升高而降低（模拟真实物理）
- 门打开时加热器自动关闭，散热速度加倍
- 温度过高（>250°C）自动触发告警并安全停机

### 功耗模拟
- 加热时：2000W + 正弦波动
- 仅风扇：50W
- 仅照明：15W
- 待机：0W

### 定时器
- 支持0-1440分钟（24小时）
- 每分钟倒计时
- 计时结束自动停止加热

## 运行方式

```bash
# 杀掉可能存在的后台进程
lsof -i -P | grep -E "(1883|121\.40\.253\.224)" | grep -v LISTEN | awk '{print $2}' | xargs -r kill -9 2>/dev/null || true

# 运行模拟器
go run .

# 或使用脚本运行（自动清理进程）
./run.sh
```

## 测试场景

### 场景1：正常烘烤
1. 设定温度180°C
2. 观察加热器启动，温度逐渐上升
3. 达到目标温度±5°C时，加热器自动调节

### 场景2：定时烘烤
1. 设定温度200°C
2. 启动30分钟定时器
3. 观察剩余时间倒计时
4. 时间到自动停止

### 场景3：门开安全保护
1. 正在加热时切换门状态
2. 观察加热器自动关闭
3. 操作模式变为"暂停（门开）"

### 场景4：过热保护
1. 设定温度300°C
2. 当实际温度超过250°C时
3. 自动触发过热告警事件
4. 安全停机

## Topic格式

属性上报：
```
$SYS/{ProductKey}/{DeviceName}/property/post
```

事件上报：
```
$SYS/{ProductKey}/{DeviceName}/event/post
```

服务调用：
```
$SYS/{ProductKey}/{DeviceName}/service/{serviceName}/invoke
```

## 调试提示

如果连接不稳定或频繁断线，检查是否有后台进程占用相同ClientID：
```bash
# 查看所有IoT连接
lsof -i -P | grep -E "(8883|1883|121\.41\.43\.80|121\.40\.253\.224)" | grep -v LISTEN

# 查看Go进程
ps aux | grep -E "(go run|/go-build/)" | grep -v grep
```