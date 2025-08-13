package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/iot-go-sdk/pkg/framework/event"
	"github.com/iot-go-sdk/pkg/framework/plugin"
)

// Framework is the main IoT framework interface
type Framework interface {
	// Lifecycle management
	Initialize(config Config) error
	Start() error
	Stop() error
	WaitForShutdown()

	// Device management
	RegisterDevice(device Device) error
	UnregisterDevice(deviceID string) error
	GetDevice(deviceID string) (Device, error)

	// Plugin management
	LoadPlugin(plugin plugin.Plugin) error
	UnloadPlugin(name string) error
	GetPlugin(name string) (plugin.Plugin, error)

	// Event management
	On(eventType event.EventType, handler event.Handler) error
	Emit(event *event.Event) error

	// Property management
	RegisterProperty(name string, getter func() interface{}, setter func(interface{}) error) error
	ReportProperty(name string, value interface{}) error
	ReportProperties(properties map[string]interface{}) error
	// Event management
	ReportEvent(eventName string, data map[string]interface{}) error

	// Service management
	RegisterService(name string, handler func(params map[string]interface{}) (interface{}, error)) error

	// Status
	GetState() LifecycleState
	GetConnectionState() ConnectionState
}

// IoTFramework is the concrete implementation of the Framework interface
type IoTFramework struct {
	// Configuration
	config Config

	// Core components
	eventBus     *event.Bus
	pluginMgr    *plugin.Manager
	devices      map[string]Device
	devicesMutex sync.RWMutex

	// Properties and services
	properties      map[string]*propertyHandler
	services        map[string]serviceHandler
	propertiesMutex sync.RWMutex
	servicesMutex   sync.RWMutex

	// State management
	state           LifecycleState
	connectionState ConnectionState
	stateMutex      sync.RWMutex

	// Lifecycle
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	shutdownCh chan os.Signal

	// Logging
	logger *log.Logger
}

type propertyHandler struct {
	getter func() interface{}
	setter func(interface{}) error
	mode   string
}

type serviceHandler func(params map[string]interface{}) (interface{}, error)

// New creates a new IoT framework instance
func New(config Config) Framework {
	return &IoTFramework{
		config:          config,
		devices:         make(map[string]Device),
		properties:      make(map[string]*propertyHandler),
		services:        make(map[string]serviceHandler),
		state:           LifecycleUninitialized,
		connectionState: StateDisconnected,
		shutdownCh:      make(chan os.Signal, 1),
		logger:          log.New(os.Stdout, "[Framework] ", log.LstdFlags),
	}
}

// Initialize initializes the framework
func (f *IoTFramework) Initialize(config Config) error {
	f.stateMutex.Lock()
	if f.state != LifecycleUninitialized {
		f.stateMutex.Unlock()
		return fmt.Errorf("framework already initialized")
	}
	f.state = LifecycleInitializing
	f.stateMutex.Unlock()

	f.logger.Println("Initializing framework...")

	// Update configuration
	f.config = config

	// Create context
	f.ctx, f.cancel = context.WithCancel(context.Background())

	// Initialize event bus
	workerCount := config.Advanced.WorkerCount
	if workerCount == 0 {
		workerCount = 10
	}
	f.eventBus = event.NewBus(workerCount)
	f.eventBus.SetLogger(log.New(os.Stdout, "[EventBus] ", log.LstdFlags))

	// Initialize plugin manager
	f.pluginMgr = plugin.NewManager()
	f.pluginMgr.SetLogger(log.New(os.Stdout, "[PluginMgr] ", log.LstdFlags))

	// Register internal event handlers
	f.registerInternalHandlers()

	// Setup signal handling
	signal.Notify(f.shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	f.stateMutex.Lock()
	f.state = LifecycleInitialized
	f.stateMutex.Unlock()

	f.logger.Println("Framework initialized successfully")
	return nil
}

// Start starts the framework
func (f *IoTFramework) Start() error {
	f.stateMutex.Lock()
	if f.state != LifecycleInitialized {
		f.stateMutex.Unlock()
		return fmt.Errorf("framework must be initialized before starting")
	}
	f.state = LifecycleStarting
	f.stateMutex.Unlock()

	f.logger.Println("Starting framework...")

	// Start event bus
	if err := f.eventBus.Start(); err != nil {
		return fmt.Errorf("failed to start event bus: %w", err)
	}

	// Initialize all loaded plugins first
	if err := f.pluginMgr.InitAll(f.ctx, f); err != nil {
		return fmt.Errorf("failed to initialize plugins: %w", err)
	}

	// Start all loaded plugins
	if err := f.pluginMgr.StartAll(); err != nil {
		return fmt.Errorf("failed to start plugins: %w", err)
	}

	// Initialize all devices
	f.devicesMutex.RLock()
	devices := make([]Device, 0, len(f.devices))
	for _, device := range f.devices {
		devices = append(devices, device)
	}
	f.devicesMutex.RUnlock()

	for _, device := range devices {
		if err := device.OnInitialize(f.ctx); err != nil {
			f.logger.Printf("Failed to initialize device %v: %v", device.GetDeviceInfo().DeviceName, err)
		}
	}

	// Emit system ready event
	f.Emit(event.NewEvent(event.EventReady, "framework", nil))

	f.stateMutex.Lock()
	f.state = LifecycleStarted
	f.stateMutex.Unlock()

	f.logger.Println("Framework started successfully")
	return nil
}

// Stop stops the framework
func (f *IoTFramework) Stop() error {
	f.stateMutex.Lock()
	if f.state != LifecycleStarted {
		f.stateMutex.Unlock()
		return fmt.Errorf("framework is not running")
	}
	f.state = LifecycleStopping
	f.stateMutex.Unlock()

	f.logger.Println("Stopping framework...")

	// Destroy all devices
	f.devicesMutex.RLock()
	devices := make([]Device, 0, len(f.devices))
	for _, device := range f.devices {
		devices = append(devices, device)
	}
	f.devicesMutex.RUnlock()

	for _, device := range devices {
		if err := device.OnDestroy(f.ctx); err != nil {
			f.logger.Printf("Failed to destroy device %v: %v", device.GetDeviceInfo().DeviceName, err)
		}
	}

	// Stop all plugins
	if err := f.pluginMgr.StopAll(); err != nil {
		f.logger.Printf("Error stopping plugins: %v", err)
	}

	// Stop event bus
	if err := f.eventBus.Stop(); err != nil {
		f.logger.Printf("Error stopping event bus: %v", err)
	}

	// Cancel context
	f.cancel()

	// Wait for goroutines to finish
	f.wg.Wait()

	f.stateMutex.Lock()
	f.state = LifecycleStopped
	f.stateMutex.Unlock()

	f.logger.Println("Framework stopped")
	return nil
}

// WaitForShutdown waits for shutdown signal
func (f *IoTFramework) WaitForShutdown() {
	f.logger.Println("Waiting for shutdown signal...")
	sig := <-f.shutdownCh
	f.logger.Printf("Shutdown signal received: %v", sig)
	if err := f.Stop(); err != nil {
		f.logger.Printf("Error during stop: %v", err)
	}
}

// RegisterDevice registers a device with the framework
func (f *IoTFramework) RegisterDevice(device Device) error {
	if device == nil {
		return fmt.Errorf("device cannot be nil")
	}

	info := device.GetDeviceInfo()
	deviceID := fmt.Sprintf("%s.%s", info.ProductKey, info.DeviceName)

	f.devicesMutex.Lock()
	defer f.devicesMutex.Unlock()

	if _, exists := f.devices[deviceID]; exists {
		return fmt.Errorf("device %s already registered", deviceID)
	}

	f.devices[deviceID] = device
	f.logger.Printf("Registered device: %s", deviceID)

	// If framework is already running, initialize the device
	if f.GetState() == LifecycleStarted {
		go func() {
			if err := device.OnInitialize(f.ctx); err != nil {
				f.logger.Printf("Failed to initialize device %s: %v", deviceID, err)
			}
		}()
	}

	return nil
}

// UnregisterDevice unregisters a device from the framework
func (f *IoTFramework) UnregisterDevice(deviceID string) error {
	f.devicesMutex.Lock()
	defer f.devicesMutex.Unlock()

	device, exists := f.devices[deviceID]
	if !exists {
		return fmt.Errorf("device %s not found", deviceID)
	}

	// Call destroy callback
	if err := device.OnDestroy(f.ctx); err != nil {
		f.logger.Printf("Error destroying device %s: %v", deviceID, err)
	}

	delete(f.devices, deviceID)
	f.logger.Printf("Unregistered device: %s", deviceID)

	return nil
}

// GetDevice gets a device by ID
func (f *IoTFramework) GetDevice(deviceID string) (Device, error) {
	f.devicesMutex.RLock()
	defer f.devicesMutex.RUnlock()

	device, exists := f.devices[deviceID]
	if !exists {
		return nil, fmt.Errorf("device %s not found", deviceID)
	}

	return device, nil
}

// LoadPlugin loads a plugin into the framework
func (f *IoTFramework) LoadPlugin(plugin plugin.Plugin) error {
	return f.pluginMgr.Register(plugin)
}

// UnloadPlugin unloads a plugin from the framework
func (f *IoTFramework) UnloadPlugin(name string) error {
	return f.pluginMgr.Unregister(name)
}

// GetPlugin gets a plugin by name
func (f *IoTFramework) GetPlugin(name string) (plugin.Plugin, error) {
	return f.pluginMgr.Get(name)
}

// On registers an event handler
func (f *IoTFramework) On(eventType event.EventType, handler event.Handler) error {
	return f.eventBus.Subscribe(eventType, handler)
}

// Emit emits an event
func (f *IoTFramework) Emit(evt *event.Event) error {
	return f.eventBus.Publish(evt)
}

// RegisterProperty registers a property
func (f *IoTFramework) RegisterProperty(name string, getter func() interface{}, setter func(interface{}) error) error {
	f.propertiesMutex.Lock()
	defer f.propertiesMutex.Unlock()

	mode := "r"
	if setter != nil {
		mode = "rw"
	}

	f.properties[name] = &propertyHandler{
		getter: getter,
		setter: setter,
		mode:   mode,
	}

	f.logger.Printf("Registered property: %s (mode: %s)", name, mode)
	return nil
}

// ReportProperty reports a single property to the cloud
func (f *IoTFramework) ReportProperty(name string, value interface{}) error {
	return f.ReportProperties(map[string]interface{}{name: value})
}

// ReportProperties reports multiple properties to the cloud
func (f *IoTFramework) ReportProperties(properties map[string]interface{}) error {
	// Emit property report event
	evt := event.NewEvent(event.EventPropertyReport, "framework", properties)
	return f.eventBus.Publish(evt)
}

// ReportEvent reports a device business event to the cloud
func (f *IoTFramework) ReportEvent(eventName string, data map[string]interface{}) error {
	payload := map[string]interface{}{
		"event_type": eventName,
		"data":       data,
		"timestamp":  time.Now().Unix(),
	}
	evt := event.NewEvent(event.EventEventReport, "framework", payload)
	return f.eventBus.Publish(evt)
}

// RegisterService registers a service handler
func (f *IoTFramework) RegisterService(name string, handler func(params map[string]interface{}) (interface{}, error)) error {
	f.servicesMutex.Lock()
	defer f.servicesMutex.Unlock()

	f.services[name] = handler
	f.logger.Printf("Registered service: %s", name)
	return nil
}

// GetState returns the current lifecycle state
func (f *IoTFramework) GetState() LifecycleState {
	f.stateMutex.RLock()
	defer f.stateMutex.RUnlock()
	return f.state
}

// GetConnectionState returns the current connection state
func (f *IoTFramework) GetConnectionState() ConnectionState {
	f.stateMutex.RLock()
	defer f.stateMutex.RUnlock()
	return f.connectionState
}

// registerInternalHandlers registers internal event handlers
func (f *IoTFramework) registerInternalHandlers() {
	// Handle connection events
	f.eventBus.Subscribe(event.EventConnected, func(evt *event.Event) error {
		f.stateMutex.Lock()
		f.connectionState = StateConnected
		f.stateMutex.Unlock()

		// Notify all devices
		f.devicesMutex.RLock()
		devices := make([]Device, 0, len(f.devices))
		for _, device := range f.devices {
			devices = append(devices, device)
		}
		f.devicesMutex.RUnlock()

		for _, device := range devices {
			go device.OnConnect(f.ctx)
		}

		return nil
	})

	f.eventBus.Subscribe(event.EventDisconnected, func(evt *event.Event) error {
		f.stateMutex.Lock()
		f.connectionState = StateDisconnected
		f.stateMutex.Unlock()

		// Notify all devices
		f.devicesMutex.RLock()
		devices := make([]Device, 0, len(f.devices))
		for _, device := range f.devices {
			devices = append(devices, device)
		}
		f.devicesMutex.RUnlock()

		for _, device := range devices {
			go device.OnDisconnect(f.ctx)
		}

		return nil
	})

	// Handle property set events
	f.eventBus.Subscribe(event.EventPropertySet, func(evt *event.Event) error {
		props, ok := evt.Data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid property data")
		}

		// Process each property
		for name, value := range props {
			f.propertiesMutex.RLock()
			handler, exists := f.properties[name]
			f.propertiesMutex.RUnlock()

			if exists && handler.setter != nil {
				if err := handler.setter(value); err != nil {
					f.logger.Printf("Error setting property %s: %v", name, err)
				}
			}

			// Notify devices
			f.devicesMutex.RLock()
			devices := make([]Device, 0, len(f.devices))
			for _, device := range f.devices {
				devices = append(devices, device)
			}
			f.devicesMutex.RUnlock()

			for _, device := range devices {
				device.OnPropertySet(Property{
					Name:  name,
					Value: value,
				})
			}
		}

		return nil
	})

	// Handle service call events
	f.eventBus.Subscribe(event.EventServiceCall, func(evt *event.Event) error {
		req, ok := evt.Data.(ServiceRequest)
		if !ok {
			return fmt.Errorf("invalid service request")
		}

		f.servicesMutex.RLock()
		handler, exists := f.services[req.Service]
		f.servicesMutex.RUnlock()

		if !exists {
			// Try devices
			f.devicesMutex.RLock()
			devices := make([]Device, 0, len(f.devices))
			for _, device := range f.devices {
				devices = append(devices, device)
			}
			f.devicesMutex.RUnlock()

			for _, device := range devices {
				resp, err := device.OnServiceInvoke(req)
				if err == nil {
					// Emit response event
					f.Emit(event.NewEvent(event.EventServiceResponse, "framework", resp))
					return nil
				}
			}

			return fmt.Errorf("service %s not found", req.Service)
		}

		// Execute service handler
		result, err := handler(req.Params)

		resp := ServiceResponse{
			ID:        req.ID,
			Timestamp: time.Now(),
		}

		if err != nil {
			resp.Code = -1
			resp.Message = err.Error()
		} else {
			resp.Code = 0
			resp.Data = result
		}

		// Emit response event
		f.Emit(event.NewEvent(event.EventServiceResponse, "framework", resp))

		return nil
	})
}
