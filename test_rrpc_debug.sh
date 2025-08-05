#!/bin/bash

# Debug RRPC test script
echo "=== RRPC Debug Test ==="
echo "Device: WjJjXbP0X1"
echo "Product: QLTMkOfW"
echo ""

# Prepare JSON payload
JSON_PAYLOAD='{"id":"123","version":"1.0","method":"LightSwitch","params":{"switch":1}}'
BASE64_PAYLOAD=$(echo "$JSON_PAYLOAD" | base64)

echo "JSON Payload: $JSON_PAYLOAD"
echo "Base64 Payload: $BASE64_PAYLOAD"
echo ""

echo "Sending RRPC request..."
echo "========================"

curl -v -X POST "https://deviot.know-act.com/api/v1/device/rrpc" \
  -H "Content-Type: application/json" \
  -H "token: 488820fb-41af-40e5-b2d3-d45a8c576eea" \
  -d "{
    \"deviceName\": \"WjJjXbP0X1\",
    \"productKey\": \"QLTMkOfW\",
    \"requestBase64Byte\": \"$BASE64_PAYLOAD\",
    \"timeout\": 5000
  }"

echo ""
echo "========================"