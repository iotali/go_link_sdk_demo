package dynreg

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/iot-go-sdk/pkg/auth"
	"github.com/iot-go-sdk/pkg/config"
)

type HTTPDynRegClient struct {
	config     *config.Config
	httpClient *http.Client
}

type DynRegRequest struct {
	ProductKey   string `json:"productKey"`
	DeviceName   string `json:"deviceName"`
	Random       string `json:"random,omitempty"`
	Sign         string `json:"sign"`
	SignMethod   string `json:"signMethod"`
}

type DynRegResponse struct {
	Code         int    `json:"code"`
	Data         Data   `json:"data"`
	Message      string `json:"message"`
	RequestId    string `json:"requestId"`
}

type Data struct {
	DeviceSecret string `json:"deviceSecret"`
}

func NewHTTPDynRegClient(cfg *config.Config) *HTTPDynRegClient {
	return &HTTPDynRegClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *HTTPDynRegClient) Register() (string, error) {
	if c.config.Device.ProductSecret == "" {
		return "", fmt.Errorf("product secret is required for dynamic registration")
	}

	random := fmt.Sprintf("%d", time.Now().UnixMilli())

	signature := auth.GenerateDynRegSignature(
		c.config.Device.ProductKey,
		c.config.Device.DeviceName,
		c.config.Device.ProductSecret,
		random,
	)

	formData := url.Values{}
	formData.Set("productKey", c.config.Device.ProductKey)
	formData.Set("deviceName", c.config.Device.DeviceName)
	formData.Set("random", random)
	formData.Set("sign", signature)
	formData.Set("signMethod", "hmacsha256")

	reqURL := fmt.Sprintf("http://%s/auth/register/device", c.config.MQTT.Host)
	
	req, err := http.NewRequest("POST", reqURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/xml,text/javascript,text/html,application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var dynRegResp DynRegResponse
	if err := json.Unmarshal(body, &dynRegResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if dynRegResp.Code != 200 {
		return "", fmt.Errorf("dynamic registration failed: code=%d, message=%s", dynRegResp.Code, dynRegResp.Message)
	}

	return dynRegResp.Data.DeviceSecret, nil
}

