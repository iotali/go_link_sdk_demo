package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iot-go-sdk/pkg/config"
	"github.com/iot-go-sdk/pkg/mqtt"
)

func testSecureMode(secureMode string, useTLS bool, port int) {
	fmt.Printf("\n=== 测试 SecureMode=%s, TLS=%v, Port=%d ===\n", secureMode, useTLS, port)
	
	cfg := config.NewConfig()
	
	cfg.Device.ProductKey = "QLTMkOfW"
	cfg.Device.DeviceName = "THYYENG5wd"
	cfg.Device.DeviceSecret = "1kaFD2aghSTw4aKV"
	
	cfg.MQTT.Host = "121.40.253.224"
	cfg.MQTT.Port = port
	cfg.MQTT.UseTLS = useTLS
	cfg.MQTT.SecureMode = secureMode
	
	cfg.TLS.SkipVerify = true // 暂时跳过证书验证
	
	client := mqtt.NewClient(cfg)
	
	fmt.Printf("配置信息:\n")
	fmt.Printf("  Host: %s:%d\n", cfg.MQTT.Host, cfg.MQTT.Port)
	fmt.Printf("  TLS: %v\n", cfg.MQTT.UseTLS)
	fmt.Printf("  SecureMode: %s (实际使用: %s)\n", cfg.MQTT.SecureMode, cfg.GetSecureMode())
	
	if err := client.Connect(); err != nil {
		log.Printf("连接失败: %v", err)
		return
	}
	defer client.Disconnect()
	
	log.Printf("连接成功! SecureMode=%s", cfg.GetSecureMode())
	
	topic := "/" + cfg.Device.ProductKey + "/" + cfg.Device.DeviceName + "/user/update"
	
	if err := client.Subscribe(topic, 0, func(topic string, payload []byte) {
		log.Printf("收到消息: %s", string(payload))
	}); err != nil {
		log.Printf("订阅失败: %v", err)
		return
	}
	
	message := fmt.Sprintf("测试消息 - SecureMode:%s, TLS:%v", cfg.GetSecureMode(), cfg.MQTT.UseTLS)
	
	if err := client.Publish(topic, []byte(message), 0, false); err != nil {
		log.Printf("发布失败: %v", err)
	} else {
		log.Printf("发布成功: %s", message)
	}
	
	time.Sleep(2 * time.Second)
}

func main() {
	log.Println("开始测试不同的 SecureMode 配置...")
	
	// 测试1: 明确指定 securemode=3，不使用TLS
	testSecureMode("3", false, 1883)
	
	// 测试2: 明确指定 securemode=2，使用TLS
	testSecureMode("2", true, 8883)
	
	// 测试3: 不指定 securemode，不使用TLS (应该自动设为3)
	cfg := config.NewConfig()
	cfg.Device.ProductKey = "QLTMkOfW"
	cfg.Device.DeviceName = "THYYENG5wd"
	cfg.Device.DeviceSecret = "1kaFD2aghSTw4aKV"
	cfg.MQTT.Host = "121.40.253.224"
	cfg.MQTT.Port = 1883
	cfg.MQTT.UseTLS = false
	
	fmt.Printf("\n=== 测试自动判断 SecureMode (TLS=false) ===\n")
	fmt.Printf("自动判断的 SecureMode: %s\n", cfg.GetSecureMode())
	
	// 测试4: 不指定 securemode，使用TLS (应该自动设为2)
	cfg.MQTT.UseTLS = true
	cfg.MQTT.Port = 8883
	
	fmt.Printf("\n=== 测试自动判断 SecureMode (TLS=true) ===\n")
	fmt.Printf("自动判断的 SecureMode: %s\n", cfg.GetSecureMode())
	
	fmt.Println("\n所有测试完成!")
	
	// 等待用户中断
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	log.Println("程序退出")
}