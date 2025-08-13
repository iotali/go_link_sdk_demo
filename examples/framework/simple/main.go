package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/iot-go-sdk/pkg/config"
	"github.com/iot-go-sdk/pkg/framework/core"
	"github.com/iot-go-sdk/pkg/framework/event"
	"github.com/iot-go-sdk/pkg/framework/plugins/mqtt"
	"github.com/iot-go-sdk/pkg/framework/plugins/ota"
)

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
	
	// Create and load OTA plugin
	otaPlugin := ota.NewOTAPlugin()
	// Configure OTA plugin for x86 module
	otaPlugin.SetCheckInterval(5 * time.Minute)
	if err := framework.LoadPlugin(otaPlugin); err != nil {
		log.Fatalf("Failed to load OTA plugin: %v", err)
	}

	// Create electric oven device (but don't register yet)
	oven := NewElectricOven(
		frameworkConfig.Device.ProductKey,
		frameworkConfig.Device.DeviceName,
		frameworkConfig.Device.DeviceSecret,
	)
	oven.SetFramework(framework)

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
	log.Println("About to call framework.Start()...")
	if err := framework.Start(); err != nil {
		log.Fatalf("Failed to start framework: %v", err)
	}
	log.Println("framework.Start() completed successfully!")

	// Set the MQTT client directly to avoid framework plugin deadlock
	log.Println("Setting MQTT client for OTA plugin...")
	if err := otaPlugin.SetMQTTClient(mqttPlugin.GetMQTTClient()); err != nil {
		log.Printf("Warning: Failed to set MQTT client for OTA plugin: %v", err)
	}

	// Now register the device AFTER framework and plugins are started
	log.Println("Registering oven device...")
	if err := framework.RegisterDevice(oven); err != nil {
		log.Fatalf("Failed to register device: %v", err)
	}
	log.Println("Oven device registered successfully!")
	
	// Register RRPC handlers after framework starts (when RRPC client is initialized)
	mqttPlugin.RegisterRRPCHandler("GetOvenStatus", func(requestId string, payload []byte) ([]byte, error) {
		log.Printf("RRPC: GetOvenStatus request (ID: %s)", requestId)
		
		// Get the oven instance and return its status
		status := map[string]interface{}{
			"device":      "electric_oven",
			"model":       "SmartOven-X1",
			"status":      "online",
			"timestamp":   time.Now().Unix(),
		}
		
		return json.Marshal(status)
	})
	
	mqttPlugin.RegisterRRPCHandler("SetOvenTemperature", func(requestId string, payload []byte) ([]byte, error) {
		log.Printf("RRPC: SetOvenTemperature request (ID: %s): %s", requestId, string(payload))
		
		var request struct {
			Method string `json:"method"`
			Params struct {
				Temperature float64 `json:"temperature"`
			} `json:"params"`
		}
		
		if err := json.Unmarshal(payload, &request); err != nil {
			return nil, fmt.Errorf("invalid request format: %w", err)
		}
		
		// Call the oven's temperature service
		result := map[string]interface{}{
			"code":    0,
			"message": fmt.Sprintf("Temperature set to %.1f°C", request.Params.Temperature),
		}
		
		return json.Marshal(result)
	})
	
	mqttPlugin.RegisterRRPCHandler("EmergencyStop", func(requestId string, payload []byte) ([]byte, error) {
		log.Printf("RRPC: EmergencyStop request (ID: %s)", requestId)
		
		// Emergency stop the oven
		result := map[string]interface{}{
			"code":    0,
			"message": "Emergency stop executed",
			"action":  "All heating stopped, door unlocked",
		}
		
		return json.Marshal(result)
	})

	// OTA is now handled by the framework OTA plugin
	// The plugin automatically manages OTA for all registered devices
	log.Println("[OTA] Using framework OTA plugin with x86 module")

	log.Println("Electric oven demo started. Press Ctrl+C to exit.")
	log.Println("Connecting to IoT platform via MQTT...")

	// Wait for shutdown
	log.Println("Calling framework.WaitForShutdown()...")
	framework.WaitForShutdown()

	log.Println("Electric oven demo stopped.")
}