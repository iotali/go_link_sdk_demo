#!/bin/bash

# RRPC API测试脚本 - 符合实际API规范
# 使用Base64编码的请求和响应

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
SERVER="https://deviot.know-act.com"  # 实际API地址
TOKEN="488820fb-41af-40e5-b2d3-d45a8c576eea"
PRODUCT_KEY="QLTMkOfW"
DEVICE_NAME="S4Wj7RZ5TO"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}RRPC API 测试 (Base64编码)${NC}"
echo -e "${GREEN}========================================${NC}"

# 辅助函数：将字符串编码为Base64
encode_base64() {
    echo -n "$1" | base64
}

# 辅助函数：解码Base64
decode_base64() {
    echo "$1" | base64 -d
}

# 测试1: 发送GetOvenStatus请求
echo -e "\n${YELLOW}测试1: GetOvenStatus - 获取烤炉状态${NC}"

# 准备请求数据
REQUEST_JSON='{"method":"GetOvenStatus","params":{}}'
REQUEST_BASE64=$(encode_base64 "$REQUEST_JSON")

echo "原始请求: $REQUEST_JSON"
echo "Base64编码: $REQUEST_BASE64"
echo ""

RESPONSE=$(curl -s -X POST "${SERVER}/api/v1/device/rrpc" \
  -H "Content-Type: application/json" \
  -H "token: ${TOKEN}" \
  -d "{
    \"deviceName\": \"${DEVICE_NAME}\",
    \"productKey\": \"${PRODUCT_KEY}\",
    \"requestBase64Byte\": \"${REQUEST_BASE64}\",
    \"timeout\": 5000
  }")

echo -e "${BLUE}API响应:${NC}"
echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"

# 解码响应
if echo "$RESPONSE" | jq -e '.playloadBase64Byte' > /dev/null 2>&1; then
    PAYLOAD_BASE64=$(echo "$RESPONSE" | jq -r '.playloadBase64Byte')
    DECODED_PAYLOAD=$(decode_base64 "$PAYLOAD_BASE64")
    echo -e "${GREEN}解码后的设备响应:${NC}"
    echo "$DECODED_PAYLOAD" | jq . 2>/dev/null || echo "$DECODED_PAYLOAD"
fi

sleep 2

# 测试2: 设置温度
echo -e "\n${YELLOW}测试2: SetOvenTemperature - 设置烤炉温度${NC}"

REQUEST_JSON='{"method":"SetOvenTemperature","params":{"temperature":200}}'
REQUEST_BASE64=$(encode_base64 "$REQUEST_JSON")

echo "原始请求: $REQUEST_JSON"
echo "Base64编码: $REQUEST_BASE64"
echo ""

RESPONSE=$(curl -s -X POST "${SERVER}/api/v1/device/rrpc" \
  -H "Content-Type: application/json" \
  -H "token: ${TOKEN}" \
  -d "{
    \"deviceName\": \"${DEVICE_NAME}\",
    \"productKey\": \"${PRODUCT_KEY}\",
    \"requestBase64Byte\": \"${REQUEST_BASE64}\",
    \"timeout\": 5000
  }")

echo -e "${BLUE}API响应:${NC}"
echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"

if echo "$RESPONSE" | jq -e '.playloadBase64Byte' > /dev/null 2>&1; then
    PAYLOAD_BASE64=$(echo "$RESPONSE" | jq -r '.playloadBase64Byte')
    DECODED_PAYLOAD=$(decode_base64 "$PAYLOAD_BASE64")
    echo -e "${GREEN}解码后的设备响应:${NC}"
    echo "$DECODED_PAYLOAD" | jq . 2>/dev/null || echo "$DECODED_PAYLOAD"
fi

sleep 2

# 测试3: 紧急停止
echo -e "\n${YELLOW}测试3: EmergencyStop - 紧急停止${NC}"

REQUEST_JSON='{"method":"EmergencyStop","params":{}}'
REQUEST_BASE64=$(encode_base64 "$REQUEST_JSON")

echo "原始请求: $REQUEST_JSON"
echo "Base64编码: $REQUEST_BASE64"
echo ""

RESPONSE=$(curl -s -X POST "${SERVER}/api/v1/device/rrpc" \
  -H "Content-Type: application/json" \
  -H "token: ${TOKEN}" \
  -d "{
    \"deviceName\": \"${DEVICE_NAME}\",
    \"productKey\": \"${PRODUCT_KEY}\",
    \"requestBase64Byte\": \"${REQUEST_BASE64}\",
    \"timeout\": 5000
  }")

echo -e "${BLUE}API响应:${NC}"
echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"

if echo "$RESPONSE" | jq -e '.playloadBase64Byte' > /dev/null 2>&1; then
    PAYLOAD_BASE64=$(echo "$RESPONSE" | jq -r '.playloadBase64Byte')
    DECODED_PAYLOAD=$(decode_base64 "$PAYLOAD_BASE64")
    echo -e "${GREEN}解码后的设备响应:${NC}"
    echo "$DECODED_PAYLOAD" | jq . 2>/dev/null || echo "$DECODED_PAYLOAD"
fi

sleep 2

# 测试4: 测试二进制数据
echo -e "\n${YELLOW}测试4: 二进制数据测试${NC}"

# 创建一个包含二进制数据的请求
# 0x01 0x03 0x00 0x00 0x00 0x01 0x84 0x0A (Modbus RTU示例)
BINARY_HEX="0103000000018410"
REQUEST_BASE64="AQMAAAABhBA="  # 上述十六进制的Base64编码

echo "原始十六进制: $BINARY_HEX"
echo "Base64编码: $REQUEST_BASE64"
echo ""

RESPONSE=$(curl -s -X POST "${SERVER}/api/v1/device/rrpc" \
  -H "Content-Type: application/json" \
  -H "token: ${TOKEN}" \
  -d "{
    \"deviceName\": \"${DEVICE_NAME}\",
    \"productKey\": \"${PRODUCT_KEY}\",
    \"requestBase64Byte\": \"${REQUEST_BASE64}\",
    \"timeout\": 5000
  }")

echo -e "${BLUE}API响应:${NC}"
echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"

if echo "$RESPONSE" | jq -e '.playloadBase64Byte' > /dev/null 2>&1; then
    PAYLOAD_BASE64=$(echo "$RESPONSE" | jq -r '.playloadBase64Byte')
    echo -e "${GREEN}Base64响应: $PAYLOAD_BASE64${NC}"
    # 尝试解码为十六进制
    echo -e "${GREEN}十六进制响应:${NC}"
    echo "$PAYLOAD_BASE64" | base64 -d | xxd -p
fi

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}测试完成${NC}"
echo -e "${GREEN}========================================${NC}"

# 显示状态码说明
echo -e "\n${YELLOW}RRPC状态码说明:${NC}"
echo "- SUCCESS: 调用成功，设备已响应"
echo "- TIMEOUT: 调用超时，未收到设备响应"
echo "- OFFLINE: 设备离线"

echo -e "\n${BLUE}注意事项:${NC}"
echo "1. 请求数据会自动进行Base64编码"
echo "2. 响应数据需要Base64解码后才能读取"
echo "3. 设备端接收的是原始字节数据，不需要解码"
echo "4. 设备端响应也是原始字节，平台会自动编码"