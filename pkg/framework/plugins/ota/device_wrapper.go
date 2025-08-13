package ota

import (
	"github.com/iot-go-sdk/pkg/framework/core"
)

// DeviceWrapper wraps a core.Device to provide additional OTA functionality
type DeviceWrapper struct {
	device     core.Device
	properties map[string]interface{}
}

// NewDeviceWrapper creates a new device wrapper
func NewDeviceWrapper(device core.Device) *DeviceWrapper {
	return &DeviceWrapper{
		device:     device,
		properties: make(map[string]interface{}),
	}
}

// GetDeviceID returns the device ID (ProductKey.DeviceName)
func (w *DeviceWrapper) GetDeviceID() string {
	info := w.device.GetDeviceInfo()
	return info.ProductKey + "." + info.DeviceName
}

// GetProductKey returns the product key
func (w *DeviceWrapper) GetProductKey() string {
	return w.device.GetDeviceInfo().ProductKey
}

// GetDeviceName returns the device name
func (w *DeviceWrapper) GetDeviceName() string {
	return w.device.GetDeviceInfo().DeviceName
}

// GetProperty gets a property value
func (w *DeviceWrapper) GetProperty(name string) interface{} {
	// First try to get from device
	if val, err := w.device.OnPropertyGet(name); err == nil && val != nil {
		return val
	}
	// Fallback to local storage
	return w.properties[name]
}

// SetProperty sets a property value
func (w *DeviceWrapper) SetProperty(name string, value interface{}) error {
	// Store locally
	w.properties[name] = value
	
	// Try to set on device
	prop := core.Property{
		Name:  name,
		Value: value,
	}
	return w.device.OnPropertySet(prop)
}

// GetDevice returns the underlying device
func (w *DeviceWrapper) GetDevice() core.Device {
	return w.device
}