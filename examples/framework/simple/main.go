package main

import (
	"log"
	"time"

	"github.com/iot-go-sdk/pkg/config"
	"github.com/iot-go-sdk/pkg/framework/core"
	"github.com/iot-go-sdk/pkg/framework/event"
	"github.com/iot-go-sdk/pkg/framework/plugins/mqtt"
)

// Note: Device implementation moved to electric_oven.go

func main() {

	// Create SDK configuration for MQTT plugin
	sdkConfig := config.NewConfig()
	sdkConfig.Device.ProductKey = "QLTMkOfW"
	sdkConfig.Device.DeviceName = "S4Wj7RZ5TO"
	sdkConfig.Device.DeviceSecret = "hM0PuEhzczeHTapI"
	sdkConfig.MQTT.Host = "121.40.253.224"
	sdkConfig.MQTT.Port = 1883
	sdkConfig.MQTT.UseTLS = false

	// Create framework configuration
	frameworkConfig := core.Config{
		Device: core.DeviceConfig{
			ProductKey:   "QLTMkOfW",
			DeviceName:   "S4Wj7RZ5TO",
			DeviceSecret: "hM0PuEhzczeHTapI",
		},
		MQTT: core.MQTTConfig{
			Host:          "121.40.253.224",
			Port:          1883,
			UseTLS:        false,
			KeepAlive:     60,
			CleanSession:  true,
			AutoReconnect: true,
			ReconnectMax:  10,
			Timeout:       10 * time.Second,
		},
		Features: core.FeatureConfig{
			EnableOTA:     true,
			EnableShadow:  false,
			EnableRules:   false,
			EnableMetrics: false,
		},
		Logging: core.LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
		Advanced: core.AdvancedConfig{
			WorkerCount:     10,
			EventBufferSize: 100,
			RequestTimeout:  30 * time.Second,
		},
	}

	// Create framework instance
	framework := core.New(frameworkConfig)

	// Initialize framework
	if err := framework.Initialize(frameworkConfig); err != nil {
		log.Fatalf("Failed to initialize framework: %v", err)
	}
	
	// Create and load MQTT plugin
	mqttPlugin := mqtt.NewMQTTPlugin(sdkConfig)
	if err := framework.LoadPlugin(mqttPlugin); err != nil {
		log.Fatalf("Failed to load MQTT plugin: %v", err)
	}

	// Create and register electric oven device
	oven := NewElectricOven(
		frameworkConfig.Device.ProductKey,
		frameworkConfig.Device.DeviceName,
		frameworkConfig.Device.DeviceSecret,
	)
	oven.SetFramework(framework)

	if err := framework.RegisterDevice(oven); err != nil {
		log.Fatalf("Failed to register device: %v", err)
	}

	// Register event handlers
	framework.On(event.EventConnected, func(evt *event.Event) error {
		log.Println("Framework connected to IoT platform")
		return nil
	})

	framework.On(event.EventDisconnected, func(evt *event.Event) error {
		log.Println("Framework disconnected from IoT platform")
		return nil
	})

	framework.On(event.EventError, func(evt *event.Event) error {
		log.Printf("Framework error: %v", evt.Data)
		return nil
	})

	framework.On(event.EventPropertyReport, func(evt *event.Event) error {
		log.Printf("Properties reported: %v", evt.Data)
		return nil
	})

	// Start framework
	if err := framework.Start(); err != nil {
		log.Fatalf("Failed to start framework: %v", err)
	}

	log.Println("Electric oven demo started. Press Ctrl+C to exit.")
	log.Println("Connecting to IoT platform via MQTT...")

	// Wait for shutdown
	framework.WaitForShutdown()

	log.Println("Electric oven demo stopped.")
}
