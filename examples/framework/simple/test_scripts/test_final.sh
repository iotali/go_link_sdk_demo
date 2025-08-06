#!/bin/bash

# 最终测试脚本 - 使用正确的API格式
# API格式说明：服务参数放在pointList中，服务标识符在servicePoint中

SERVER="https://deviot.know-act.com"
TOKEN="488820fb-41af-40e5-b2d3-d45a8c576eea"
DEVICE_NAME="S4Wj7RZ5TO"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

function print_header() {
    echo -e "\n${GREEN}========================================${NC}"
    echo -e "${GREEN}$1${NC}"
    echo -e "${GREEN}========================================${NC}\n"
}

function print_step() {
    echo -e "${YELLOW}>>> $1${NC}"
}

function wait_seconds() {
    echo -e "${BLUE}等待 $1 秒...${NC}"
    sleep $1
}

print_header "电烤炉API测试 - 正确格式"

# 测试1: 启动定时器（参数在pointList中）
print_step "测试1: 启动3分钟定时器"
echo "请查看日志是否显示: 'params:{time:3}'"
./invoke_timer.sh 3
wait_seconds 5

# 测试2: 检查上报频率
print_step "测试2: 检查是否切换到2秒上报"
echo "日志应该显示: 'Switching to fast reporting mode (2s)'"
wait_seconds 6

# 测试3: 切换门状态
print_step "测试3: 切换门状态（打开）"
echo "应该停止定时器并显示: 'Timer cancelled'"
./toggle_door_correct.sh
wait_seconds 3

# 测试4: 再次切换门状态
print_step "测试4: 切换门状态（关闭）"
./toggle_door_correct.sh
wait_seconds 3

# 测试5: 设置温度
print_step "测试5: 设置温度到200°C"
./set_temperature.sh 200
wait_seconds 3

# 测试6: 停止加热
print_step "测试6: 停止加热"
./set_temperature.sh 0
wait_seconds 2

print_header "测试完成"
echo -e "${GREEN}API格式总结：${NC}"
echo "1. 服务参数放在 pointList 数组中"
echo "2. 每个参数是一个对象，包含 identifier 和 value"
echo "3. 服务标识符放在 servicePoint.identifier 中"
echo ""
echo "正确格式示例："
echo '{'
echo '  "deviceName": "设备名",'
echo '  "pointList": ['
echo '    {'
echo '      "identifier": "参数名",'
echo '      "value": "参数值"'
echo '    }'
echo '  ],'
echo '  "servicePoint": {'
echo '    "identifier": "服务名"'
echo '  }'
echo '}'