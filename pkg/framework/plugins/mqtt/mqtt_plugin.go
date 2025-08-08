package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/iot-go-sdk/pkg/config"
	"github.com/iot-go-sdk/pkg/framework/core"
	"github.com/iot-go-sdk/pkg/framework/event"
	"github.com/iot-go-sdk/pkg/framework/plugin"
	"github.com/iot-go-sdk/pkg/mqtt"
	"github.com/iot-go-sdk/pkg/rrpc"
)

// MQTTPlugin provides MQTT connectivity for the framework
type MQTTPlugin struct {
	plugin.BasePlugin

	client     *mqtt.Client
	rrpcClient *rrpc.RRPCClient
	config     *config.Config
	framework  core.Framework
	logger     *log.Logger

	// Topic mappings
	propertySetTopic         string
	propertySetReplyTopic    string
	propertyReportTopic      string
	propertyReportReplyTopic string
	eventReportTopic         string
	eventReportReplyTopic    string
	serviceCallTopic         string
	serviceReplyTopic        string
}

// NewMQTTPlugin creates a new MQTT plugin
func NewMQTTPlugin(cfg *config.Config) *MQTTPlugin {
	return &MQTTPlugin{
		BasePlugin: *plugin.NewBasePlugin(
			"mqtt",
			"1.0.0",
			"MQTT connectivity plugin for IoT framework",
		),
		config: cfg,
		logger: log.Default(),
	}
}

// Init initializes the plugin
func (p *MQTTPlugin) Init(ctx context.Context, framework interface{}) error {
	p.framework = framework.(core.Framework)

	// Create MQTT client with the existing SDK implementation
	p.client = mqtt.NewClient(p.config)

	// Setup topic names using Thing Model topics with $ prefix
	pk := p.config.Device.ProductKey
	dn := p.config.Device.DeviceName

	// Property topics - 属性上报使用 $SYS/xxx/property/post
	p.propertyReportTopic = fmt.Sprintf("$SYS/%s/%s/property/post", pk, dn)
	p.propertyReportReplyTopic = fmt.Sprintf("$SYS/%s/%s/property/post/reply", pk, dn)
	p.propertySetTopic = fmt.Sprintf("$SYS/%s/%s/property/set", pk, dn)
	p.propertySetReplyTopic = fmt.Sprintf("$SYS/%s/%s/property/set/reply", pk, dn)

	// Event topics - 事件上报使用 event/post
	p.eventReportTopic = fmt.Sprintf("$SYS/%s/%s/event/post", pk, dn)
	p.eventReportReplyTopic = fmt.Sprintf("$SYS/%s/%s/event/post/reply", pk, dn)

	// Service topics
	p.serviceCallTopic = fmt.Sprintf("$SYS/%s/%s/service/+/invoke", pk, dn)
	p.serviceReplyTopic = fmt.Sprintf("$SYS/%s/%s/service/+/invoke/reply", pk, dn)

	// Register event handlers
	p.registerEventHandlers()

	p.logger.Printf("[MQTT Plugin] Initialized for device %s.%s", pk, dn)
	return nil
}

// Start starts the plugin
func (p *MQTTPlugin) Start() error {
	p.logger.Println("[MQTT Plugin] Starting...")

	// Connect to MQTT broker
	if err := p.client.Connect(); err != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", err)
	}

	p.logger.Printf("[MQTT Plugin] Connected to MQTT broker at %s:%d", p.config.MQTT.Host, p.config.MQTT.Port)

	// Initialize and start RRPC client
	p.rrpcClient = rrpc.NewRRPCClient(p.client, p.config.Device.ProductKey, p.config.Device.DeviceName)
	p.rrpcClient.SetLogger(p.logger)
	
	// Register RRPC handlers from framework
	p.registerRRPCHandlers()
	
	if err := p.rrpcClient.Start(); err != nil {
		p.logger.Printf("[MQTT Plugin] Warning: Failed to start RRPC client: %v", err)
		// Continue without RRPC support
	} else {
		p.logger.Println("[MQTT Plugin] RRPC client started successfully")
	}

	// Subscribe to topics
	if err := p.subscribeToTopics(); err != nil {
		p.client.Disconnect()
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	// Emit connected event
	p.framework.Emit(event.NewEvent(event.EventConnected, "mqtt", nil))

	return nil
}

// Stop stops the plugin
func (p *MQTTPlugin) Stop() error {
	p.logger.Println("[MQTT Plugin] Stopping...")

	// Stop RRPC client
	if p.rrpcClient != nil {
		p.rrpcClient.Stop()
		p.logger.Println("[MQTT Plugin] RRPC client stopped")
	}

	// Emit disconnected event
	p.framework.Emit(event.NewEvent(event.EventDisconnected, "mqtt", nil))

	// Disconnect from MQTT broker
	if p.client != nil {
		p.client.Disconnect()
	}

	p.logger.Println("[MQTT Plugin] Stopped")
	return nil
}

// registerEventHandlers registers handlers for framework events
func (p *MQTTPlugin) registerEventHandlers() {
	// Handle property report events
	p.framework.On(event.EventPropertyReport, func(evt *event.Event) error {
		properties, ok := evt.Data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid property data")
		}

		return p.reportProperties(properties)
	})

	// Handle service response events
	p.framework.On(event.EventServiceResponse, func(evt *event.Event) error {
		response, ok := evt.Data.(core.ServiceResponse)
		if !ok {
			return fmt.Errorf("invalid service response data")
		}

		return p.sendServiceResponse(response)
	})

	// Handle explicit event report from framework
	p.framework.On(event.EventEventReport, func(evt *event.Event) error {
		eventData, ok := evt.Data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid event data")
		}
		return p.reportEvent(eventData)
	})

	// Backward compatibility: still handle custom events carrying `event_type`
	p.framework.On(event.EventCustom, func(evt *event.Event) error {
		eventData, ok := evt.Data.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid event data")
		}
		if _, ok := eventData["event_type"].(string); ok {
			return p.reportEvent(eventData)
		}
		return nil
	})
}

// subscribeToTopics subscribes to all necessary MQTT topics
func (p *MQTTPlugin) subscribeToTopics() error {
	// For topics with $, we need to handle them differently
	// Some brokers don't allow subscribing to $ topics directly

	// Try subscribing to property set topic
	if err := p.client.Subscribe(p.propertySetTopic, 0, p.handlePropertySet); err != nil {
		p.logger.Printf("[MQTT Plugin] Warning: Could not subscribe to %s: %v", p.propertySetTopic, err)
		// Try alternative format without $
		altTopic := fmt.Sprintf("/sys/%s/%s/thing/service/property/set", p.config.Device.ProductKey, p.config.Device.DeviceName)
		if err := p.client.Subscribe(altTopic, 0, p.handlePropertySet); err != nil {
			p.logger.Printf("[MQTT Plugin] Warning: Could not subscribe to alternative topic: %v", err)
		} else {
			p.logger.Printf("[MQTT Plugin] Subscribed to alternative topic: %s", altTopic)
		}
	}

	// Try subscribing to service call topics
	if err := p.client.Subscribe(p.serviceCallTopic, 0, p.handleServiceCall); err != nil {
		p.logger.Printf("[MQTT Plugin] Warning: Could not subscribe to %s: %v", p.serviceCallTopic, err)
		// Try alternative format
		altTopic := fmt.Sprintf("/sys/%s/%s/thing/service/+", p.config.Device.ProductKey, p.config.Device.DeviceName)
		if err := p.client.Subscribe(altTopic, 0, p.handleServiceCall); err != nil {
			p.logger.Printf("[MQTT Plugin] Warning: Could not subscribe to alternative service topic: %v", err)
		} else {
			p.logger.Printf("[MQTT Plugin] Subscribed to alternative service topic: %s", altTopic)
		}
	}

	// Skip reply topics for now as they may not be critical
	p.logger.Printf("[MQTT Plugin] Topic subscription completed")
	return nil
}

// handlePropertySet handles property set messages from the cloud
func (p *MQTTPlugin) handlePropertySet(topic string, payload []byte) {
	p.logger.Printf("[MQTT Plugin] Property set message: %s", string(payload))

	var msg struct {
		ID     string                 `json:"id"`
		Params map[string]interface{} `json:"params"`
	}

	if err := json.Unmarshal(payload, &msg); err != nil {
		p.logger.Printf("[MQTT Plugin] Failed to parse property set message: %v", err)
		return
	}

	// Emit property set event
	evt := event.NewEvent(event.EventPropertySet, "mqtt", msg.Params)
	evt.WithMetadata("messageId", msg.ID)

	if err := p.framework.Emit(evt); err != nil {
		p.logger.Printf("[MQTT Plugin] Failed to emit property set event: %v", err)
	}

	// Send reply to property set
	reply := map[string]interface{}{
		"id":   msg.ID,
		"code": 200,
		"data": map[string]interface{}{},
	}

	replyData, _ := json.Marshal(reply)

	if err := p.client.Publish(p.propertySetReplyTopic, replyData, 0, false); err != nil {
		p.logger.Printf("[MQTT Plugin] Failed to send property set reply: %v", err)
	}
}

// handleServiceCall handles service call messages from the cloud
func (p *MQTTPlugin) handleServiceCall(topic string, payload []byte) {
	// Skip reply topics
	if strings.Contains(topic, "_reply") {
		return
	}

	p.logger.Printf("[MQTT Plugin] Service call message on topic %s: %s", topic, string(payload))

	// Extract service name from topic
	// Topic format: $SYS/{ProductKey}/{DeviceName}/service/{ServiceName}/invoke
	parts := strings.Split(topic, "/")
	if len(parts) < 6 {
		p.logger.Printf("[MQTT Plugin] Invalid service topic: %s", topic)
		return
	}
	serviceName := parts[4] // Service name is at index 4, not 5

	var msg struct {
		ID     string                 `json:"id"`
		Params map[string]interface{} `json:"params"`
	}

	if err := json.Unmarshal(payload, &msg); err != nil {
		p.logger.Printf("[MQTT Plugin] Failed to parse service call message: %v", err)
		return
	}

	// Create service request
	request := core.ServiceRequest{
		ID:        msg.ID,
		Service:   serviceName,
		Params:    msg.Params,
		Timestamp: time.Now(),
	}

	// Emit service call event
	evt := event.NewEvent(event.EventServiceCall, "mqtt", request)

	if err := p.framework.Emit(evt); err != nil {
		p.logger.Printf("[MQTT Plugin] Failed to emit service call event: %v", err)
	}
}

// reportProperties reports properties to the cloud
func (p *MQTTPlugin) reportProperties(properties map[string]interface{}) error {
	// Convert properties to Thing Model format with value and timestamp
	timestamp := time.Now().Unix()
	params := make(map[string]interface{})

	for key, value := range properties {
		params[key] = map[string]interface{}{
			"value": fmt.Sprintf("%v", value), // Convert to string as per spec
			"time":  timestamp,
		}
	}

	// Create property report message in Thing Model format
	msg := map[string]interface{}{
		"id":      fmt.Sprintf("%d", timestamp),
		"version": "1.0",
		"params":  params,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal property report: %w", err)
	}

	// Publish to property report topic
	if err := p.client.Publish(p.propertyReportTopic, data, 0, false); err != nil {
		return fmt.Errorf("failed to publish property report: %w", err)
	}

	p.logger.Printf("[MQTT Plugin] Reported properties to %s: %s", p.propertyReportTopic, string(data))
	return nil
}

// sendServiceResponse sends a service response to the cloud
func (p *MQTTPlugin) sendServiceResponse(response core.ServiceResponse) error {
	// Create service response message
	msg := map[string]interface{}{
		"id":   response.ID,
		"code": response.Code,
		"data": response.Data,
	}

	if response.Message != "" {
		msg["message"] = response.Message
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal service response: %w", err)
	}

	// Determine reply topic (this is simplified, actual implementation would track the original service)
	// For now, we'll use a generic reply topic
	replyTopic := fmt.Sprintf("/sys/%s/%s/thing/service/property/set_reply",
		p.config.Device.ProductKey, p.config.Device.DeviceName)

	// Publish to service reply topic
	if err := p.client.Publish(replyTopic, data, 0, false); err != nil {
		return fmt.Errorf("failed to publish service response: %w", err)
	}

	p.logger.Printf("[MQTT Plugin] Sent service response: %v", msg)
	return nil
}

// handlePropertyReportReply handles property report reply from cloud
func (p *MQTTPlugin) handlePropertyReportReply(topic string, payload []byte) {
	p.logger.Printf("[MQTT Plugin] Property report reply: %s", string(payload))

	// Parse reply to check if cloud accepted the property report
	var reply struct {
		ID   string `json:"id"`
		Code int    `json:"code"`
		Msg  string `json:"msg,omitempty"`
	}

	if err := json.Unmarshal(payload, &reply); err != nil {
		p.logger.Printf("[MQTT Plugin] Failed to parse property report reply: %v", err)
		return
	}

	if reply.Code != 200 {
		p.logger.Printf("[MQTT Plugin] Property report failed with code %d: %s", reply.Code, reply.Msg)
	}
}

// handleEventReportReply handles event report reply from cloud
func (p *MQTTPlugin) handleEventReportReply(topic string, payload []byte) {
	p.logger.Printf("[MQTT Plugin] Event report reply: %s", string(payload))

	// Parse reply to check if cloud accepted the event
	var reply struct {
		ID   string `json:"id"`
		Code int    `json:"code"`
		Msg  string `json:"msg,omitempty"`
	}

	if err := json.Unmarshal(payload, &reply); err != nil {
		p.logger.Printf("[MQTT Plugin] Failed to parse event report reply: %v", err)
		return
	}

	if reply.Code != 200 {
		p.logger.Printf("[MQTT Plugin] Event report failed with code %d: %s", reply.Code, reply.Msg)
	}
}

// reportEvent reports an event to the cloud
func (p *MQTTPlugin) reportEvent(eventData map[string]interface{}) error {
	eventType, _ := eventData["event_type"].(string)

	// Create event message in Thing Model format
	msg := map[string]interface{}{
		"id":      fmt.Sprintf("%d", time.Now().Unix()),
		"version": "1.0",
		"params": map[string]interface{}{
			"eventType": eventType,
			"value":     eventData["data"],
			"time":      eventData["timestamp"],
		},
		"method": fmt.Sprintf("thing.event.%s.post", eventType),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish to event report topic
	if err := p.client.Publish(p.eventReportTopic, data, 0, false); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	p.logger.Printf("[MQTT Plugin] Reported event %s to %s: %s", eventType, p.eventReportTopic, string(data))
	return nil
}

// GetClient returns the underlying MQTT client (for advanced usage)
func (p *MQTTPlugin) GetClient() *mqtt.Client {
	return p.client
}

// RegisterRRPCHandler registers a handler for RRPC requests
func (p *MQTTPlugin) RegisterRRPCHandler(method string, handler func(requestId string, payload []byte) ([]byte, error)) {
	if p.rrpcClient != nil {
		p.rrpcClient.RegisterHandler(method, handler)
		p.logger.Printf("[MQTT Plugin] Registered RRPC handler for method: %s", method)
	}
}

// registerRRPCHandlers registers framework-level RRPC handlers
func (p *MQTTPlugin) registerRRPCHandlers() {
	// Register a default handler that routes RRPC requests to framework services
	p.rrpcClient.RegisterHandler("InvokeService", func(requestId string, payload []byte) ([]byte, error) {
		// Parse the request
		var request struct {
			Service string                 `json:"service"`
			Params  map[string]interface{} `json:"params"`
		}
		
		if err := json.Unmarshal(payload, &request); err != nil {
			return nil, fmt.Errorf("invalid request format: %w", err)
		}
		
		// Create a service request and emit it to the framework
		serviceReq := core.ServiceRequest{
			ID:        requestId,
			Service:   request.Service,
			Params:    request.Params,
			Timestamp: time.Now(),
		}
		
		// Emit service call event
		evt := event.NewEvent(event.EventServiceCall, "rrpc", serviceReq)
		if err := p.framework.Emit(evt); err != nil {
			return nil, fmt.Errorf("service invocation failed: %w", err)
		}
		
		// For now, return a success response
		// In a real implementation, we'd wait for the service response
		response := map[string]interface{}{
			"code":    0,
			"message": "Service invoked successfully",
		}
		
		return json.Marshal(response)
	})
	
	// Register a handler to get device status
	p.rrpcClient.RegisterHandler("GetDeviceStatus", func(requestId string, payload []byte) ([]byte, error) {
		// Collect all current property values
		status := map[string]interface{}{
			"status":    "online",
			"timestamp": time.Now().Unix(),
		}
		
		return json.Marshal(status)
	})
}
