package ota

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/iot-go-sdk/pkg/mqtt"
)

// DigestType represents the digest method type
type DigestType int

const (
	DigestMD5 DigestType = iota
	DigestSHA256
)

// ProtocolType represents the OTA download protocol
type ProtocolType int

const (
	ProtocolHTTPS ProtocolType = iota
	ProtocolMQTT
)

// RecvType represents the type of OTA message
type RecvType int

const (
	RecvTypeFOTA RecvType = iota // Firmware Over-The-Air
	RecvTypeCOTA                 // Configuration Over-The-Air
)

// TaskDesc describes an OTA task
type TaskDesc struct {
	ProductKey    string       `json:"productKey"`
	DeviceName    string       `json:"deviceName"`
	URL           string       `json:"url"`
	StreamID      uint32       `json:"streamId,omitempty"`
	StreamFileID  uint32       `json:"streamFileId,omitempty"`
	Size          uint32       `json:"size"`
	DigestMethod  DigestType   `json:"digestMethod"`
	ExpectDigest  string       `json:"sign"`
	Version       string       `json:"version"`
	Module        string       `json:"module,omitempty"`
	ExtraData     string       `json:"extData,omitempty"`
	FileName      string       `json:"fileName,omitempty"`
	FileNum       uint32       `json:"fileNum,omitempty"`
	FileID        uint32       `json:"fileId,omitempty"`
	ProtocolType  ProtocolType `json:"-"`
}

// RecvHandler is the callback for OTA messages
type RecvHandler func(client *Client, recvType RecvType, task *TaskDesc)

// DownloadHandler is the callback for download progress
type DownloadHandler func(percent int, data []byte, err error)

// Client represents an OTA client
type Client struct {
	mqttClient      *mqtt.Client
	productKey      string
	deviceName      string
	currentVersion  string
	recvHandler     RecvHandler
	downloadHandler DownloadHandler
	logger          *log.Logger
	mutex           sync.RWMutex
	downloadCtx     context.Context
	downloadCancel  context.CancelFunc
}

// NewClient creates a new OTA client
func NewClient(mqttClient *mqtt.Client, productKey, deviceName string) *Client {
	return &Client{
		mqttClient: mqttClient,
		productKey: productKey,
		deviceName: deviceName,
		logger:     log.Default(),
	}
}

// SetLogger sets the logger for the OTA client
func (c *Client) SetLogger(logger *log.Logger) {
	c.logger = logger
}

// SetRecvHandler sets the OTA message receive handler
func (c *Client) SetRecvHandler(handler RecvHandler) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.recvHandler = handler
}

// SetDownloadHandler sets the download progress handler
func (c *Client) SetDownloadHandler(handler DownloadHandler) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.downloadHandler = handler
}

// Start starts the OTA client
func (c *Client) Start() error {
	// Subscribe to OTA topics using wildcards like C SDK
	// This allows receiving messages for any productKey/deviceName combination
	fotaTopic := "/ota/device/upgrade/+/+"
	if err := c.mqttClient.Subscribe(fotaTopic, 0, c.handleOTAMessage); err != nil {
		return fmt.Errorf("failed to subscribe to FOTA topic: %w", err)
	}

	cotaTopic := "/sys/+/+/thing/config/push"
	if err := c.mqttClient.Subscribe(cotaTopic, 0, c.handleOTAMessage); err != nil {
		return fmt.Errorf("failed to subscribe to COTA topic: %w", err)
	}

	c.logger.Printf("OTA client started, subscribed to FOTA topic: %s and COTA topic: %s", fotaTopic, cotaTopic)
	return nil
}

// Stop stops the OTA client
func (c *Client) Stop() error {
	// Cancel any ongoing downloads
	if c.downloadCancel != nil {
		c.downloadCancel()
	}

	// Unsubscribe from OTA topics (using same wildcards as Start)
	fotaTopic := "/ota/device/upgrade/+/+"
	c.mqttClient.Unsubscribe(fotaTopic)

	cotaTopic := "/sys/+/+/thing/config/push"
	c.mqttClient.Unsubscribe(cotaTopic)

	return nil
}

// ReportVersion reports the current firmware version to the cloud
func (c *Client) ReportVersion(version string) error {
	c.mutex.Lock()
	c.currentVersion = version
	c.mutex.Unlock()

	topic := fmt.Sprintf("/ota/device/inform/%s/%s", c.productKey, c.deviceName)
	
	payload := map[string]interface{}{
		"id":      fmt.Sprintf("%d", time.Now().UnixNano()),
		"params": map[string]interface{}{
			"version": version,
			"module":  "",
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal version report: %w", err)
	}

	if err := c.mqttClient.Publish(topic, data, 0, false); err != nil {
		return fmt.Errorf("failed to publish version report: %w", err)
	}

	c.logger.Printf("Reported version: %s", version)
	return nil
}

// ReportProgress reports OTA download/upgrade progress
func (c *Client) ReportProgress(step string, desc string, progress int, module string) error {
	topic := fmt.Sprintf("/ota/device/progress/%s/%s", c.productKey, c.deviceName)
	
	payload := map[string]interface{}{
		"id":      fmt.Sprintf("%d", time.Now().UnixNano()),
		"params": map[string]interface{}{
			"step":     step,
			"desc":     desc,
			"progress": progress,
		},
	}

	if module != "" {
		payload["params"].(map[string]interface{})["module"] = module
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal progress report: %w", err)
	}

	if err := c.mqttClient.Publish(topic, data, 0, false); err != nil {
		return fmt.Errorf("failed to publish progress report: %w", err)
	}

	return nil
}

// QueryFirmware queries for available firmware updates
func (c *Client) QueryFirmware() error {
	topic := fmt.Sprintf("/sys/%s/%s/thing/ota/firmware/get", c.productKey, c.deviceName)
	
	payload := map[string]interface{}{
		"id":      fmt.Sprintf("%d", time.Now().UnixNano()),
		"version": "1.0",
		"params":  map[string]interface{}{},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal firmware query: %w", err)
	}

	if err := c.mqttClient.Publish(topic, data, 0, false); err != nil {
		return fmt.Errorf("failed to publish firmware query: %w", err)
	}

	c.logger.Printf("Queried for firmware updates")
	return nil
}

// handleOTAMessage handles incoming OTA messages
func (c *Client) handleOTAMessage(topic string, payload []byte) {
	c.logger.Printf("Received OTA message on topic %s: %s", topic, string(payload))

	var msg map[string]interface{}
	if err := json.Unmarshal(payload, &msg); err != nil {
		c.logger.Printf("Failed to unmarshal OTA message: %v", err)
		return
	}

	// Determine message type based on topic pattern (with wildcards)
	var recvType RecvType
	// Check if topic matches FOTA pattern: /ota/device/upgrade/+/+
	if len(topic) > 20 && topic[:20] == "/ota/device/upgrade/" {
		recvType = RecvTypeFOTA
	} else {
		recvType = RecvTypeCOTA
	}

	// Parse task description
	task := c.parseTaskDesc(msg)
	if task == nil {
		c.logger.Printf("Failed to parse task description")
		return
	}

	// Call user handler
	c.mutex.RLock()
	handler := c.recvHandler
	c.mutex.RUnlock()

	if handler != nil {
		handler(c, recvType, task)
	}
}

// parseTaskDesc parses OTA task description from message
func (c *Client) parseTaskDesc(msg map[string]interface{}) *TaskDesc {
	data, ok := msg["data"].(map[string]interface{})
	if !ok {
		return nil
	}

	task := &TaskDesc{
		ProductKey: c.productKey,
		DeviceName: c.deviceName,
	}

	// Parse common fields
	if url, ok := data["url"].(string); ok {
		task.URL = url
		task.ProtocolType = ProtocolHTTPS
	}

	if size, ok := data["size"].(float64); ok {
		task.Size = uint32(size)
	}

	if sign, ok := data["sign"].(string); ok {
		task.ExpectDigest = sign
	}

	if signMethod, ok := data["signMethod"].(string); ok {
		if signMethod == "Md5" || signMethod == "MD5" {
			task.DigestMethod = DigestMD5
		} else {
			task.DigestMethod = DigestSHA256
		}
	}

	if version, ok := data["version"].(string); ok {
		task.Version = version
	}

	if module, ok := data["module"].(string); ok {
		task.Module = module
	}

	if extData, ok := data["extData"].(string); ok {
		task.ExtraData = extData
	}

	return task
}

// Download downloads firmware from the given task
func (c *Client) Download(ctx context.Context, task *TaskDesc, rangeStart, rangeEnd uint32) error {
	if task.ProtocolType != ProtocolHTTPS {
		return fmt.Errorf("only HTTPS protocol is supported")
	}

	c.downloadCtx, c.downloadCancel = context.WithCancel(ctx)
	defer c.downloadCancel()

	// Create HTTP request
	req, err := http.NewRequestWithContext(c.downloadCtx, "GET", task.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set range header if specified
	if rangeEnd > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd))
	} else if rangeStart > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", rangeStart))
	}

	// Send request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Initialize hash calculator
	var hasher hash.Hash
	if task.DigestMethod == DigestMD5 {
		hasher = md5.New()
	} else {
		hasher = sha256.New()
	}

	// Download with progress reporting
	var downloaded uint32
	totalSize := task.Size
	if rangeEnd > 0 && rangeStart <= rangeEnd {
		totalSize = rangeEnd - rangeStart + 1
	}

	buffer := make([]byte, 2048)
	lastPercent := -1

	for {
		select {
		case <-c.downloadCtx.Done():
			return fmt.Errorf("download cancelled")
		default:
			n, err := resp.Body.Read(buffer)
			if err != nil && err != io.EOF {
				c.notifyDownloadHandler(-1, nil, err)
				return fmt.Errorf("failed to read response: %w", err)
			}

			if n > 0 {
				data := buffer[:n]
				hasher.Write(data)
				downloaded += uint32(n)

				// Calculate progress
				percent := int(downloaded * 100 / totalSize)
				if percent != lastPercent {
					lastPercent = percent
					c.notifyDownloadHandler(percent, data, nil)
				}
			}

			if err == io.EOF {
				// Verify digest
				digest := fmt.Sprintf("%x", hasher.Sum(nil))
				if digest != task.ExpectDigest {
					err := fmt.Errorf("digest mismatch: expected %s, got %s", task.ExpectDigest, digest)
					c.notifyDownloadHandler(-3, nil, err)
					return err
				}

				// Download completed
				c.notifyDownloadHandler(100, nil, nil)
				return nil
			}
		}
	}
}

// notifyDownloadHandler calls the download handler if set
func (c *Client) notifyDownloadHandler(percent int, data []byte, err error) {
	c.mutex.RLock()
	handler := c.downloadHandler
	c.mutex.RUnlock()

	if handler != nil {
		handler(percent, data, err)
	}
}