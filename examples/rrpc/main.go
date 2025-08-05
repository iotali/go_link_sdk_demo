package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iot-go-sdk/pkg/config"
	"github.com/iot-go-sdk/pkg/mqtt"
	"github.com/iot-go-sdk/pkg/rrpc"
)

func main() {
	cfg := config.NewConfig()

	cfg.Device.ProductKey = "QLTMkOfW"
	cfg.Device.DeviceName = "WjJjXbP0X1"
	cfg.Device.DeviceSecret = "Vt1OU489RAylT8MV"

	cfg.MQTT.Host = "121.41.43.80"
	cfg.MQTT.Port = 8883
	cfg.MQTT.UseTLS = true

	cfg.TLS.SkipVerify = false  // 使用内置CA证书验证
	cfg.TLS.ServerName = "IoT"  // 匹配证书CN

	mqttClient := mqtt.NewClient(cfg)

	if err := mqttClient.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer mqttClient.Disconnect()

	log.Println("Connected to MQTT broker successfully")

	rrpcClient := rrpc.NewRRPCClient(mqttClient, cfg.Device.ProductKey, cfg.Device.DeviceName)

	rrpcClient.RegisterHandler("LightSwitch", func(requestId string, payload []byte) ([]byte, error) {
		log.Printf("Received RRPC request (ID: %s): %s", requestId, string(payload))

		response := map[string]interface{}{
			"LightSwitch": 0,
		}

		return json.Marshal(response)
	})

	rrpcClient.RegisterHandler("GetStatus", func(requestId string, payload []byte) ([]byte, error) {
		log.Printf("Received GetStatus request (ID: %s): %s", requestId, string(payload))

		response := map[string]interface{}{
			"status":      "online",
			"temperature": 25.6,
			"humidity":    60.3,
			"timestamp":   1234567890,
		}

		return json.Marshal(response)
	})

	if err := rrpcClient.Start(); err != nil {
		log.Fatalf("Failed to start RRPC client: %v", err)
	}
	defer rrpcClient.Stop()

	log.Println("RRPC client started. Waiting for requests...")
	log.Println("Registered handlers: LightSwitch, GetStatus")

	// Subscribe to a test topic to verify MQTT is working
	testTopic := "/" + cfg.Device.ProductKey + "/" + cfg.Device.DeviceName + "/user/test"
	if err := mqttClient.Subscribe(testTopic, 0, func(topic string, payload []byte) {
		log.Printf("Received test message on topic %s: %s", topic, string(payload))
	}); err != nil {
		log.Printf("Failed to subscribe to test topic: %v", err)
	} else {
		log.Printf("Also subscribed to test topic: %s", testTopic)
	}

	// Periodically check connection status and publish heartbeat
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if mqttClient.IsConnected() {
				log.Println("MQTT connection status: CONNECTED")
				// Publish a heartbeat message
				heartbeatTopic := "/" + cfg.Device.ProductKey + "/" + cfg.Device.DeviceName + "/user/heartbeat"
				heartbeatMsg := fmt.Sprintf(`{"timestamp": %d, "status": "alive"}`, time.Now().Unix())
				if err := mqttClient.Publish(heartbeatTopic, []byte(heartbeatMsg), 0, false); err != nil {
					log.Printf("Failed to publish heartbeat: %v", err)
				} else {
					log.Printf("Published heartbeat to %s", heartbeatTopic)
				}
			} else {
				log.Println("MQTT connection status: DISCONNECTED")
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
}
