package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// Test different topic patterns
	topics := []string{
		"/sys/" + cfg.Device.ProductKey + "/" + cfg.Device.DeviceName + "/rrpc/request/+",
		"/" + cfg.Device.ProductKey + "/" + cfg.Device.DeviceName + "/rrpc/request/+",
		"/sys/" + cfg.Device.ProductKey + "/" + cfg.Device.DeviceName + "/#",
		"/" + cfg.Device.ProductKey + "/" + cfg.Device.DeviceName + "/#",
		"$sys/" + cfg.Device.ProductKey + "/" + cfg.Device.DeviceName + "/rrpc/request/+",
	}

	for _, topic := range topics {
		err := client.Subscribe(topic, 0, func(receivedTopic string, payload []byte) {
			log.Printf("✅ Received message on topic '%s': %s", receivedTopic, string(payload))
			log.Printf("   Payload hex: %x", payload)
		})
		if err != nil {
			log.Printf("❌ Failed to subscribe to topic '%s': %v", topic, err)
		} else {
			log.Printf("✅ Successfully subscribed to topic: %s", topic)
		}
	}

	// Publish test messages to see if we can receive our own messages
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		counter := 0
		for range ticker.C {
			counter++
			testTopic := "/" + cfg.Device.ProductKey + "/" + cfg.Device.DeviceName + "/user/test"
			testMessage := []byte("Test message " + string(rune(counter+'0')))
			
			if err := client.Publish(testTopic, testMessage, 0, false); err != nil {
				log.Printf("Failed to publish test message: %v", err)
			} else {
				log.Printf("Published test message to %s", testTopic)
			}
		}
	}()

	log.Println("Waiting for messages... Press Ctrl+C to exit")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
}