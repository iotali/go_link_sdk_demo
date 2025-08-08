package event

import (
	"context"
	"fmt"
	"time"
)

// EventType defines the type of event
type EventType string

// System events
const (
	EventConnected    EventType = "system.connected"
	EventDisconnected EventType = "system.disconnected"
	EventError        EventType = "system.error"
	EventReady        EventType = "system.ready"
)

// Property events
const (
	EventPropertySet    EventType = "property.set"
	EventPropertyGet    EventType = "property.get"
	EventPropertyReport EventType = "property.report"
)

// Business events
const (
	// EventEventReport is emitted when a device business event should be reported to cloud
	EventEventReport EventType = "event.report"
)

// Service events
const (
	EventServiceCall     EventType = "service.call"
	EventServiceResponse EventType = "service.response"
)

// OTA events
const (
	EventOTANotify   EventType = "ota.notify"
	EventOTAProgress EventType = "ota.progress"
	EventOTAComplete EventType = "ota.complete"
	EventOTAFailed   EventType = "ota.failed"
)

// Device events
const (
	EventDeviceOnline  EventType = "device.online"
	EventDeviceOffline EventType = "device.offline"
	EventDeviceUpdate  EventType = "device.update"
)

// Custom events
const (
	EventCustom EventType = "custom"
)

// Event represents a system or business event
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Data      interface{}            `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Context   context.Context        `json:"-"`
}

// NewEvent creates a new event
func NewEvent(eventType EventType, source string, data interface{}) *Event {
	return &Event{
		ID:        generateEventID(),
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now(),
		Data:      data,
		Metadata:  make(map[string]interface{}),
		Context:   context.Background(),
	}
}

// WithContext sets the context for the event
func (e *Event) WithContext(ctx context.Context) *Event {
	e.Context = ctx
	return e
}

// WithMetadata adds metadata to the event
func (e *Event) WithMetadata(key string, value interface{}) *Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// Handler is a function that handles events
type Handler func(event *Event) error

// HandlerInfo contains handler information
type HandlerInfo struct {
	Handler  Handler
	Priority int
	Async    bool
}
