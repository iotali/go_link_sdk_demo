package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/iot-go-sdk/pkg/config"
	"github.com/iot-go-sdk/pkg/framework/core"
	"github.com/iot-go-sdk/pkg/framework/event"
	"github.com/iot-go-sdk/pkg/framework/plugins/mqtt"
)

// SmartSensor represents a smart temperature and humidity sensor
type SmartSensor struct {
	core.BaseDevice

	// Sensor data
	temperature float64
	humidity    float64
	online      bool

	// Framework reference
	framework core.Framework

	// Simulation ticker
	ticker *time.Ticker
}

// NewSmartSensor creates a new smart sensor device
func NewSmartSensor(productKey, deviceName, deviceSecret string) *SmartSensor {
	return &SmartSensor{
		BaseDevice: core.BaseDevice{
			DeviceInfo: core.DeviceInfo{
				ProductKey:   productKey,
				DeviceName:   deviceName,
				DeviceSecret: deviceSecret,
				Model:        "SmartSensor-1000",
				Version:      "1.0.0",
			},
		},
		temperature: 25.0,
		humidity:    60.0,
		online:      false,
	}
}

// OnInitialize is called when the device is initialized
func (s *SmartSensor) OnInitialize(ctx context.Context) error {
	log.Printf("[%s] Initializing smart sensor...", s.DeviceInfo.DeviceName)

	// Register properties
	s.framework.RegisterProperty("temperature", s.getTemperature, nil)
	s.framework.RegisterProperty("humidity", s.getHumidity, nil)
	s.framework.RegisterProperty("online", s.getOnline, nil)

	// Register services
	s.framework.RegisterService("calibrate", s.calibrate)
	s.framework.RegisterService("reset", s.reset)

	// Start sensor simulation (generate random data)
	s.startSimulation()

	return nil
}

// OnConnect is called when the device connects to the platform
func (s *SmartSensor) OnConnect(ctx context.Context) error {
	log.Printf("[%s] Connected to IoT platform", s.DeviceInfo.DeviceName)
	s.online = true

	// Report initial state
	s.reportStatus()

	return nil
}

// OnDisconnect is called when the device disconnects from the platform
func (s *SmartSensor) OnDisconnect(ctx context.Context) error {
	log.Printf("[%s] Disconnected from IoT platform", s.DeviceInfo.DeviceName)
	s.online = false
	return nil
}

// OnDestroy is called when the device is being destroyed
func (s *SmartSensor) OnDestroy(ctx context.Context) error {
	log.Printf("[%s] Destroying smart sensor...", s.DeviceInfo.DeviceName)

	// Stop simulation
	if s.ticker != nil {
		s.ticker.Stop()
	}

	return nil
}

// OnPropertySet handles property set requests from the cloud
func (s *SmartSensor) OnPropertySet(property core.Property) error {
	log.Printf("[%s] Property set request: %s = %v", s.DeviceInfo.DeviceName, property.Name, property.Value)

	// In this example, we don't allow setting sensor values from cloud
	return fmt.Errorf("sensor values are read-only")
}

// OnServiceInvoke handles service invocation from the cloud
func (s *SmartSensor) OnServiceInvoke(service core.ServiceRequest) (core.ServiceResponse, error) {
	log.Printf("[%s] Service invoke: %s with params %v", s.DeviceInfo.DeviceName, service.Service, service.Params)

	response := core.ServiceResponse{
		ID:        service.ID,
		Timestamp: time.Now(),
	}

	switch service.Service {
	case "calibrate":
		offset := 0.0
		if val, ok := service.Params["offset"].(float64); ok {
			offset = val
		}
		s.temperature += offset
		response.Code = 0
		response.Data = map[string]interface{}{
			"message": fmt.Sprintf("Calibrated with offset %.2f", offset),
		}

	case "reset":
		s.temperature = 25.0
		s.humidity = 60.0
		response.Code = 0
		response.Data = map[string]interface{}{
			"message": "Sensor reset to default values",
		}

	default:
		response.Code = -1
		response.Message = fmt.Sprintf("Unknown service: %s", service.Service)
	}

	return response, nil
}

// Property getters
func (s *SmartSensor) getTemperature() interface{} {
	return s.temperature
}

func (s *SmartSensor) getHumidity() interface{} {
	return s.humidity
}

func (s *SmartSensor) getOnline() interface{} {
	return s.online
}

// Service handlers
func (s *SmartSensor) calibrate(params map[string]interface{}) (interface{}, error) {
	offset := 0.0
	if val, ok := params["offset"].(float64); ok {
		offset = val
	}

	s.temperature += offset

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Temperature calibrated by %.2f degrees", offset),
	}, nil
}

func (s *SmartSensor) reset(params map[string]interface{}) (interface{}, error) {
	s.temperature = 25.0
	s.humidity = 60.0

	return map[string]interface{}{
		"success": true,
		"message": "Sensor values reset to defaults",
	}, nil
}

// startSimulation starts generating random sensor data
func (s *SmartSensor) startSimulation() {
	s.ticker = time.NewTicker(10 * time.Second)

	go func() {
		for range s.ticker.C {
			// Generate random variations
			s.temperature += (rand.Float64() - 0.5) * 2 // ±1 degree
			s.humidity += (rand.Float64() - 0.5) * 5    // ±2.5%

			// Keep values in reasonable range
			if s.temperature < 15 {
				s.temperature = 15
			} else if s.temperature > 35 {
				s.temperature = 35
			}

			if s.humidity < 30 {
				s.humidity = 30
			} else if s.humidity > 90 {
				s.humidity = 90
			}

			// Report if online
			if s.online {
				s.reportStatus()
			}
		}
	}()
}

// reportStatus reports current sensor status to the platform
func (s *SmartSensor) reportStatus() {
	log.Printf("[%s] Reporting status: temp=%.2f°C, humidity=%.2f%%",
		s.DeviceInfo.DeviceName, s.temperature, s.humidity)

	err := s.framework.ReportProperties(map[string]interface{}{
		"temperature": s.temperature,
		"humidity":    s.humidity,
		"online":      s.online,
	})

	if err != nil {
		log.Printf("[%s] Failed to report properties: %v", s.DeviceInfo.DeviceName, err)
	}
}

func main() {
	// Set random seed
	rand.Seed(time.Now().UnixNano())

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

	// Create and register device
	sensor := NewSmartSensor(
		frameworkConfig.Device.ProductKey,
		frameworkConfig.Device.DeviceName,
		frameworkConfig.Device.DeviceSecret,
	)
	sensor.framework = framework

	if err := framework.RegisterDevice(sensor); err != nil {
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

	log.Println("Smart sensor demo started. Press Ctrl+C to exit.")
	log.Println("Connecting to IoT platform via MQTT...")

	// Wait for shutdown
	framework.WaitForShutdown()

	log.Println("Smart sensor demo stopped.")
}
