#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
RRPC API测试脚本 - Python版本
支持Base64编解码和完整的API测试
"""

import base64
import json
import time
import requests
from typing import Dict, Any, Optional

# 配置
SERVER = "https://deviot.know-act.com"
TOKEN = "488820fb-41af-40e5-b2d3-d45a8c576eea"
PRODUCT_KEY = "QLTMkOfW"
DEVICE_NAME = "S4Wj7RZ5TO"

class RRPCTester:
    """RRPC测试器"""
    
    def __init__(self, server: str, token: str, product_key: str, device_name: str):
        self.server = server
        self.token = token
        self.product_key = product_key
        self.device_name = device_name
        self.headers = {
            "Content-Type": "application/json",
            "token": self.token
        }
    
    def encode_request(self, data: Any) -> str:
        """将数据编码为Base64"""
        if isinstance(data, dict):
            json_str = json.dumps(data, separators=(',', ':'))
            return base64.b64encode(json_str.encode('utf-8')).decode('ascii')
        elif isinstance(data, bytes):
            return base64.b64encode(data).decode('ascii')
        else:
            return base64.b64encode(str(data).encode('utf-8')).decode('ascii')
    
    def decode_response(self, base64_str: str) -> Any:
        """解码Base64响应"""
        try:
            decoded = base64.b64decode(base64_str)
            # 尝试解析为JSON
            try:
                return json.loads(decoded)
            except:
                # 如果不是JSON，返回原始字节
                return decoded
        except Exception as e:
            print(f"解码错误: {e}")
            return None
    
    def call_rrpc(self, request_data: Any, timeout: int = 5000) -> Dict[str, Any]:
        """调用RRPC API"""
        # 编码请求数据
        request_base64 = self.encode_request(request_data)
        
        # 构建API请求
        api_request = {
            "deviceName": self.device_name,
            "productKey": self.product_key,
            "requestBase64Byte": request_base64,
            "timeout": timeout
        }
        
        print(f"原始请求: {request_data}")
        print(f"Base64编码: {request_base64}")
        print(f"API请求: {json.dumps(api_request, indent=2)}")
        
        # 发送请求
        try:
            response = requests.post(
                f"{self.server}/api/v1/device/rrpc",
                headers=self.headers,
                json=api_request,
                timeout=timeout/1000 + 5  # 额外5秒的HTTP超时
            )
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"请求失败: {e}")
            return {"success": False, "error": str(e)}
    
    def test_get_oven_status(self):
        """测试获取烤炉状态"""
        print("\n" + "="*50)
        print("测试1: GetOvenStatus - 获取烤炉状态")
        print("="*50)
        
        request = {
            "method": "GetOvenStatus",
            "params": {}
        }
        
        response = self.call_rrpc(request)
        self.print_response(response)
    
    def test_set_temperature(self, temperature: float = 180.0):
        """测试设置温度"""
        print("\n" + "="*50)
        print(f"测试2: SetOvenTemperature - 设置温度为{temperature}°C")
        print("="*50)
        
        request = {
            "method": "SetOvenTemperature",
            "params": {
                "temperature": temperature
            }
        }
        
        response = self.call_rrpc(request)
        self.print_response(response)
    
    def test_emergency_stop(self):
        """测试紧急停止"""
        print("\n" + "="*50)
        print("测试3: EmergencyStop - 紧急停止")
        print("="*50)
        
        request = {
            "method": "EmergencyStop",
            "params": {}
        }
        
        response = self.call_rrpc(request)
        self.print_response(response)
    
    def test_binary_data(self):
        """测试二进制数据"""
        print("\n" + "="*50)
        print("测试4: 二进制数据测试")
        print("="*50)
        
        # Modbus RTU示例数据
        binary_data = bytes([0x01, 0x03, 0x00, 0x00, 0x00, 0x01, 0x84, 0x0A])
        print(f"原始二进制: {binary_data.hex()}")
        
        response = self.call_rrpc(binary_data)
        self.print_response(response)
    
    def test_invoke_service(self, service_name: str = "toggle_door"):
        """测试调用框架服务"""
        print("\n" + "="*50)
        print(f"测试5: InvokeService - 调用{service_name}服务")
        print("="*50)
        
        request = {
            "method": "InvokeService",
            "params": {
                "service": service_name,
                "params": {}
            }
        }
        
        response = self.call_rrpc(request)
        self.print_response(response)
    
    def print_response(self, response: Dict[str, Any]):
        """打印响应结果"""
        print("\nAPI响应:")
        print(json.dumps(response, indent=2, ensure_ascii=False))
        
        if response.get("success"):
            rrpc_code = response.get("rrpcCode", "")
            
            if rrpc_code == "SUCCESS":
                print("\n✅ RRPC调用成功")
                
                # 解码设备响应
                payload_base64 = response.get("playloadBase64Byte", "")
                if payload_base64:
                    decoded = self.decode_response(payload_base64)
                    print("\n解码后的设备响应:")
                    if isinstance(decoded, dict):
                        print(json.dumps(decoded, indent=2, ensure_ascii=False))
                    elif isinstance(decoded, bytes):
                        print(f"二进制数据: {decoded.hex()}")
                    else:
                        print(decoded)
            
            elif rrpc_code == "TIMEOUT":
                print("\n⏱️ RRPC调用超时")
            
            elif rrpc_code == "OFFLINE":
                print("\n❌ 设备离线")
            
            else:
                print(f"\n⚠️ 未知状态码: {rrpc_code}")
        else:
            print("\n❌ API调用失败")
    
    def run_all_tests(self):
        """运行所有测试"""
        print("="*60)
        print("RRPC API测试套件")
        print("="*60)
        print(f"服务器: {self.server}")
        print(f"产品: {self.product_key}")
        print(f"设备: {self.device_name}")
        
        # 运行测试
        self.test_get_oven_status()
        time.sleep(2)
        
        self.test_set_temperature(200)
        time.sleep(2)
        
        self.test_emergency_stop()
        time.sleep(2)
        
        self.test_binary_data()
        time.sleep(2)
        
        self.test_invoke_service("toggle_door")
        
        print("\n" + "="*60)
        print("测试完成！")
        print("="*60)
        
        # 打印说明
        print("\nRRPC状态码说明:")
        print("- SUCCESS: 调用成功，设备已响应")
        print("- TIMEOUT: 调用超时，未收到设备响应")
        print("- OFFLINE: 设备离线")
        
        print("\n注意事项:")
        print("1. 请求数据会自动进行Base64编码")
        print("2. 响应数据需要Base64解码后才能读取")
        print("3. 设备端接收的是原始字节数据")
        print("4. 设备端响应也是原始字节")


def main():
    """主函数"""
    tester = RRPCTester(SERVER, TOKEN, PRODUCT_KEY, DEVICE_NAME)
    
    # 运行所有测试
    tester.run_all_tests()
    
    # 或者运行单个测试
    # tester.test_get_oven_status()
    # tester.test_set_temperature(180)
    # tester.test_emergency_stop()


if __name__ == "__main__":
    main()