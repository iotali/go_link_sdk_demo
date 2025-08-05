package dynreg

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/iot-go-sdk/pkg/auth"
	"github.com/iot-go-sdk/pkg/config"
	tlsutil "github.com/iot-go-sdk/pkg/tls"
)

type MQTTDynRegClient struct {
	config     *config.Config
	mqttClient mqtt.Client
	logger     *log.Logger
	response   chan *MQTTDynRegResponse
	mutex      sync.Mutex
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

	if err := c.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %w", err)
	}
	defer c.disconnect()

	responseTopic := fmt.Sprintf("/ext/register/%s/%s", c.config.Device.ProductKey, c.config.Device.DeviceName)
	if err := c.subscribe(responseTopic); err != nil {
		return nil, fmt.Errorf("failed to subscribe to response topic: %w", err)
	}

	if err := c.publishRequest(skipPreRegist); err != nil {
		return nil, fmt.Errorf("failed to publish registration request: %w", err)
	}

	select {
	case resp := <-c.response:
		if resp.Code != 200 {
			return nil, fmt.Errorf("dynamic registration failed: code=%d, message=%s", resp.Code, resp.Message)
		}
		return &resp.Data, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("dynamic registration timeout after %v", timeout)
	}
}

func (c *MQTTDynRegClient) connect() error {
	clientID := fmt.Sprintf("%s.%s", c.config.Device.ProductKey, c.config.Device.DeviceName)
	
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
	opts.SetCleanSession(true)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetAutoReconnect(false)

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
	c.logger.Printf("Received message on topic %s: %s", msg.Topic(), string(msg.Payload()))
	
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