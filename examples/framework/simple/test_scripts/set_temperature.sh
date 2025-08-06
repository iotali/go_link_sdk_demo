#!/bin/bash

# 设置温度测试脚本
# 用法: ./set_temperature.sh [温度值]

TEMP=${1:-180}
SERVER="https://deviot.know-act.com"
TOKEN="488820fb-41af-40e5-b2d3-d45a8c576eea"
DEVICE_NAME="S4Wj7RZ5TO"

echo "Setting temperature to ${TEMP}°C for device ${DEVICE_NAME}..."
echo "API Endpoint: ${SERVER}/api/v1/thing/setDevicesProperty"
echo ""

# 发送请求并保存响应
RESPONSE=$(curl -s -X POST "${SERVER}/api/v1/thing/setDevicesProperty" \
  -H "Content-Type: application/json" \
  -H "token: ${TOKEN}" \
  -d "{
    \"deviceName\": \"${DEVICE_NAME}\",
    \"pointList\": [
      {
        \"identifier\": \"target_temperature\",
        \"value\": \"${TEMP}\"
      }
    ]
  }")

# 检查是否安装了jq
if command -v jq &> /dev/null; then
    echo "Response:"
    echo "$RESPONSE" | jq .
else
    echo "Response:"
    echo "$RESPONSE"
fi

echo ""
echo "Temperature set request completed."