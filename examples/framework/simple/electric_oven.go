package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/iot-go-sdk/pkg/framework/core"
	"github.com/iot-go-sdk/pkg/framework/event"
)

// ElectricOven represents a smart electric oven with temperature control
type ElectricOven struct {
	core.BaseDevice

	// Properties
	currentTemp      float64  // 当前温度
	targetTemp       float64  // 目标温度
	heaterStatus     bool     // 加热器状态
	timerSetting     int32    // 定时器设置（分钟）
	remainingTime    int32    // 剩余时间（分钟）
	doorStatus       bool     // 门状态 (true=open, false=closed)
	powerConsumption float64  // 功耗（瓦）
	operationMode    string   // 操作模式
	internalLight    bool     // 内部照明
	fanStatus        bool     // 风扇状态

	// Internal state
	isRunning    bool
	lastHeatTime time.Time
	mutex        sync.RWMutex

	// Framework reference
	framework core.Framework

	// Control channels
	stopCh   chan struct{}
	timerCh  chan struct{}
}

// NewElectricOven creates a new electric oven device
func NewElectricOven(productKey, deviceName, deviceSecret string) *ElectricOven {
	return &ElectricOven{
		BaseDevice: core.BaseDevice{
			DeviceInfo: core.DeviceInfo{
				ProductKey:   productKey,
				DeviceName:   deviceName,
				DeviceSecret: deviceSecret,
				Model:        "SmartOven-X1",
				Version:      "2.0.0",
			},
		},
		currentTemp:      25.0,  // Room temperature
		targetTemp:       0.0,
		heaterStatus:     false,
		timerSetting:     0,
		remainingTime:    0,
		doorStatus:       false, // Door closed
		powerConsumption: 0.0,
		operationMode:    "待机",
		internalLight:    false,
		fanStatus:        false,
		stopCh:           make(chan struct{}),
		timerCh:          make(chan struct{}, 1),
	}
}

// OnInitialize is called when the device is initialized
func (o *ElectricOven) OnInitialize(ctx context.Context) error {
	log.Printf("[%s] Initializing electric oven...", o.DeviceInfo.DeviceName)

	// Register properties (read-only)
	o.framework.RegisterProperty("current_temperature", o.getCurrentTemp, nil)
	o.framework.RegisterProperty("target_temperature", o.getTargetTemp, o.setTargetTemp)
	o.framework.RegisterProperty("heater_status", o.getHeaterStatus, nil)
	o.framework.RegisterProperty("timer_setting", o.getTimerSetting, nil)
	o.framework.RegisterProperty("remaining_time", o.getRemainingTime, nil)
	o.framework.RegisterProperty("door_status", o.getDoorStatus, nil)
	o.framework.RegisterProperty("power_consumption", o.getPowerConsumption, nil)
	o.framework.RegisterProperty("operation_mode", o.getOperationMode, nil)
	o.framework.RegisterProperty("internal_light", o.getInternalLight, o.setInternalLight)
	o.framework.RegisterProperty("fan_status", o.getFanStatus, nil)

	// Register services
	o.framework.RegisterService("set_temperature", o.setTemperatureService)
	o.framework.RegisterService("start_timer", o.startTimerService)
	o.framework.RegisterService("toggle_door", o.toggleDoorService)

	// Start simulation
	o.startSimulation()

	return nil
}

// OnConnect is called when the device connects to the platform
func (o *ElectricOven) OnConnect(ctx context.Context) error {
	log.Printf("[%s] Electric oven connected to IoT platform", o.DeviceInfo.DeviceName)
	
	// Report initial state
	o.reportFullStatus()
	
	return nil
}

// OnDisconnect is called when the device disconnects from the platform
func (o *ElectricOven) OnDisconnect(ctx context.Context) error {
	log.Printf("[%s] Electric oven disconnected from IoT platform", o.DeviceInfo.DeviceName)
	return nil
}

// OnDestroy is called when the device is being destroyed
func (o *ElectricOven) OnDestroy(ctx context.Context) error {
	log.Printf("[%s] Destroying electric oven...", o.DeviceInfo.DeviceName)
	
	// Stop simulation
	close(o.stopCh)
	
	return nil
}

// OnPropertySet handles property set requests from the cloud
func (o *ElectricOven) OnPropertySet(property core.Property) error {
	log.Printf("[%s] Property set request: %s = %v", o.DeviceInfo.DeviceName, property.Name, property.Value)
	
	switch property.Name {
	case "target_temperature":
		if val, ok := property.Value.(float64); ok {
			return o.setTargetTemp(val)
		}
	case "internal_light":
		if val, ok := property.Value.(bool); ok {
			return o.setInternalLight(val)
		}
	}
	
	return fmt.Errorf("property %s cannot be set or invalid value", property.Name)
}

// OnServiceInvoke handles service invocation from the cloud
func (o *ElectricOven) OnServiceInvoke(service core.ServiceRequest) (core.ServiceResponse, error) {
	log.Printf("[%s] Service invoke: %s with params %v", o.DeviceInfo.DeviceName, service.Service, service.Params)
	
	// Services are handled via registered handlers
	return core.ServiceResponse{
		ID:        service.ID,
		Code:      -1,
		Message:   "Service handled by framework",
		Timestamp: time.Now(),
	}, nil
}

// Property getters
func (o *ElectricOven) getCurrentTemp() interface{} {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.currentTemp
}

func (o *ElectricOven) getTargetTemp() interface{} {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.targetTemp
}

func (o *ElectricOven) getHeaterStatus() interface{} {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.heaterStatus
}

func (o *ElectricOven) getTimerSetting() interface{} {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.timerSetting
}

func (o *ElectricOven) getRemainingTime() interface{} {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.remainingTime
}

func (o *ElectricOven) getDoorStatus() interface{} {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.doorStatus
}

func (o *ElectricOven) getPowerConsumption() interface{} {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.powerConsumption
}

func (o *ElectricOven) getOperationMode() interface{} {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.operationMode
}

func (o *ElectricOven) getInternalLight() interface{} {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.internalLight
}

func (o *ElectricOven) getFanStatus() interface{} {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	return o.fanStatus
}

// Property setters
func (o *ElectricOven) setTargetTemp(value interface{}) error {
	temp, ok := value.(float64)
	if !ok {
		return fmt.Errorf("invalid temperature value")
	}
	
	if temp < 0 || temp > 300 {
		return fmt.Errorf("temperature out of range (0-300)")
	}
	
	o.mutex.Lock()
	o.targetTemp = temp
	if temp > 0 {
		o.isRunning = true
		o.operationMode = "加热中"
	} else {
		o.isRunning = false
		o.operationMode = "待机"
	}
	o.mutex.Unlock()
	
	log.Printf("[%s] Target temperature set to %.1f°C", o.DeviceInfo.DeviceName, temp)
	o.reportFullStatus()
	
	return nil
}

func (o *ElectricOven) setInternalLight(value interface{}) error {
	light, ok := value.(bool)
	if !ok {
		return fmt.Errorf("invalid light value")
	}
	
	o.mutex.Lock()
	o.internalLight = light
	o.mutex.Unlock()
	
	log.Printf("[%s] Internal light set to %v", o.DeviceInfo.DeviceName, light)
	o.reportFullStatus()
	
	return nil
}

// Service handlers
func (o *ElectricOven) setTemperatureService(params map[string]interface{}) (interface{}, error) {
	temp, ok := params["temperature"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid temperature parameter")
	}
	
	if err := o.setTargetTemp(temp); err != nil {
		return nil, err
	}
	
	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Temperature set to %.1f°C", temp),
	}, nil
}

func (o *ElectricOven) startTimerService(params map[string]interface{}) (interface{}, error) {
	timeMin, ok := params["time"]
	if !ok {
		return nil, fmt.Errorf("missing time parameter")
	}
	
	var minutes int32
	switch v := timeMin.(type) {
	case float64:
		minutes = int32(v)
	case int32:
		minutes = v
	case int:
		minutes = int32(v)
	default:
		return nil, fmt.Errorf("invalid time parameter type")
	}
	
	if minutes < 0 || minutes > 1440 {
		return nil, fmt.Errorf("time out of range (0-1440 minutes)")
	}
	
	o.mutex.Lock()
	o.timerSetting = minutes
	o.remainingTime = minutes
	o.mutex.Unlock()
	
	// Trigger timer processing
	select {
	case o.timerCh <- struct{}{}:
	default:
	}
	
	log.Printf("[%s] Timer started for %d minutes", o.DeviceInfo.DeviceName, minutes)
	o.reportFullStatus()
	
	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Timer set for %d minutes", minutes),
	}, nil
}

func (o *ElectricOven) toggleDoorService(params map[string]interface{}) (interface{}, error) {
	o.mutex.Lock()
	o.doorStatus = !o.doorStatus
	newStatus := o.doorStatus
	
	// If door is opened while heating, pause heating
	if o.doorStatus && o.isRunning {
		o.heaterStatus = false
		o.operationMode = "暂停（门开）"
	} else if !o.doorStatus && o.isRunning {
		o.operationMode = "加热中"
	}
	o.mutex.Unlock()
	
	statusStr := "closed"
	if newStatus {
		statusStr = "open"
	}
	
	log.Printf("[%s] Door toggled to %s", o.DeviceInfo.DeviceName, statusStr)
	o.reportFullStatus()
	
	return map[string]interface{}{
		"door_status": newStatus,
		"message":     fmt.Sprintf("Door is now %s", statusStr),
	}, nil
}

// startSimulation starts the oven simulation
func (o *ElectricOven) startSimulation() {
	// Temperature control loop
	go o.temperatureControlLoop()
	
	// Timer countdown loop
	go o.timerCountdownLoop()
	
	// Status reporting loop
	go o.statusReportingLoop()
}

// temperatureControlLoop simulates temperature changes and heater control
func (o *ElectricOven) temperatureControlLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-o.stopCh:
			return
		case <-ticker.C:
			o.updateTemperature()
		}
	}
}

// updateTemperature simulates temperature physics
func (o *ElectricOven) updateTemperature() {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	
	roomTemp := 25.0
	maxHeatingRate := 5.0  // degrees per update
	coolingRate := 2.0     // degrees per update
	
	// Don't heat if door is open
	if o.doorStatus {
		o.heaterStatus = false
		o.fanStatus = false
		coolingRate *= 2 // Cool faster with door open
	} else if o.isRunning && o.targetTemp > 0 {
		// Control logic
		if o.currentTemp < o.targetTemp - 5 {
			o.heaterStatus = true
			o.fanStatus = true
		} else if o.currentTemp > o.targetTemp + 5 {
			o.heaterStatus = false
			o.fanStatus = true // Keep fan on for even temperature
		}
	} else {
		o.heaterStatus = false
		o.fanStatus = false
	}
	
	// Update temperature based on heater state
	if o.heaterStatus {
		// Heating
		heatingPower := maxHeatingRate * (1 - o.currentTemp/400) // Slower heating at higher temps
		o.currentTemp += heatingPower
		o.powerConsumption = 2000 + 500*math.Sin(o.currentTemp/50) // Varying power
	} else {
		// Cooling towards room temperature
		if o.currentTemp > roomTemp {
			coolingAmount := coolingRate * ((o.currentTemp - roomTemp) / 100)
			o.currentTemp -= coolingAmount
			if o.currentTemp < roomTemp {
				o.currentTemp = roomTemp
			}
		}
		o.powerConsumption = 0
		if o.fanStatus {
			o.powerConsumption = 50 // Fan power
		}
		if o.internalLight {
			o.powerConsumption += 15 // Light power
		}
	}
	
	// Check for overheat alarm
	if o.currentTemp > 250 {
		o.triggerOverheatAlarm()
	}
	
	// Limit temperature
	if o.currentTemp > 300 {
		o.currentTemp = 300
	}
}

// timerCountdownLoop handles timer countdown
func (o *ElectricOven) timerCountdownLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-o.stopCh:
			return
		case <-o.timerCh:
			// Timer started or restarted
		case <-ticker.C:
			o.mutex.Lock()
			if o.remainingTime > 0 {
				o.remainingTime--
				if o.remainingTime == 0 {
					// Timer expired
					o.isRunning = false
					o.targetTemp = 0
					o.operationMode = "定时结束"
					o.timerSetting = 0
					log.Printf("[%s] Timer expired, stopping oven", o.DeviceInfo.DeviceName)
					
					// Report timer completion event
					o.mutex.Unlock()
					o.reportTimerComplete()
					o.reportFullStatus()
				} else {
					o.mutex.Unlock()
				}
			} else {
				o.mutex.Unlock()
			}
		}
	}
}

// statusReportingLoop periodically reports device status
func (o *ElectricOven) statusReportingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-o.stopCh:
			return
		case <-ticker.C:
			o.reportFullStatus()
		}
	}
}

// reportFullStatus reports all properties to the platform
func (o *ElectricOven) reportFullStatus() {
	o.mutex.RLock()
	status := map[string]interface{}{
		"current_temperature": o.currentTemp,
		"target_temperature":  o.targetTemp,
		"heater_status":       o.heaterStatus,
		"timer_setting":       o.timerSetting,
		"remaining_time":      o.remainingTime,
		"door_status":         o.doorStatus,
		"power_consumption":   o.powerConsumption,
		"operation_mode":      o.operationMode,
		"internal_light":      o.internalLight,
		"fan_status":          o.fanStatus,
	}
	o.mutex.RUnlock()
	
	log.Printf("[%s] Reporting status: temp=%.1f°C, target=%.1f°C, heater=%v, mode=%s",
		o.DeviceInfo.DeviceName, status["current_temperature"], 
		status["target_temperature"], status["heater_status"], status["operation_mode"])
	
	if err := o.framework.ReportProperties(status); err != nil {
		log.Printf("[%s] Failed to report properties: %v", o.DeviceInfo.DeviceName, err)
	}
}

// triggerOverheatAlarm triggers an overheat alarm event
func (o *ElectricOven) triggerOverheatAlarm() {
	log.Printf("[%s] ALARM: Temperature too high! %.1f°C", o.DeviceInfo.DeviceName, o.currentTemp)
	
	// Create overheat event
	eventData := map[string]interface{}{
		"event_type": "overheat_alarm",
		"data": map[string]interface{}{
			"current_temperature": o.currentTemp,
		},
		"timestamp": time.Now().Unix(),
	}
	
	// Emit event through framework
	evt := event.NewEvent(event.EventCustom, o.DeviceInfo.DeviceName, eventData)
	if err := o.framework.Emit(evt); err != nil {
		log.Printf("[%s] Failed to emit overheat event: %v", o.DeviceInfo.DeviceName, err)
	}
	
	// Auto-shutdown for safety
	o.isRunning = false
	o.targetTemp = 0
	o.operationMode = "安全停机"
}

// reportTimerComplete reports timer completion event
func (o *ElectricOven) reportTimerComplete() {
	log.Printf("[%s] Timer completed", o.DeviceInfo.DeviceName)
	
	// Create timer complete event
	eventData := map[string]interface{}{
		"event_type": "timer_complete",
		"data": map[string]interface{}{
			"message": "Timer has completed",
		},
		"timestamp": time.Now().Unix(),
	}
	
	// Emit event through framework
	evt := event.NewEvent(event.EventCustom, o.DeviceInfo.DeviceName, eventData)
	if err := o.framework.Emit(evt); err != nil {
		log.Printf("[%s] Failed to emit timer complete event: %v", o.DeviceInfo.DeviceName, err)
	}
}

// SetFramework sets the framework reference
func (o *ElectricOven) SetFramework(framework core.Framework) {
	o.framework = framework
}