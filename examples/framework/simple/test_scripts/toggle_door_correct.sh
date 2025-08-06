#!/bin/bash

# 切换门状态服务测试脚本
# 用法: ./toggle_door_correct.sh

SERVER="https://deviot.know-act.com"
TOKEN="488820fb-41af-40e5-b2d3-d45a8c576eea"
DEVICE_NAME="S4Wj7RZ5TO"

echo "Toggling door status for device ${DEVICE_NAME}..."
echo ""

# 发送请求并保存响应
RESPONSE=$(curl -s -X POST "${SERVER}/api/v1/thing/invokeThingsService" \
  -H "Content-Type: application/json" \
  -H "token: ${TOKEN}" \
  -d "{
    \"deviceName\": \"${DEVICE_NAME}\",
    \"pointList\": [],
    \"servicePoint\": {
      \"identifier\": \"toggle_door\"
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
echo "Door toggle service invocation completed."