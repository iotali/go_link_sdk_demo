package main

import (
	"fmt"
	"log"

	"github.com/iot-go-sdk/pkg/auth"
	"github.com/iot-go-sdk/pkg/config"
)

func main() {
	fmt.Println("=== 调试认证信息 ===")
	
	cfg := config.NewConfig()
	cfg.Device.ProductKey = "QLTMkOfW"
	cfg.Device.DeviceName = "THYYENG5wd"
	cfg.Device.DeviceSecret = "1kaFD2aghSTw4aKV"
	cfg.MQTT.UseTLS = false
	
	fmt.Printf("配置信息:\n")
	fmt.Printf("  ProductKey: %s\n", cfg.Device.ProductKey)
	fmt.Printf("  DeviceName: %s\n", cfg.Device.DeviceName)
	fmt.Printf("  DeviceSecret: %s\n", cfg.Device.DeviceSecret)
	fmt.Printf("  UseTLS: %v\n", cfg.MQTT.UseTLS)
	fmt.Printf("  GetSecureMode(): %s\n", cfg.GetSecureMode())
	
	// 测试新的认证方式
	secureMode := cfg.GetSecureMode()
	credentials := auth.GenerateMQTTCredentials(
		cfg.Device.ProductKey,
		cfg.Device.DeviceName,
		cfg.Device.DeviceSecret,
		secureMode,
	)
	
	fmt.Printf("\n新认证方式 (SecureMode=%s):\n", secureMode)
	fmt.Printf("  ClientID: %s\n", credentials.ClientID)
	fmt.Printf("  Username: %s\n", credentials.Username)
	fmt.Printf("  Password: %s\n", credentials.Password)
	
	// 测试旧的认证方式
	legacyCredentials := auth.GenerateMQTTCredentialsLegacy(
		cfg.Device.ProductKey,
		cfg.Device.DeviceName,
		cfg.Device.DeviceSecret,
	)
	
	fmt.Printf("\n旧认证方式 (SecureMode=2):\n")
	fmt.Printf("  ClientID: %s\n", legacyCredentials.ClientID)
	fmt.Printf("  Username: %s\n", legacyCredentials.Username)
	fmt.Printf("  Password: %s\n", legacyCredentials.Password)
	
	// 比较密码是否相同
	if credentials.Password == legacyCredentials.Password {
		log.Println("\n✅ 新旧认证密码相同")
	} else {
		log.Println("\n❌ 新旧认证密码不同")
		log.Printf("  新密码: %s", credentials.Password)
		log.Printf("  旧密码: %s", legacyCredentials.Password)
	}
}