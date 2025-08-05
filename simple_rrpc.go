package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/iot-go-sdk/pkg/config"
	"github.com/iot-go-sdk/pkg/mqtt"
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
	cfg.TLS.ServerName = "IoT"

	client := mqtt.NewClient(cfg)

	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	log.Println("Connected to MQTT broker successfully")

	// Subscribe to RRPC request topic
	rrpcTopic := "/sys/" + cfg.Device.ProductKey + "/" + cfg.Device.DeviceName + "/rrpc/request/+"
	err := client.Subscribe(rrpcTopic, 0, func(topic string, payload []byte) {
		log.Printf("🔥 RECEIVED RRPC REQUEST!")
		log.Printf("Topic: %s", topic)
		log.Printf("Payload: %s", string(payload))
		log.Printf("Payload hex: %x", payload)
		
		// Extract request ID from topic
		// Format: /sys/ProductKey/DeviceName/rrpc/request/{requestId}
		requestId := ""
		if len(topic) > len(rrpcTopic)-1 {
			requestId = topic[len(rrpcTopic)-1:]
		}
		
		log.Printf("Request ID: %s", requestId)
		
		if requestId != "" {
			// Send a simple response
			responseTopic := "/sys/" + cfg.Device.ProductKey + "/" + cfg.Device.DeviceName + "/rrpc/response/" + requestId
			response := map[string]interface{}{
				"id":      "1",
				"code":    200,
				"data":    map[string]interface{}{"result": "success", "LightSwitch": 1},
				"message": "OK",
			}
			
			responseData, _ := json.Marshal(response)
			
			if err := client.Publish(responseTopic, responseData, 0, false); err != nil {
				log.Printf("Failed to send response: %v", err)
			} else {
				log.Printf("✅ Sent response to: %s", responseTopic)
				log.Printf("Response: %s", string(responseData))
			}
		}
	})

	if err != nil {
		log.Fatalf("Failed to subscribe to RRPC topic: %v", err)
	}

	log.Printf("Subscribed to RRPC topic: %s", rrpcTopic)
	log.Println("Waiting for RRPC requests... Press Ctrl+C to exit")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
}