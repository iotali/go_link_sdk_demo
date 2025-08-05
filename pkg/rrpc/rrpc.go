package rrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"

	"github.com/iot-go-sdk/pkg/mqtt"
)

type RequestHandler func(requestId string, payload []byte) ([]byte, error)

type RRPCClient struct {
	mqttClient   *mqtt.Client
	productKey   string
	deviceName   string
	handlers     map[string]RequestHandler
	mutex        sync.RWMutex
	logger       *log.Logger
	requestIdReg *regexp.Regexp
}

type RRPCRequest struct {
	ID      string                 `json:"id"`
	Version string                 `json:"version"`
	Params  map[string]interface{} `json:"params"`
	Method  string                 `json:"method,omitempty"`
}

type RRPCResponse struct {
	ID      string                 `json:"id"`
	Version string                 `json:"version"`
	Code    int                    `json:"code,omitempty"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Message string                 `json:"message,omitempty"`
}

func NewRRPCClient(mqttClient *mqtt.Client, productKey, deviceName string) *RRPCClient {
	requestIdReg := regexp.MustCompile(`/sys/` + regexp.QuoteMeta(productKey) + `/` + regexp.QuoteMeta(deviceName) + `/rrpc/request/(.+)`)

	return &RRPCClient{
		mqttClient:   mqttClient,
		productKey:   productKey,
		deviceName:   deviceName,
		handlers:     make(map[string]RequestHandler),
		logger:       log.Default(),
		requestIdReg: requestIdReg,
	}
}

func (c *RRPCClient) SetLogger(logger *log.Logger) {
	c.logger = logger
}

func (c *RRPCClient) Start() error {
	if !c.mqttClient.IsConnected() {
		return fmt.Errorf("MQTT client is not connected")
	}

	requestTopic := fmt.Sprintf("/sys/%s/%s/rrpc/request/+", c.productKey, c.deviceName)
	return c.mqttClient.Subscribe(requestTopic, 0, c.handleRRPCRequest)
}

func (c *RRPCClient) Stop() error {
	requestTopic := fmt.Sprintf("/sys/%s/%s/rrpc/request/+", c.productKey, c.deviceName)
	return c.mqttClient.Unsubscribe(requestTopic)
}

func (c *RRPCClient) RegisterHandler(method string, handler RequestHandler) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.handlers[method] = handler
}

func (c *RRPCClient) UnregisterHandler(method string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.handlers, method)
}

func (c *RRPCClient) handleRRPCRequest(topic string, payload []byte) {
	c.logger.Printf("Received RRPC request on topic: %s, payload: %s", topic, string(payload))

	requestId := c.extractRequestId(topic)
	if requestId == "" {
		c.logger.Printf("Failed to extract request ID from topic: %s", topic)
		return
	}

	var request RRPCRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		c.logger.Printf("Failed to unmarshal RRPC request: %v", err)
		c.sendErrorResponse(requestId, 400, "Invalid JSON format")
		return
	}

	c.mutex.RLock()
	handler, exists := c.handlers[request.Method]
	c.mutex.RUnlock()

	if !exists {
		c.logger.Printf("No handler registered for method: %s", request.Method)
		c.sendErrorResponse(requestId, 404, fmt.Sprintf("Method '%s' not found", request.Method))
		return
	}

	responseData, err := handler(requestId, payload)
	if err != nil {
		c.logger.Printf("Handler returned error: %v", err)
		c.sendErrorResponse(requestId, 500, err.Error())
		return
	}

	c.sendSuccessResponse(requestId, responseData)
}

func (c *RRPCClient) extractRequestId(topic string) string {
	matches := c.requestIdReg.FindStringSubmatch(topic)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func (c *RRPCClient) sendSuccessResponse(requestId string, data []byte) {
	response := RRPCResponse{
		ID:      "1",
		Version: "1.0",
		Code:    200,
	}

	if data != nil && len(data) > 0 {
		var responseData map[string]interface{}
		if err := json.Unmarshal(data, &responseData); err == nil {
			response.Data = responseData
		} else {
			response.Data = map[string]interface{}{
				"result": string(data),
			}
		}
	}

	c.sendResponse(requestId, response)
}

func (c *RRPCClient) sendErrorResponse(requestId string, code int, message string) {
	response := RRPCResponse{
		ID:      "1",
		Version: "1.0",
		Code:    code,
		Message: message,
	}

	c.sendResponse(requestId, response)
}

func (c *RRPCClient) sendResponse(requestId string, response RRPCResponse) {
	responseTopic := fmt.Sprintf("/sys/%s/%s/rrpc/response/%s", c.productKey, c.deviceName, requestId)

	responseData, err := json.Marshal(response)
	if err != nil {
		c.logger.Printf("Failed to marshal RRPC response: %v", err)
		return
	}

	if err := c.mqttClient.Publish(responseTopic, responseData, 0, false); err != nil {
		c.logger.Printf("Failed to publish RRPC response: %v", err)
		return
	}

	c.logger.Printf("Sent RRPC response to topic: %s, payload: %s", responseTopic, string(responseData))
}

func (c *RRPCClient) Call(ctx context.Context, method string, params map[string]interface{}) (*RRPCResponse, error) {
	requestId := fmt.Sprintf("%d", time.Now().UnixNano())

	request := RRPCRequest{
		ID:      requestId,
		Version: "1.0",
		Method:  method,
		Params:  params,
	}

	requestData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RRPC request: %w", err)
	}

	requestTopic := fmt.Sprintf("/sys/%s/%s/rrpc/request/%s", c.productKey, c.deviceName, requestId)
	responseTopic := fmt.Sprintf("/sys/%s/%s/rrpc/response/%s", c.productKey, c.deviceName, requestId)

	responseChan := make(chan *RRPCResponse, 1)
	errorChan := make(chan error, 1)

	if err := c.mqttClient.Subscribe(responseTopic, 0, func(topic string, payload []byte) {
		var response RRPCResponse
		if err := json.Unmarshal(payload, &response); err != nil {
			errorChan <- fmt.Errorf("failed to unmarshal RRPC response: %w", err)
			return
		}
		responseChan <- &response
	}); err != nil {
		return nil, fmt.Errorf("failed to subscribe to response topic: %w", err)
	}

	defer c.mqttClient.Unsubscribe(responseTopic)

	if err := c.mqttClient.Publish(requestTopic, requestData, 0, false); err != nil {
		return nil, fmt.Errorf("failed to publish RRPC request: %w", err)
	}

	select {
	case response := <-responseChan:
		return response, nil
	case err := <-errorChan:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("RRPC call timeout: %w", ctx.Err())
	}
}

func DefaultLightSwitchHandler(requestId string, payload []byte) ([]byte, error) {
	response := map[string]interface{}{
		"LightSwitch": 0,
	}

	return json.Marshal(response)
}
