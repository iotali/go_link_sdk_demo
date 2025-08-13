package ota

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/iot-go-sdk/pkg/framework/core"
	"github.com/iot-go-sdk/pkg/framework/event"
	"github.com/iot-go-sdk/pkg/mqtt"
)

// PluginStatus represents the status of a plugin
type PluginStatus int

const (
	PluginStatusStopped PluginStatus = iota
	PluginStatusRunning
	PluginStatusError
)

// OTAPlugin implements OTA functionality as a framework plugin
type OTAPlugin struct {
	name          string
	version       string
	description   string
	status        PluginStatus
	framework     core.Framework
	mqttClient    *mqtt.Client
	managers      map[string]Manager
	deviceWrappers map[string]*DeviceWrapper
	mu            sync.RWMutex
	logger        *log.Logger
	autoUpdate    bool
	checkInterval time.Duration
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// NewOTAPlugin creates a new OTA plugin
func NewOTAPlugin() *OTAPlugin {
	return &OTAPlugin{
		name:           "ota",
		version:        "1.0.0",
		description:    "OTA firmware update plugin",
		status:         PluginStatusStopped,
		managers:       make(map[string]Manager),
		deviceWrappers: make(map[string]*DeviceWrapper),
		logger:         log.New(log.Writer(), "[OTA Plugin] ", log.LstdFlags),
		autoUpdate:     true,
		checkInterval:  5 * time.Minute,
		stopCh:         make(chan struct{}),
	}
}

// Name returns the plugin name
func (p *OTAPlugin) Name() string {
	return p.name
}

// Version returns the plugin version
func (p *OTAPlugin) Version() string {
	return p.version
}

// Description returns the plugin description
func (p *OTAPlugin) Description() string {
	return p.description
}

// GetStatus returns the plugin status
func (p *OTAPlugin) GetStatus() PluginStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// SetStatus sets the plugin status
func (p *OTAPlugin) SetStatus(status PluginStatus) {
	p.mu.Lock()
	p.status = status
	p.mu.Unlock()
}

// Init initializes the OTA plugin
func (p *OTAPlugin) Init(ctx context.Context, framework interface{}) error {
	fw, ok := framework.(core.Framework)
	if !ok {
		return fmt.Errorf("invalid framework type")
	}
	p.framework = fw
	p.logger.Println("Initializing OTA plugin")
	
	// Register event handlers
	p.registerEventHandlers()
	
	return nil
}

// Dependencies returns the plugin dependencies
func (p *OTAPlugin) Dependencies() []string {
	return []string{"mqtt"} // OTA plugin depends on MQTT plugin
}

// Configure configures the plugin
func (p *OTAPlugin) Configure(config map[string]interface{}) error {
	// Configure from map if needed
	if autoUpdate, ok := config["auto_update"].(bool); ok {
		p.autoUpdate = autoUpdate
	}
	if checkInterval, ok := config["check_interval"].(time.Duration); ok {
		p.checkInterval = checkInterval
	}
	return nil
}

// Start starts the OTA plugin
func (p *OTAPlugin) Start() error {
	p.logger.Println("Starting OTA plugin")
	
	// Get MQTT client from framework
	mqttPlugin, err := p.framework.GetPlugin("mqtt")
	if err != nil {
		return fmt.Errorf("MQTT plugin not found: %v", err)
	}
	
	// Type assert to get MQTT client
	type mqttClientProvider interface {
		GetMQTTClient() *mqtt.Client
	}
	
	provider, ok := mqttPlugin.(mqttClientProvider)
	if !ok {
		return fmt.Errorf("MQTT plugin does not provide GetMQTTClient method")
	}
	
	p.mqttClient = provider.GetMQTTClient()
	
	// Note: Device registration will be handled via events
	// since framework doesn't expose GetAllDevices directly
	
	// Don't start auto-update checker if no devices registered
	// It will be started when first device is registered
	
	p.SetStatus(PluginStatusRunning)
	return nil
}

// Stop stops the OTA plugin
func (p *OTAPlugin) Stop() error {
	p.logger.Println("Stopping OTA plugin")
	
	// Check if already stopped
	if p.GetStatus() == PluginStatusStopped {
		return nil
	}
	
	// Stop all managers first
	p.mu.Lock()
	for deviceID, manager := range p.managers {
		if manager != nil {
			if err := manager.Stop(); err != nil {
				p.logger.Printf("Failed to stop OTA manager for device %s: %v", deviceID, err)
			}
		}
	}
	p.managers = make(map[string]Manager)
	p.deviceWrappers = make(map[string]*DeviceWrapper)
	p.mu.Unlock()
	
	// Then signal stop to plugin goroutines
	if p.stopCh != nil {
		select {
		case <-p.stopCh:
			// Already closed
		default:
			close(p.stopCh)
		}
	}
	
	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// All goroutines finished
	case <-time.After(5 * time.Second):
		p.logger.Println("Warning: Timeout waiting for OTA plugin goroutines to stop")
	}
	
	p.SetStatus(PluginStatusStopped)
	return nil
}

// RegisterDevice registers a device for OTA management
func (p *OTAPlugin) RegisterDevice(dev core.Device) error {
	return p.createManagerForDevice(dev)
}

// UnregisterDevice unregisters a device from OTA management
func (p *OTAPlugin) UnregisterDevice(deviceID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if manager, exists := p.managers[deviceID]; exists {
		if err := manager.Stop(); err != nil {
			return err
		}
		delete(p.managers, deviceID)
		delete(p.deviceWrappers, deviceID)
	}
	
	return nil
}

// GetManager gets the OTA manager for a specific device
func (p *OTAPlugin) GetManager(deviceID string) Manager {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.managers[deviceID]
}

// SetAutoUpdate enables or disables auto-update for all devices
func (p *OTAPlugin) SetAutoUpdate(enabled bool) {
	p.mu.Lock()
	p.autoUpdate = enabled
	for _, manager := range p.managers {
		manager.SetAutoUpdate(enabled)
	}
	p.mu.Unlock()
}

// SetCheckInterval sets the update check interval
func (p *OTAPlugin) SetCheckInterval(interval time.Duration) {
	p.mu.Lock()
	p.checkInterval = interval
	p.mu.Unlock()
}

// createManagerForDevice creates an OTA manager for a device
func (p *OTAPlugin) createManagerForDevice(dev core.Device) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Create device wrapper
	wrapper := NewDeviceWrapper(dev)
	deviceID := wrapper.GetDeviceID()
	
	// Check if manager already exists
	if _, exists := p.managers[deviceID]; exists {
		return nil
	}
	
	// Store wrapper
	p.deviceWrappers[deviceID] = wrapper
	
	// Get device credentials
	productKey := wrapper.GetProductKey()
	deviceName := wrapper.GetDeviceName()
	
	// Create version provider wrapper
	versionProvider := &deviceVersionProvider{wrapper: wrapper}
	
	// Create OTA manager
	manager := NewManager(p.mqttClient, productKey, deviceName, versionProvider)
	
	// Set status callback to update device properties
	manager.SetStatusCallback(func(status Status, progress int32, message string) {
		p.updateDeviceOTAStatus(wrapper, status, progress, message)
	})
	
	// Set auto-update
	manager.SetAutoUpdate(p.autoUpdate)
	
	// Start manager
	if err := manager.Start(); err != nil {
		return fmt.Errorf("failed to start OTA manager: %v", err)
	}
	
	p.managers[deviceID] = manager
	p.logger.Printf("Created OTA manager for device %s", deviceID)
	
	// Start auto-update checker on first device registration
	if len(p.managers) == 1 && p.autoUpdate && p.GetStatus() == PluginStatusRunning {
		p.wg.Add(1)
		go p.autoUpdateLoop()
		p.logger.Println("Started auto-update checker")
	}
	
	return nil
}

// updateDeviceOTAStatus updates device OTA status properties
func (p *OTAPlugin) updateDeviceOTAStatus(wrapper *DeviceWrapper, status Status, progress int32, message string) {
	// Update OTA status property
	wrapper.SetProperty("ota_status", string(status))
	wrapper.SetProperty("ota_progress", progress)
	
	if message != "" {
		wrapper.SetProperty("ota_message", message)
	}
	
	// Update last update time when completed
	if status == StatusIdle {
		wrapper.SetProperty("last_update_time", time.Now().Format(time.RFC3339))
	}
	
	// Emit OTA status event
	evt := &event.Event{
		Type:      "ota.status_changed",
		Source:    p.name,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"device_id": wrapper.GetDeviceID(),
			"status":    status,
			"progress":  progress,
			"message":   message,
		},
	}
	p.framework.Emit(evt)
}

// registerEventHandlers registers event handlers
func (p *OTAPlugin) registerEventHandlers() {
	// Handle device registration
	p.framework.On("device.registered", func(evt *event.Event) error {
		if data, ok := evt.Data.(map[string]interface{}); ok {
			if deviceID, ok := data["device_id"].(string); ok {
				dev, err := p.framework.GetDevice(deviceID)
				if err == nil {
					if err := p.RegisterDevice(dev); err != nil {
						p.logger.Printf("Failed to register device %s for OTA: %v", deviceID, err)
					}
				}
			}
		}
		return nil
	})
	
	// Handle device unregistration
	p.framework.On("device.unregistered", func(evt *event.Event) error {
		if data, ok := evt.Data.(map[string]interface{}); ok {
			if deviceID, ok := data["device_id"].(string); ok {
				if err := p.UnregisterDevice(deviceID); err != nil {
					p.logger.Printf("Failed to unregister device %s from OTA: %v", deviceID, err)
				}
			}
		}
		return nil
	})
	
	// Handle OTA commands
	p.framework.On("ota.check_update", func(evt *event.Event) error {
		if data, ok := evt.Data.(map[string]interface{}); ok {
			if deviceID, ok := data["device_id"].(string); ok {
				if manager := p.GetManager(deviceID); manager != nil {
					go func() {
						if info, err := manager.CheckUpdate(); err == nil && info != nil {
							p.logger.Printf("Update available for device %s: %s", deviceID, info.Version)
						}
					}()
				}
			}
		}
		return nil
	})
	
	p.framework.On("ota.perform_update", func(evt *event.Event) error {
		if data, ok := evt.Data.(map[string]interface{}); ok {
			if deviceID, ok := data["device_id"].(string); ok {
				if manager := p.GetManager(deviceID); manager != nil {
					if info, ok := data["update_info"].(*UpdateInfo); ok {
						go func() {
							result, _ := manager.PerformUpdate(info)
							p.logger.Printf("Update result for device %s: %v", deviceID, result)
						}()
					}
				}
			}
		}
		return nil
	})
}

// autoUpdateLoop periodically checks for updates
func (p *OTAPlugin) autoUpdateLoop() {
	defer p.wg.Done()
	
	ticker := time.NewTicker(p.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			p.checkAllDevices()
		case <-p.stopCh:
			return
		}
	}
}

// checkAllDevices checks updates for all devices
func (p *OTAPlugin) checkAllDevices() {
	p.mu.RLock()
	managers := make(map[string]Manager)
	for k, v := range p.managers {
		managers[k] = v
	}
	p.mu.RUnlock()
	
	for deviceID, manager := range managers {
		if manager.GetStatus() == StatusIdle {
			if info, err := manager.CheckUpdate(); err == nil && info != nil {
				p.logger.Printf("Auto-update available for device %s: %s", deviceID, info.Version)
				if p.autoUpdate {
					go func(m Manager, i *UpdateInfo) {
						result, _ := m.PerformUpdate(i)
						if result.Success {
							p.logger.Printf("Auto-update successful for device %s", deviceID)
						} else {
							p.logger.Printf("Auto-update failed for device %s: %s", deviceID, result.Message)
						}
					}(manager, info)
				}
			}
		}
	}
}

// deviceVersionProvider wraps a device wrapper to provide version information
type deviceVersionProvider struct {
	wrapper *DeviceWrapper
}

func (p *deviceVersionProvider) GetVersion() string {
	if val := p.wrapper.GetProperty("firmware_version"); val != nil {
		if version, ok := val.(string); ok {
			return version
		}
	}
	return "1.0.0"
}

func (p *deviceVersionProvider) SetVersion(version string) error {
	return p.wrapper.SetProperty("firmware_version", version)
}

func (p *deviceVersionProvider) GetModule() string {
	if val := p.wrapper.GetProperty("firmware_module"); val != nil {
		if module, ok := val.(string); ok {
			return module
		}
	}
	return "default"
}

func (p *deviceVersionProvider) SetModule(module string) error {
	return p.wrapper.SetProperty("firmware_module", module)
}