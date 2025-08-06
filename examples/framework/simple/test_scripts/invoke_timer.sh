#!/bin/bash

# 调用定时器服务测试脚本
# 用法: ./invoke_timer.sh [分钟数]

MINUTES=${1:-5}
SERVER="https://deviot.know-act.com"
TOKEN="488820fb-41af-40e5-b2d3-d45a8c576eea"
DEVICE_NAME="S4Wj7RZ5TO"

echo "Starting timer for ${MINUTES} minutes on device ${DEVICE_NAME}..."
echo "API Endpoint: ${SERVER}/api/v1/thing/invokeThingsService"
echo ""

# 发送请求并保存响应
RESPONSE=$(curl -s -X POST "${SERVER}/api/v1/thing/invokeThingsService" \
  -H "Content-Type: application/json" \
  -H "token: ${TOKEN}" \
  -d "{
    \"deviceName\": \"${DEVICE_NAME}\",
    \"pointList\": [
      {
        \"identifier\": \"time\",
        \"value\": \"${MINUTES}\"
      }
    ],
    \"servicePoint\": {
      \"identifier\": \"start_timer\"
    }
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
echo "Timer service invocation completed."