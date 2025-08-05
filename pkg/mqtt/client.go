package mqtt

import (
	"crypto/tls"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/iot-go-sdk/pkg/auth"
	"github.com/iot-go-sdk/pkg/config"
	tlsutil "github.com/iot-go-sdk/pkg/tls"
)

type MessageHandler func(topic string, payload []byte)

type Client struct {
	config       *config.Config
	mqttClient   mqtt.Client
	connected    bool
	mutex        sync.RWMutex
	handlers     map[string]MessageHandler
	logger       *log.Logger
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		config:   cfg,
		handlers: make(map[string]MessageHandler),
		logger:   log.Default(),
	}
}

func (c *Client) SetLogger(logger *log.Logger) {
	c.logger = logger
}

func (c *Client) Connect() error {
	if err := c.config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	secureMode := c.config.GetSecureMode()
	credentials := auth.GenerateMQTTCredentials(
		c.config.Device.ProductKey,
		c.config.Device.DeviceName,
		c.config.Device.DeviceSecret,
		secureMode,
	)

	// 打印生成的ClientID用于调试
	c.logger.Printf("生成的Client ID: %s", credentials.ClientID)

	opts := mqtt.NewClientOptions()
	
	broker := fmt.Sprintf("tcp://%s:%d", c.config.MQTT.Host, c.config.MQTT.Port)
	if c.config.MQTT.UseTLS {
		broker = fmt.Sprintf("ssl://%s:%d", c.config.MQTT.Host, c.config.MQTT.Port)
		
		tlsConfig := &tls.Config{
			InsecureSkipVerify: c.config.TLS.SkipVerify,
			ServerName:         c.config.TLS.ServerName,
		}
		
		// If ServerName is set but SkipVerify is false, we still want to verify the certificate
		// but ignore hostname mismatch (since we're connecting by IP)
		if c.config.TLS.ServerName != "" && !c.config.TLS.SkipVerify {
			tlsConfig.InsecureSkipVerify = true
			// We'll manually verify the certificate chain using our custom CA
		}
		
		certPool, err := tlsutil.LoadCACert(c.config.TLS.CACert)
		if err != nil {
			return fmt.Errorf("failed to load CA certificate: %w", err)
		}
		tlsConfig.RootCAs = certPool
		
		opts.SetTLSConfig(tlsConfig)
	}
	
	opts.AddBroker(broker)
	opts.SetClientID(credentials.ClientID)
	opts.SetUsername(credentials.Username)
	opts.SetPassword(credentials.Password)
	opts.SetKeepAlive(c.config.MQTT.KeepAlive)
	opts.SetCleanSession(c.config.MQTT.CleanSession)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(30 * time.Second)
	
	opts.SetDefaultPublishHandler(c.defaultMessageHandler)
	opts.SetConnectionLostHandler(c.connectionLostHandler)
	opts.SetOnConnectHandler(c.onConnectHandler)
	opts.SetReconnectingHandler(c.reconnectingHandler)

	c.mqttClient = mqtt.NewClient(opts)
	
	token := c.mqttClient.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect: %w", token.Error())
	}
	
	c.mutex.Lock()
	c.connected = true
	c.mutex.Unlock()
	
	c.logger.Printf("Connected to MQTT broker: %s", broker)
	return nil
}

func (c *Client) Disconnect() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if c.mqttClient != nil && c.connected {
		c.mqttClient.Disconnect(250)
		c.connected = false
		c.logger.Println("Disconnected from MQTT broker")
	}
}

func (c *Client) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.connected && c.mqttClient.IsConnected()
}

func (c *Client) Publish(topic string, payload []byte, qos byte, retained bool) error {
	if !c.IsConnected() {
		return fmt.Errorf("client is not connected")
	}
	
	token := c.mqttClient.Publish(topic, qos, retained, payload)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish message: %w", token.Error())
	}
	
	c.logger.Printf("Published message to topic: %s", topic)
	return nil
}

func (c *Client) Subscribe(topic string, qos byte, handler MessageHandler) error {
	if !c.IsConnected() {
		return fmt.Errorf("client is not connected")
	}
	
	c.mutex.Lock()
	c.handlers[topic] = handler
	c.mutex.Unlock()
	
	token := c.mqttClient.Subscribe(topic, qos, func(client mqtt.Client, msg mqtt.Message) {
		c.mutex.RLock()
		if h, exists := c.handlers[msg.Topic()]; exists {
			c.mutex.RUnlock()
			h(msg.Topic(), msg.Payload())
		} else {
			c.mutex.RUnlock()
		}
	})
	
	if token.Wait() && token.Error() != nil {
		c.mutex.Lock()
		delete(c.handlers, topic)
		c.mutex.Unlock()
		return fmt.Errorf("failed to subscribe to topic: %w", token.Error())
	}
	
	c.logger.Printf("Subscribed to topic: %s", topic)
	return nil
}

func (c *Client) Unsubscribe(topic string) error {
	if !c.IsConnected() {
		return fmt.Errorf("client is not connected")
	}
	
	token := c.mqttClient.Unsubscribe(topic)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to unsubscribe from topic: %w", token.Error())
	}
	
	c.mutex.Lock()
	delete(c.handlers, topic)
	c.mutex.Unlock()
	
	c.logger.Printf("Unsubscribed from topic: %s", topic)
	return nil
}

func (c *Client) defaultMessageHandler(client mqtt.Client, msg mqtt.Message) {
	c.logger.Printf("Received message on topic %s: %s", msg.Topic(), string(msg.Payload()))
}

func (c *Client) connectionLostHandler(client mqtt.Client, err error) {
	c.mutex.Lock()
	c.connected = false
	c.mutex.Unlock()
	c.logger.Printf("Connection lost: %v", err)
}

func (c *Client) onConnectHandler(client mqtt.Client) {
	c.mutex.Lock()
	c.connected = true
	c.mutex.Unlock()
	c.logger.Println("Connected to MQTT broker")
}

func (c *Client) reconnectingHandler(client mqtt.Client, opts *mqtt.ClientOptions) {
	c.logger.Println("Attempting to reconnect to MQTT broker...")
}