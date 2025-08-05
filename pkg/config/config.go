package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type DeviceConfig struct {
	ProductKey    string
	DeviceName    string
	DeviceSecret  string
	ProductSecret string
}

type MQTTConfig struct {
	Host         string
	Port         int
	UseTLS       bool
	KeepAlive    time.Duration
	ClientID     string
	Username     string
	Password     string
	CleanSession bool
	SecureMode   string
}

type TLSConfig struct {
	CACert     string
	ClientCert string
	ClientKey  string
	SkipVerify bool
}

type Config struct {
	Device DeviceConfig
	MQTT   MQTTConfig
	TLS    TLSConfig
}

func NewConfig() *Config {
	return &Config{
		MQTT: MQTTConfig{
			Host:         "localhost",
			Port:         1883,
			UseTLS:       false,
			KeepAlive:    60 * time.Second,
			CleanSession: true,
		},
		TLS: TLSConfig{
			SkipVerify: false,
		},
	}
}

func (c *Config) LoadFromEnv() error {
	if val := os.Getenv("IOT_PRODUCT_KEY"); val != "" {
		c.Device.ProductKey = val
	}
	if val := os.Getenv("IOT_DEVICE_NAME"); val != "" {
		c.Device.DeviceName = val
	}
	if val := os.Getenv("IOT_DEVICE_SECRET"); val != "" {
		c.Device.DeviceSecret = val
	}
	if val := os.Getenv("IOT_PRODUCT_SECRET"); val != "" {
		c.Device.ProductSecret = val
	}

	if val := os.Getenv("IOT_MQTT_HOST"); val != "" {
		c.MQTT.Host = val
	}
	if val := os.Getenv("IOT_MQTT_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			c.MQTT.Port = port
		}
	}
	if val := os.Getenv("IOT_MQTT_USE_TLS"); val != "" {
		if useTLS, err := strconv.ParseBool(val); err == nil {
			c.MQTT.UseTLS = useTLS
		}
	}
	if val := os.Getenv("IOT_MQTT_KEEPALIVE"); val != "" {
		if keepAlive, err := strconv.Atoi(val); err == nil {
			c.MQTT.KeepAlive = time.Duration(keepAlive) * time.Second
		}
	}

	if val := os.Getenv("IOT_TLS_CA_CERT"); val != "" {
		c.TLS.CACert = val
	}
	if val := os.Getenv("IOT_TLS_SKIP_VERIFY"); val != "" {
		if skipVerify, err := strconv.ParseBool(val); err == nil {
			c.TLS.SkipVerify = skipVerify
		}
	}
	if val := os.Getenv("IOT_MQTT_SECURE_MODE"); val != "" {
		c.MQTT.SecureMode = val
	}

	return nil
}

func (c *Config) Validate() error {
	if c.Device.ProductKey == "" {
		return fmt.Errorf("product key is required")
	}
	if c.Device.DeviceName == "" {
		return fmt.Errorf("device name is required")
	}
	if c.Device.DeviceSecret == "" && c.Device.ProductSecret == "" {
		return fmt.Errorf("either device secret or product secret is required")
	}
	if c.MQTT.Host == "" {
		return fmt.Errorf("MQTT host is required")
	}
	if c.MQTT.Port <= 0 || c.MQTT.Port > 65535 {
		return fmt.Errorf("MQTT port must be between 1 and 65535")
	}
	return nil
}

func (c *Config) GenerateClientID() string {
	if c.MQTT.ClientID != "" {
		return c.MQTT.ClientID
	}
	return fmt.Sprintf("%s.%s", c.Device.ProductKey, c.Device.DeviceName)
}

func (c *Config) GetSecureMode() string {
	if c.MQTT.SecureMode != "" {
		return c.MQTT.SecureMode
	}
	
	if c.MQTT.UseTLS {
		return "2"
	}
	
	return "3"
}