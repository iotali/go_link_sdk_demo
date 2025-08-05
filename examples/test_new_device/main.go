package main

import (
	"log"
	"time"

	"github.com/iot-go-sdk/pkg/config"
	"github.com/iot-go-sdk/pkg/mqtt"
)

func main() {
	cfg := config.NewConfig()
	
	// 使用新的三元组
	cfg.Device.ProductKey = "QLTMkOfW"
	cfg.Device.DeviceName = "WjJjXbP0X1"
	cfg.Device.DeviceSecret = "Vt1OU489RAylT8MV"
	
	// 先测试非TLS连接
	cfg.MQTT.Host = "121.40.253.224"
	cfg.MQTT.Port = 1883
	cfg.MQTT.UseTLS = false
	
	log.Printf("测试新设备三元组的非TLS连接...")
	log.Printf("Host: %s:%d", cfg.MQTT.Host, cfg.MQTT.Port)
	log.Printf("ProductKey: %s", cfg.Device.ProductKey)
	log.Printf("DeviceName: %s", cfg.Device.DeviceName)
	log.Printf("SecureMode: %s", cfg.GetSecureMode())
	
	client := mqtt.NewClient(cfg)
	
	if err := client.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer client.Disconnect()
	
	log.Println("✅ 连接成功!")
	
	topic := "/" + cfg.Device.ProductKey + "/" + cfg.Device.DeviceName + "/user/update"
	
	if err := client.Subscribe(topic, 0, func(topic string, payload []byte) {
		log.Printf("收到消息: %s", string(payload))
	}); err != nil {
		log.Fatalf("订阅失败: %v", err)
	}
	
	message := "Hello from new device!"
	if err := client.Publish(topic, []byte(message), 0, false); err != nil {
		log.Printf("发布失败: %v", err)
	} else {
		log.Printf("✅ 发布成功: %s", message)
	}
	
	time.Sleep(2 * time.Second)
	log.Println("测试完成!")
}