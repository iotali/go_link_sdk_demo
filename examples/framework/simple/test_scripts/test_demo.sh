#!/bin/bash

# 简单的测试演示脚本
# 演示电烤炉的主要功能

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

# 确保脚本有执行权限
chmod +x *.sh

print_header "电烤炉功能测试演示"

# 测试1: 设置温度
print_step "测试1: 设置温度到180°C"
./set_temperature.sh 180
wait_seconds 3

# 测试2: 启动定时器（会触发2秒快速上报）
print_step "测试2: 启动5分钟定时器"
echo -e "${BLUE}注意: 定时器启动后应该切换到2秒上报模式${NC}"
./invoke_timer.sh 5
wait_seconds 5

# 测试3: 切换门状态（会停止定时器）
print_step "测试3: 打开门（应该停止定时器）"
echo -e "${BLUE}注意: 门打开应该停止定时器并触发timer_cancelled事件${NC}"
./toggle_door.sh
wait_seconds 3

# 测试4: 尝试在门开时设置温度（应该失败）
print_step "测试4: 尝试在门开时设置温度200°C（应该失败）"
./set_temperature.sh 200
wait_seconds 3

# 测试5: 关闭门
print_step "测试5: 关闭门"
./toggle_door.sh
wait_seconds 3

# 测试6: 现在可以设置温度了
print_step "测试6: 门关闭后设置温度200°C（应该成功）"
./set_temperature.sh 200
wait_seconds 3

# 测试7: 停止加热
print_step "测试7: 停止加热（设置温度为0）"
./set_temperature.sh 0
wait_seconds 2

print_header "测试完成！"
echo -e "${GREEN}请查看电烤炉程序的日志输出以验证以下行为：${NC}"
echo "1. 定时器启动后切换到2秒上报模式"
echo "2. 门打开时停止定时器"
echo "3. 门开时拒绝设置温度"
echo "4. 温度根据不同状态显示不同操作模式（预热中、加热中、保温中等）"