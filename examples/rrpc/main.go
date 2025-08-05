package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	cfg.TLS.SkipVerify = false

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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
}
