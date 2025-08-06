package core

import (
	"time"
)

// DeviceInfo contains device identification information
type DeviceInfo struct {
	ProductKey   string                 `json:"productKey"`
	DeviceName   string                 `json:"deviceName"`
	DeviceSecret string                 `json:"deviceSecret,omitempty"`
	Model        string                 `json:"model,omitempty"`
	Version      string                 `json:"version,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Property represents a device property
type Property struct {
	Name     string      `json:"name"`
	Value    interface{} `json:"value"`
	Mode     string      `json:"mode"` // "r", "w", "rw"
	DataType string      `json:"dataType"`
	Unit     string      `json:"unit,omitempty"`
	Min      interface{} `json:"min,omitempty"`
	Max      interface{} `json:"max,omitempty"`
	Step     interface{} `json:"step,omitempty"`
}

// Service represents a device service
type Service struct {
	Name        string                 `json:"name"`
	Identifier  string                 `json:"identifier"`
	InputParams map[string]interface{} `json:"inputParams,omitempty"`
	OutputType  string                 `json:"outputType,omitempty"`
}

// ServiceRequest represents a service invocation request
type ServiceRequest struct {
	ID         string                 `json:"id"`
	Service    string                 `json:"service"`
	Params     map[string]interface{} `json:"params"`
	Timestamp  time.Time              `json:"timestamp"`
}

// ServiceResponse represents a service invocation response
type ServiceResponse struct {
	ID        string                 `json:"id"`
	Code      int                    `json:"code"`
	Data      interface{}            `json:"data,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// DeviceEvent represents a device-generated event
type DeviceEvent struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// OTATask represents an OTA upgrade task
type OTATask struct {
	Version      string    `json:"version"`
	Module       string    `json:"module,omitempty"`
	URL          string    `json:"url"`
	Size         int64     `json:"size"`
	MD5          string    `json:"md5"`
	Description  string    `json:"description,omitempty"`
	ForceUpgrade bool      `json:"forceUpgrade"`
	Timestamp    time.Time `json:"timestamp"`
}

// ConnectionState represents the connection state
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateReconnecting
	StateError
)

func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// Config represents the framework configuration
type Config struct {
	Device   DeviceConfig   `json:"device"`
	MQTT     MQTTConfig     `json:"mqtt"`
	Features FeatureConfig  `json:"features"`
	Logging  LoggingConfig  `json:"logging"`
	Advanced AdvancedConfig `json:"advanced"`
}

// DeviceConfig contains device configuration
type DeviceConfig struct {
	ProductKey    string `json:"productKey"`
	DeviceName    string `json:"deviceName"`
	DeviceSecret  string `json:"deviceSecret,omitempty"`
	ProductSecret string `json:"productSecret,omitempty"`
	Region        string `json:"region,omitempty"`
}

// MQTTConfig contains MQTT configuration
type MQTTConfig struct {
	Host          string        `json:"host"`
	Port          int           `json:"port"`
	UseTLS        bool          `json:"useTLS"`
	KeepAlive     int           `json:"keepAlive"`
	CleanSession  bool          `json:"cleanSession"`
	AutoReconnect bool          `json:"autoReconnect"`
	ReconnectMax  int           `json:"reconnectMax"`
	Timeout       time.Duration `json:"timeout"`
}

// FeatureConfig contains feature toggles
type FeatureConfig struct {
	EnableOTA    bool `json:"enableOTA"`
	EnableShadow bool `json:"enableShadow"`
	EnableRules  bool `json:"enableRules"`
	EnableMetrics bool `json:"enableMetrics"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	Output     string `json:"output"`
	MaxSize    int    `json:"maxSize"`
	MaxBackups int    `json:"maxBackups"`
	MaxAge     int    `json:"maxAge"`
}

// AdvancedConfig contains advanced configuration
type AdvancedConfig struct {
	WorkerCount      int           `json:"workerCount"`
	EventBufferSize  int           `json:"eventBufferSize"`
	RequestTimeout   time.Duration `json:"requestTimeout"`
	PropertyCacheTTL time.Duration `json:"propertyCacheTime"`
}

// LifecycleState represents the lifecycle state
type LifecycleState int

const (
	LifecycleUninitialized LifecycleState = iota
	LifecycleInitializing
	LifecycleInitialized
	LifecycleStarting
	LifecycleStarted
	LifecycleStopping
	LifecycleStopped
	LifecycleError
)

func (s LifecycleState) String() string {
	switch s {
	case LifecycleUninitialized:
		return "uninitialized"
	case LifecycleInitializing:
		return "initializing"
	case LifecycleInitialized:
		return "initialized"
	case LifecycleStarting:
		return "starting"
	case LifecycleStarted:
		return "started"
	case LifecycleStopping:
		return "stopping"
	case LifecycleStopped:
		return "stopped"
	case LifecycleError:
		return "error"
	default:
		return "unknown"
	}
}

// Context keys for passing values
type contextKey string

const (
	ContextKeyDeviceID contextKey = "deviceID"
	ContextKeyTraceID  contextKey = "traceID"
	ContextKeyUserData contextKey = "userData"
)