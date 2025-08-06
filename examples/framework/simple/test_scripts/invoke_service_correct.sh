#!/bin/bash

# 调用服务的正确格式 - 参数在pointList中
# 用法: ./invoke_service_correct.sh [服务名] [参数标识符] [参数值]

SERVICE=${1:-start_timer}
PARAM_ID=${2:-time}
PARAM_VALUE=${3:-5}

SERVER="https://deviot.know-act.com"
TOKEN="488820fb-41af-40e5-b2d3-d45a8c576eea"
DEVICE_NAME="S4Wj7RZ5TO"

echo "Invoking service '${SERVICE}' with ${PARAM_ID}=${PARAM_VALUE}"
echo "Device: ${DEVICE_NAME}"
echo ""

# 发送请求并保存响应
RESPONSE=$(curl -s -X POST "${SERVER}/api/v1/thing/invokeThingsService" \
  -H "Content-Type: application/json" \
  -H "token: ${TOKEN}" \
  -d "{
    \"deviceName\": \"${DEVICE_NAME}\",
    \"pointList\": [
      {
        \"identifier\": \"${PARAM_ID}\",
        \"value\": \"${PARAM_VALUE}\"
      }
    ],
    \"servicePoint\": {
      \"identifier\": \"${SERVICE}\"
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
echo "Service invocation completed."