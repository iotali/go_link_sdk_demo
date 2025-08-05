package main

import (
	"log"
	"time"

	"github.com/iot-go-sdk/pkg/config"
	"github.com/iot-go-sdk/pkg/dynreg"
)

func main() {
	cfg := config.NewConfig()

	cfg.Device.ProductKey = "xmzgAofi"
	cfg.Device.DeviceName = "02H0UA0TzL"
	cfg.Device.ProductSecret = "MUxIfsXrXwCUdhAg"

	cfg.MQTT.Host = "121.41.43.80"
	cfg.MQTT.Port = 8883
	cfg.MQTT.UseTLS = true

	cfg.TLS.SkipVerify = true // 跳过证书验证（自签名证书）

	client := dynreg.NewMQTTDynRegClient(cfg)

	log.Println("Starting MQTT dynamic registration...")

	skipPreRegist := false  // Use whitelist mode (same as C SDK, skip_pre_regist = 0)
	timeout := 60 * time.Second

	responseData, err := client.Register(skipPreRegist, timeout)
	if err != nil {
		log.Fatalf("MQTT dynamic registration failed: %v", err)
	}

	log.Printf("MQTT dynamic registration successful!")

	if responseData.DeviceSecret != "" {
		log.Printf("Device Secret: %s", responseData.DeviceSecret)
		cfg.Device.DeviceSecret = responseData.DeviceSecret
	}

	if responseData.ClientId != "" {
		log.Printf("Client ID: %s", responseData.ClientId)
		cfg.MQTT.ClientID = responseData.ClientId
	}

	if responseData.Username != "" {
		log.Printf("Username: %s", responseData.Username)
		cfg.MQTT.Username = responseData.Username
	}

	if responseData.Password != "" {
		log.Printf("Password: %s", responseData.Password)
		cfg.MQTT.Password = responseData.Password
	}

	log.Println("You can now use these credentials for MQTT connection")
}
