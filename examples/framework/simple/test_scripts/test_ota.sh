#!/bin/bash

# OTA测试脚本 - 触发电烤炉模拟器的OTA更新
# 这个脚本会通过IoT平台API触发OTA更新任务

# 配置
PRODUCT_KEY="QLTMkOfW"
DEVICE_NAME="S4Wj7RZ5TO"
SERVER_URL="https://deviot.know-act.com"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}     电烤炉OTA更新测试脚本${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 生成测试版本号
VERSION="1.0.$(date +%s | tail -c 3)"
echo -e "${YELLOW}新版本号: ${VERSION}${NC}"

# 创建一个简单的测试固件文件
echo "Creating test firmware..."
FIRMWARE_FILE="/tmp/test_firmware_${VERSION}.bin"

# 创建一个包含版本信息的测试固件
cat > "$FIRMWARE_FILE" << EOF
#!/bin/bash
echo "Test Firmware Version ${VERSION}"
echo "This is a test firmware file for OTA update testing"
echo "Generated at: $(date)"
# Add some random data to make file size reasonable
head -c 10240 /dev/urandom | base64
EOF

# 计算MD5
FIRMWARE_MD5=$(md5sum "$FIRMWARE_FILE" | cut -d' ' -f1)
FIRMWARE_SIZE=$(stat -f%z "$FIRMWARE_FILE" 2>/dev/null || stat -c%s "$FIRMWARE_FILE" 2>/dev/null)

echo -e "${GREEN}固件文件: ${FIRMWARE_FILE}${NC}"
echo -e "${GREEN}文件大小: ${FIRMWARE_SIZE} bytes${NC}"
echo -e "${GREEN}MD5摘要: ${FIRMWARE_MD5}${NC}"
echo ""

# 上传固件到临时HTTP服务器
# 注意：实际使用时需要将固件上传到可访问的HTTP服务器
# 这里使用Python简单HTTP服务器作为示例
echo "Starting temporary HTTP server for firmware download..."
FIRMWARE_PORT=8888
FIRMWARE_URL="http://localhost:${FIRMWARE_PORT}/$(basename "$FIRMWARE_FILE")"

# 启动HTTP服务器（后台运行）
cd /tmp
python3 -m http.server ${FIRMWARE_PORT} > /dev/null 2>&1 &
HTTP_SERVER_PID=$!
echo "HTTP server started with PID: ${HTTP_SERVER_PID}"
sleep 2

# 准备OTA推送请求
echo -e "${YELLOW}触发OTA更新...${NC}"

# 构建OTA任务参数
OTA_TASK=$(cat <<EOF
{
  "version": "${VERSION}",
  "url": "${FIRMWARE_URL}",
  "size": ${FIRMWARE_SIZE},
  "md5": "${FIRMWARE_MD5}",
  "signMethod": "md5",
  "sign": "${FIRMWARE_MD5}",
  "module": "default"
}
EOF
)

# Base64编码
OTA_TASK_BASE64=$(echo "$OTA_TASK" | base64 | tr -d '\n')

# 调用API触发OTA
echo "Calling OTA API..."
echo "Request payload:"
echo "$OTA_TASK" | jq . 2>/dev/null || echo "$OTA_TASK"

# 发送OTA任务（这里需要实际的API端点）
# 注意：实际的OTA触发需要通过IoT平台控制台或专用API
echo ""
echo -e "${YELLOW}注意：实际的OTA触发需要通过以下方式之一：${NC}"
echo "1. IoT平台控制台手动创建OTA任务"
echo "2. 使用平台提供的OTA管理API"
echo "3. 通过MQTT发送OTA通知消息"
echo ""
echo "示例MQTT消息格式："
echo "Topic: /ota/device/upgrade/${PRODUCT_KEY}/${DEVICE_NAME}"
echo "Payload:"
cat <<EOF | jq .
{
  "code": "1000",
  "data": {
    "size": ${FIRMWARE_SIZE},
    "version": "${VERSION}",
    "url": "${FIRMWARE_URL}",
    "md5": "${FIRMWARE_MD5}",
    "signMethod": "md5",
    "sign": "${FIRMWARE_MD5}",
    "module": "default"
  },
  "id": 1,
  "message": "success"
}
EOF

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}测试准备完成！${NC}"
echo ""
echo "请执行以下步骤："
echo "1. 确保电烤炉模拟器正在运行"
echo "2. 通过IoT平台控制台创建OTA任务"
echo "3. 观察模拟器日志中的OTA进度"
echo ""
echo "临时HTTP服务器运行在: ${FIRMWARE_URL}"
echo "按Ctrl+C结束测试并停止HTTP服务器"

# 等待用户中断
trap "echo 'Stopping HTTP server...'; kill $HTTP_SERVER_PID 2>/dev/null; rm -f $FIRMWARE_FILE; exit" INT TERM

while true; do
    sleep 1
done