#!/bin/bash

# 运行电烤炉程序并执行测试的综合脚本

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}电烤炉模拟器运行和测试脚本${NC}"
echo -e "${GREEN}========================================${NC}"

# 1. 清理可能存在的连接
echo -e "\n${YELLOW}步骤1: 清理现有连接...${NC}"
lsof -i -P | grep -E "(1883|121\.40\.253\.224)" | grep -v LISTEN | awk '{print $2}' | xargs -r kill -9 2>/dev/null || true
sleep 1

# 2. 编译程序
echo -e "\n${YELLOW}步骤2: 编译电烤炉程序...${NC}"
go build -o electric_oven_demo .
if [ $? -ne 0 ]; then
    echo -e "${RED}编译失败！${NC}"
    exit 1
fi
echo -e "${GREEN}编译成功！${NC}"

# 3. 启动电烤炉程序
echo -e "\n${YELLOW}步骤3: 启动电烤炉程序...${NC}"
./electric_oven_demo > oven.log 2>&1 &
OVEN_PID=$!
echo -e "${GREEN}电烤炉程序已启动 (PID: $OVEN_PID)${NC}"

# 4. 等待程序初始化
echo -e "\n${YELLOW}步骤4: 等待程序连接到IoT平台...${NC}"
sleep 5

# 检查程序是否运行
if ! ps -p $OVEN_PID > /dev/null; then
    echo -e "${RED}程序启动失败！查看日志：${NC}"
    tail -20 oven.log
    exit 1
fi

# 显示初始日志
echo -e "\n${BLUE}初始日志：${NC}"
tail -20 oven.log

# 5. 询问是否运行测试
echo -e "\n${YELLOW}程序已启动并连接。${NC}"
echo -e "${YELLOW}是否运行测试脚本？(y/n)${NC}"
read -n 1 -r
echo

if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo -e "\n${GREEN}开始运行测试...${NC}"
    cd test_scripts
    ./test_demo.sh
    cd ..
fi

# 6. 监控日志
echo -e "\n${YELLOW}持续监控日志（按Ctrl+C退出）...${NC}"
echo -e "${BLUE}提示: 查看日志中的以下关键信息：${NC}"
echo "  - 'Switching to fast reporting mode' - 切换到2秒上报"
echo "  - 'Switching to normal reporting mode' - 恢复30秒上报"
echo "  - 'Timer cancelled' - 定时器被取消"
echo "  - 'cannot set temperature when door is open' - 门开时拒绝设置温度"
echo ""

# 捕获Ctrl+C信号
trap 'echo -e "\n${YELLOW}停止电烤炉程序...${NC}"; kill $OVEN_PID 2>/dev/null; exit' INT

# 持续显示日志
tail -f oven.log