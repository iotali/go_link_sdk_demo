package dynreg

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/iot-go-sdk/pkg/auth"
	"github.com/iot-go-sdk/pkg/config"
	tlsutil "github.com/iot-go-sdk/pkg/tls"
)

type MQTTDynRegClient struct {
	config        *config.Config
	mqttClient    mqtt.Client
	logger        *log.Logger
	response      chan *MQTTDynRegResponse
	mutex         sync.Mutex
	skipPreRegist bool // Store skipPreRegist flag for auth type determination
}

type MQTTDynRegRequest struct {
	ProductKey      string `json:"productKey"`
	DeviceName      string `json:"deviceName"`
	Random          string `json:"random,omitempty"`
	Sign            string `json:"sign"`
	SignMethod      string `json:"signMethod"`
	SkipPreRegist   int    `json:"skipPreRegist,omitempty"`
}

type MQTTDynRegResponse struct {
	Code         int                    `json:"code"`
	Data         MQTTDynRegResponseData `json:"data"`
	Message      string                 `json:"message"`
	RequestId    string                 `json:"requestId"`
}

type MQTTDynRegResponseData struct {
	DeviceSecret string `json:"deviceSecret,omitempty"`
	ClientId     string `json:"clientId,omitempty"`
	Username     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
}

func NewMQTTDynRegClient(cfg *config.Config) *MQTTDynRegClient {
	return &MQTTDynRegClient{
		config:   cfg,
		logger:   log.Default(),
		response: make(chan *MQTTDynRegResponse, 1),
	}
}

func (c *MQTTDynRegClient) SetLogger(logger *log.Logger) {
	c.logger = logger
}

func (c *MQTTDynRegClient) Register(skipPreRegist bool, timeout time.Duration) (*MQTTDynRegResponseData, error) {
	if c.config.Device.ProductSecret == "" {
		return nil, fmt.Errorf("product secret is required for MQTT dynamic registration")
	}

	// Store skipPreRegist flag for use in connect()
	c.skipPreRegist = skipPreRegist

	if err := c.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %w", err)
	}
	defer c.disconnect()

	// In dynamic registration, server will automatically subscribe the client
	// to /ext/register/{productKey}/{deviceName} and send response
	// We just need to wait for the message to arrive
	
	// Wait for registration response
	// The server will automatically push the result once connected
	select {
	case resp := <-c.response:
		if resp.Code != 200 && resp.Code != 0 {  // Some servers may return 0 for success
			return nil, fmt.Errorf("dynamic registration failed: code=%d, message=%s", resp.Code, resp.Message)
		}
		return &resp.Data, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("dynamic registration timeout after %v", timeout)
	}
}

func (c *MQTTDynRegClient) connect() error {
	// Generate random number for dynamic registration (10 digits max)
	random := fmt.Sprintf("%d", time.Now().UnixNano()%10000000000)
	
	// Determine auth type based on skipPreRegist flag
	authType := "register"  // Whitelist mode (default)
	if c.skipPreRegist {
		authType = "regnwl"  // Non-whitelist mode
	}
	
	// Generate authentication credentials for dynamic registration
	// Format: deviceName.productKey|random={random},authType={authType},securemode=2,signmethod=hmacsha256|
	// Note: C SDK source shows order is deviceName first, then productKey
	clientID := fmt.Sprintf("%s.%s|random=%s,authType=%s,securemode=2,signmethod=hmacsha256|", 
		c.config.Device.DeviceName,
		c.config.Device.ProductKey,
		random,
		authType)
	
	// For dynamic registration, username is deviceName&productKey
	username := fmt.Sprintf("%s&%s", c.config.Device.DeviceName, c.config.Device.ProductKey)
	
	// Generate password using HMAC-SHA256
	// For dynamic registration: sign(content) where content = "deviceName{deviceName}productKey{productKey}random{random}"
	content := fmt.Sprintf("deviceName%sproductKey%srandom%s", 
		c.config.Device.DeviceName,
		c.config.Device.ProductKey,
		random)
	password := calculateHMACSHA256(content, c.config.Device.ProductSecret)
	
	c.logger.Printf("Dynamic registration auth type: %s", authType)
	c.logger.Printf("Dynamic registration random: %s", random)
	c.logger.Printf("Dynamic registration sign content: %s", content)
	c.logger.Printf("Dynamic registration product secret: %s", c.config.Device.ProductSecret)
	c.logger.Printf("Dynamic registration password: %s", password)
	
	opts := mqtt.NewClientOptions()
	
	broker := fmt.Sprintf("tcp://%s:%d", c.config.MQTT.Host, c.config.MQTT.Port)
	if c.config.MQTT.UseTLS {
		broker = fmt.Sprintf("ssl://%s:%d", c.config.MQTT.Host, c.config.MQTT.Port)
		
		tlsConfig := &tls.Config{
			InsecureSkipVerify: c.config.TLS.SkipVerify,
		}
		
		certPool, err := tlsutil.LoadCACert(c.config.TLS.CACert)
		if err != nil {
			return fmt.Errorf("failed to load CA certificate: %w", err)
		}
		tlsConfig.RootCAs = certPool
		
		opts.SetTLSConfig(tlsConfig)
	}
	
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetCleanSession(true)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetAutoReconnect(false)
	
	// Set default message handler to receive registration response
	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		c.logger.Printf("Received message on topic %s: %s", msg.Topic(), string(msg.Payload()))
		
		// Check if this is a registration response
		expectedTopic := fmt.Sprintf("/ext/register/%s/%s", c.config.Device.ProductKey, c.config.Device.DeviceName)
		if msg.Topic() == expectedTopic {
			c.messageHandler(client, msg)
		}
	})
	
	c.logger.Printf("Dynamic registration connecting with ClientID: %s", clientID)
	c.logger.Printf("Dynamic registration Username: %s", username)

	c.mqttClient = mqtt.NewClient(opts)
	
	token := c.mqttClient.Connect()
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	
	c.logger.Printf("Connected to MQTT broker for dynamic registration: %s", broker)
	return nil
}

func (c *MQTTDynRegClient) disconnect() {
	if c.mqttClient != nil && c.mqttClient.IsConnected() {
		c.mqttClient.Disconnect(250)
		c.logger.Println("Disconnected from MQTT broker")
	}
}

func (c *MQTTDynRegClient) subscribe(topic string) error {
	token := c.mqttClient.Subscribe(topic, 0, c.messageHandler)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	
	c.logger.Printf("Subscribed to topic: %s", topic)
	return nil
}

func (c *MQTTDynRegClient) publishRequest(skipPreRegist bool) error {
	random := fmt.Sprintf("%d", time.Now().UnixMilli())

	signature := auth.GenerateDynRegSignature(
		c.config.Device.ProductKey,
		c.config.Device.DeviceName,
		c.config.Device.ProductSecret,
		random,
	)

	skipPreRegistInt := 0
	if skipPreRegist {
		skipPreRegistInt = 1
	}

	reqData := MQTTDynRegRequest{
		ProductKey:    c.config.Device.ProductKey,
		DeviceName:    c.config.Device.DeviceName,
		Random:        random,
		Sign:          signature,
		SignMethod:    "hmacsha256",
		SkipPreRegist: skipPreRegistInt,
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	topic := fmt.Sprintf("/ext/register/%s/%s", c.config.Device.ProductKey, c.config.Device.DeviceName)
	token := c.mqttClient.Publish(topic, 0, false, jsonData)
	if token.Wait() && token.Error() != nil {
		return token.Error()
	}
	
	c.logger.Printf("Published registration request to topic: %s", topic)
	return nil
}

func (c *MQTTDynRegClient) messageHandler(client mqtt.Client, msg mqtt.Message) {
	c.logger.Printf("Processing registration response: %s", string(msg.Payload()))
	
	// First try to parse as direct response (like C SDK receives)
	// Format: {"deviceSecret":"xxx"} or {"clientId":"xxx","username":"xxx","password":"xxx"}
	var directResponse MQTTDynRegResponseData
	if err := json.Unmarshal(msg.Payload(), &directResponse); err == nil {
		// Direct response format
		response := &MQTTDynRegResponse{
			Code: 0,  // Success
			Data: directResponse,
		}
		
		select {
		case c.response <- response:
			c.logger.Printf("Registration successful, received deviceSecret: %s", directResponse.DeviceSecret)
		default:
			c.logger.Println("Response channel is full, dropping message")
		}
		return
	}
	
	// Try to parse as wrapped response
	var response MQTTDynRegResponse
	if err := json.Unmarshal(msg.Payload(), &response); err != nil {
		c.logger.Printf("Failed to unmarshal response: %v", err)
		return
	}
	
	select {
	case c.response <- &response:
	default:
		c.logger.Println("Response channel is full, dropping message")
	}
}

func calculateHMACSHA256(data, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	// Use uppercase hex to match C SDK format
	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
}