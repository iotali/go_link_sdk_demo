#!/bin/bash

# RRPC 本地调试测试脚本
# 用于测试RRPC功能是否正常工作

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}RRPC 本地调试测试${NC}"
echo -e "${GREEN}========================================${NC}"

# 1. 检查设备是否运行
echo -e "\n${YELLOW}步骤1: 检查电烤炉程序是否运行${NC}"
if pgrep -f "electric_oven" > /dev/null || pgrep -f "examples/framework/simple" > /dev/null; then
    echo -e "${GREEN}✓ 电烤炉程序正在运行${NC}"
else
    echo -e "${RED}✗ 电烤炉程序未运行${NC}"
    echo "请先运行: cd examples/framework/simple && ./run.sh"
    exit 1
fi

# 2. 检查MQTT连接
echo -e "\n${YELLOW}步骤2: 检查MQTT连接${NC}"
if lsof -i :1883 | grep -q ESTABLISHED || lsof -i :8883 | grep -q ESTABLISHED; then
    echo -e "${GREEN}✓ MQTT连接已建立${NC}"
else
    echo -e "${YELLOW}⚠ 未检测到MQTT连接${NC}"
fi

# 3. 显示日志监控提示
echo -e "\n${YELLOW}步骤3: 日志监控${NC}"
echo "请在另一个终端运行以下命令监控日志："
echo -e "${BLUE}tail -f oven.log | grep -E '(RRPC|request|response)'${NC}"

# 4. 模拟RRPC调用说明
echo -e "\n${YELLOW}步骤4: RRPC功能说明${NC}"
echo "已注册的RRPC处理器："
echo "  1. GetOvenStatus     - 获取烤炉状态"
echo "  2. SetOvenTemperature - 设置温度"
echo "  3. EmergencyStop     - 紧急停止"
echo "  4. InvokeService     - 调用框架服务"
echo "  5. GetDeviceStatus   - 获取设备状态"

# 5. 测试MQTT主题订阅
echo -e "\n${YELLOW}步骤5: 验证RRPC主题订阅${NC}"
echo "RRPC请求主题: /sys/QLTMkOfW/S4Wj7RZ5TO/rrpc/request/+"
echo "RRPC响应主题: /sys/QLTMkOfW/S4Wj7RZ5TO/rrpc/response/{messageId}"

# 6. 创建测试消息
echo -e "\n${YELLOW}步骤6: 测试消息格式${NC}"
echo "GetOvenStatus请求示例："
cat << 'EOF'
{
  "id": "1234567890",
  "version": "1.0",
  "method": "GetOvenStatus",
  "params": {}
}
EOF

echo -e "\n${YELLOW}SetOvenTemperature请求示例:${NC}"
cat << 'EOF'
{
  "id": "1234567891",
  "version": "1.0",
  "method": "SetOvenTemperature",
  "params": {
    "temperature": 180
  }
}
EOF

# 7. 提供手动测试方法
echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}手动测试方法${NC}"
echo -e "${GREEN}========================================${NC}"

echo -e "\n${YELLOW}使用mosquitto_pub发送RRPC请求:${NC}"
echo 'mosquitto_pub -h 121.40.253.224 -p 1883 -t "/sys/QLTMkOfW/S4Wj7RZ5TO/rrpc/request/test123" \'
echo '  -m "{\"id\":\"test123\",\"version\":\"1.0\",\"method\":\"GetOvenStatus\",\"params\":{}}"'

echo -e "\n${YELLOW}使用mosquitto_sub监听RRPC响应:${NC}"
echo 'mosquitto_sub -h 121.40.253.224 -p 1883 -t "/sys/QLTMkOfW/S4Wj7RZ5TO/rrpc/response/+"'

echo -e "\n${BLUE}提示:${NC}"
echo "1. 确保设备程序已启动并连接到MQTT"
echo "2. 查看程序日志确认RRPC客户端已启动"
echo "3. 日志应显示: '[MQTT Plugin] RRPC client started successfully'"
echo "4. 收到RRPC请求时会有日志: 'RRPC: [Method] request (ID: xxx)'"

echo -e "\n${GREEN}调试完成${NC}"