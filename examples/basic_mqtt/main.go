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
	cfg.Device.DeviceName = "THYYENG5wd"
	cfg.Device.DeviceSecret = "1kaFD2aghSTw4aKV"

	cfg.MQTT.Host = "121.40.253.224"
	cfg.MQTT.Port = 1883
	cfg.MQTT.UseTLS = false

	client := mqtt.NewClient(cfg)

	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	log.Println("Connected to MQTT broker successfully")

	topic := "/" + cfg.Device.ProductKey + "/" + cfg.Device.DeviceName + "/user/update"

	if err := client.Subscribe(topic, 0, func(topic string, payload []byte) {
		log.Printf("Received message on topic %s: %s", topic, string(payload))
	}); err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		counter := 0
		for range ticker.C {
			counter++
			message := []byte("Hello from Go IoT SDK #" + string(rune(counter+'0')))

			if err := client.Publish(topic, message, 0, false); err != nil {
				log.Printf("Failed to publish message: %v", err)
			} else {
				log.Printf("Published message: %s", string(message))
			}
		}
	}()

	log.Println("Starting message loop. Press Ctrl+C to exit...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
}
