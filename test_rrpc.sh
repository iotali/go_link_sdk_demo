#!/bin/bash

# Test RRPC with JSON format - LightSwitch method
echo "Testing LightSwitch method..."
curl -X POST "https://deviot.know-act.com/api/v1/device/rrpc" \
  -H "Content-Type: application/json" \
  -H "token: 488820fb-41af-40e5-b2d3-d45a8c576eea" \
  -d '{
    "deviceName": "WjJjXbP0X1",
    "productKey": "QLTMkOfW",
    "requestBase64Byte": "'$(echo '{"id":"123","version":"1.0","method":"LightSwitch","params":{"switch":1}}' | base64)'",
    "timeout": 5000
    }'


echo -e "\n\nTesting GetStatus method..."
curl -X POST "https://deviot.know-act.com/api/v1/device/rrpc" \
  -H "Content-Type: application/json" \
  -H "token: 488820fb-41af-40e5-b2d3-d45a8c576eea" \
  -d '{
    "deviceName": "WjJjXbP0X1",
    "productKey": "QLTMkOfW",
    "requestBase64Byte": "'$(echo '{"id":"456","version":"1.0","method":"GetStatus","params":{}}' | base64)'",
    "timeout": 5000
    }'
