package core

import (
	"context"
)

// Device interface represents an IoT device in the framework
type Device interface {
	// Device identification
	GetDeviceInfo() DeviceInfo
	
	// Lifecycle callbacks
	OnInitialize(ctx context.Context) error
	OnConnect(ctx context.Context) error
	OnDisconnect(ctx context.Context) error
	OnDestroy(ctx context.Context) error
	
	// Property handling
	OnPropertySet(property Property) error
	OnPropertyGet(name string) (interface{}, error)
	
	// Service handling
	OnServiceInvoke(service ServiceRequest) (ServiceResponse, error)
	
	// Event handling
	OnEventReceive(event DeviceEvent) error
	
	// OTA handling (optional, can return nil if not supported)
	OnOTANotify(task OTATask) error
}

// BaseDevice provides a default implementation of the Device interface
// Users can embed this struct to avoid implementing all methods
type BaseDevice struct {
	DeviceInfo DeviceInfo
}

// GetDeviceInfo returns device information
func (d *BaseDevice) GetDeviceInfo() DeviceInfo {
	return d.DeviceInfo
}

// OnInitialize is called when the device is initialized
func (d *BaseDevice) OnInitialize(ctx context.Context) error {
	// Default implementation does nothing
	return nil
}

// OnConnect is called when the device connects to the platform
func (d *BaseDevice) OnConnect(ctx context.Context) error {
	// Default implementation does nothing
	return nil
}

// OnDisconnect is called when the device disconnects from the platform
func (d *BaseDevice) OnDisconnect(ctx context.Context) error {
	// Default implementation does nothing
	return nil
}

// OnDestroy is called when the device is being destroyed
func (d *BaseDevice) OnDestroy(ctx context.Context) error {
	// Default implementation does nothing
	return nil
}

// OnPropertySet is called when a property is set from the cloud
func (d *BaseDevice) OnPropertySet(property Property) error {
	// Default implementation does nothing
	return nil
}

// OnPropertyGet is called when a property value is requested
func (d *BaseDevice) OnPropertyGet(name string) (interface{}, error) {
	// Default implementation returns nil
	return nil, nil
}

// OnServiceInvoke is called when a service is invoked from the cloud
func (d *BaseDevice) OnServiceInvoke(service ServiceRequest) (ServiceResponse, error) {
	// Default implementation returns success with no data
	return ServiceResponse{
		ID:   service.ID,
		Code: 0,
	}, nil
}

// OnEventReceive is called when an event is received
func (d *BaseDevice) OnEventReceive(event DeviceEvent) error {
	// Default implementation does nothing
	return nil
}

// OnOTANotify is called when an OTA notification is received
func (d *BaseDevice) OnOTANotify(task OTATask) error {
	// Default implementation does nothing
	return nil
}