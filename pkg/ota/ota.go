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
	"strings"
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
	// Subscribe to specific OTA topics for this device
	// Using specific productKey/deviceName instead of wildcards for better compatibility
	fotaTopic := fmt.Sprintf("/ota/device/upgrade/%s/%s", c.productKey, c.deviceName)
	if err := c.mqttClient.Subscribe(fotaTopic, 0, c.handleOTAMessage); err != nil {
		return fmt.Errorf("failed to subscribe to FOTA topic: %w", err)
	}

	// Subscribe to firmware query reply topic
	firmwareReplyTopic := fmt.Sprintf("/sys/%s/%s/thing/ota/firmware/get_reply", c.productKey, c.deviceName)
	if err := c.mqttClient.Subscribe(firmwareReplyTopic, 0, c.handleOTAMessage); err != nil {
		return fmt.Errorf("failed to subscribe to firmware reply topic: %w", err)
	}

	c.logger.Printf("OTA client started, subscribed to FOTA topic: %s and firmware reply topic: %s", fotaTopic, firmwareReplyTopic)
	return nil
}

// Stop stops the OTA client
func (c *Client) Stop() error {
	// Cancel any ongoing downloads
	if c.downloadCancel != nil {
		c.downloadCancel()
	}

	// Unsubscribe from OTA topics (using specific topics)
	fotaTopic := fmt.Sprintf("/ota/device/upgrade/%s/%s", c.productKey, c.deviceName)
	c.mqttClient.Unsubscribe(fotaTopic)

	firmwareReplyTopic := fmt.Sprintf("/sys/%s/%s/thing/ota/firmware/get_reply", c.productKey, c.deviceName)
	c.mqttClient.Unsubscribe(firmwareReplyTopic)

	return nil
}

// ReportVersion reports the current firmware version to the cloud
func (c *Client) ReportVersion(version string) error {
	// Call ReportVersionWithModule with default module
	return c.ReportVersionWithModule(version, "default")
}

// ReportVersionWithModule reports the current firmware version with module to the cloud
func (c *Client) ReportVersionWithModule(version string, module string) error {
	c.mutex.Lock()
	c.currentVersion = version
	c.mutex.Unlock()

	topic := fmt.Sprintf("/ota/device/inform/%s/%s", c.productKey, c.deviceName)
	
	params := map[string]interface{}{
		"version": version,
	}
	
	// Only add module if it's not empty (following C SDK behavior)
	if module != "" {
		params["module"] = module
	}
	
	payload := map[string]interface{}{
		"id":      fmt.Sprintf("%d", time.Now().UnixNano()),
		"params": params,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal version report: %w", err)
	}

	if err := c.mqttClient.Publish(topic, data, 0, false); err != nil {
		return fmt.Errorf("failed to publish version report: %w", err)
	}

	if module != "" {
		c.logger.Printf("Reported version: %s (module: %s)", version, module)
	} else {
		c.logger.Printf("Reported version: %s", version)
	}
	return nil
}

// ReportProgress reports OTA download/upgrade progress
func (c *Client) ReportProgress(step string, desc string, progress int, module string) error {
	topic := fmt.Sprintf("/ota/device/progress/%s/%s", c.productKey, c.deviceName)
	
	params := map[string]interface{}{
		"step":     step,
		"desc":     desc,
		"progress": progress,
	}
	
	// Only add module if it's not empty (following C SDK behavior)
	if module != "" {
		params["module"] = module
	}
	
	payload := map[string]interface{}{
		"id":      fmt.Sprintf("%d", time.Now().UnixNano()),
		"params": params,
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
	return c.QueryFirmwareWithModule("")
}

// QueryFirmwareWithModule queries for available firmware updates with module parameter
func (c *Client) QueryFirmwareWithModule(module string) error {
	topic := fmt.Sprintf("/sys/%s/%s/thing/ota/firmware/get", c.productKey, c.deviceName)
	
	params := map[string]interface{}{}
	if module != "" {
		params["module"] = module
	}
	
	payload := map[string]interface{}{
		"id":      fmt.Sprintf("%d", time.Now().UnixNano()),
		"version": "1.0",
		"params":  params,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal firmware query: %w", err)
	}

	if err := c.mqttClient.Publish(topic, data, 0, false); err != nil {
		return fmt.Errorf("failed to publish firmware query: %w", err)
	}

	if module != "" {
		c.logger.Printf("Queried for firmware updates (module: %s)", module)
	} else {
		c.logger.Printf("Queried for firmware updates")
	}
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

	// Check if this is a firmware query reply
	if strings.Contains(topic, "/thing/ota/firmware/get_reply") {
		// This is a firmware update notification from the query
		// Parse it as a FOTA task
		task := c.parseTaskDesc(msg)
		if task == nil {
			// No firmware update available or invalid data - this is not an error
			return
		}

		// Call user handler with FOTA type
		c.mutex.RLock()
		handler := c.recvHandler
		c.mutex.RUnlock()

		if handler != nil {
			handler(c, RecvTypeFOTA, task)
		}
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

	// Check if data is empty (no firmware update available)
	if len(data) == 0 {
		c.logger.Printf("No firmware update available")
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

	// Validate required fields for firmware update
	if task.URL == "" || task.Size == 0 {
		c.logger.Printf("Invalid firmware update data: missing URL or size")
		return nil
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
	// Use longer timeout for large file downloads
	client := &http.Client{
		Timeout: 10 * time.Minute, // Increased timeout for large files
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	// Log response headers for debugging
	if c.logger != nil {
		c.logger.Printf("HTTP Status: %s, Content-Length: %d", resp.Status, resp.ContentLength)
		if contentType := resp.Header.Get("Content-Type"); contentType != "" {
			c.logger.Printf("Content-Type: %s", contentType)
		}
	}

	// For partial downloads, we don't verify digest
	// The digest verification should be done after downloading the complete file
	isPartialDownload := (rangeStart > 0 || rangeEnd > 0)

	// Initialize hash calculator only for full downloads
	var hasher hash.Hash
	if !isPartialDownload {
		if task.DigestMethod == DigestMD5 {
			hasher = md5.New()
		} else {
			hasher = sha256.New()
		}
	}

	// Download with progress reporting
	totalSize := uint64(task.Size)
	if rangeEnd > 0 && rangeStart <= rangeEnd {
		totalSize = uint64(rangeEnd - rangeStart + 1)
	}
	
	// Get actual content length from response header
	if contentLength := resp.ContentLength; contentLength > 0 {
		totalSize = uint64(contentLength)
		if c.logger != nil {
			c.logger.Printf("Actual download size: %d bytes (task.Size was %d)", totalSize, task.Size)
		}
	}

	// Use io.Copy with a custom writer to track progress
	type progressWriter struct {
		totalSize  uint64
		downloaded uint64
		lastPercent int
		hasher     hash.Hash
		client     *Client
	}
	
	pw := &progressWriter{
		totalSize:   totalSize,
		downloaded:  0,
		lastPercent: -1,
		hasher:      hasher,
		client:      c,
	}
	
	// Custom Write method for progress tracking
	writeFunc := func(p []byte) (n int, err error) {
		n = len(p)
		if n > 0 {
			// Update hash
			if !isPartialDownload && pw.hasher != nil {
				pw.hasher.Write(p)
			}
			
			pw.downloaded += uint64(n)
			
			// Calculate and report progress
			percent := int(pw.downloaded * 100 / pw.totalSize)
			if percent > 100 {
				percent = 100
			}
			
			if percent != pw.lastPercent {
				pw.lastPercent = percent
				pw.client.notifyDownloadHandler(percent, p, nil)
			}
		}
		return n, nil
	}
	
	// Use io.Copy to ensure we read all data
	buffer := make([]byte, 32*1024) // Use larger buffer for efficiency
	var totalDownloaded int64
	
	for {
		select {
		case <-c.downloadCtx.Done():
			return fmt.Errorf("download cancelled")
		default:
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				_, writeErr := writeFunc(buffer[:n])
				if writeErr != nil {
					return fmt.Errorf("failed to process data: %w", writeErr)
				}
				totalDownloaded += int64(n)
			}
			
			if err != nil {
				if err == io.EOF {
					// EOF reached, check if download is complete
					if pw.downloaded < totalSize {
						// Try to continue reading in case it's a false EOF
						if c.logger != nil {
							c.logger.Printf("Received EOF at %d bytes, expected %d bytes. Attempting to continue...", pw.downloaded, totalSize)
						}
						// Give it a small delay and retry
						time.Sleep(100 * time.Millisecond)
						
						// Try one more read
						n2, err2 := resp.Body.Read(buffer)
						if n2 > 0 {
							_, writeErr := writeFunc(buffer[:n2])
							if writeErr != nil {
								return fmt.Errorf("failed to process data: %w", writeErr)
							}
							totalDownloaded += int64(n2)
							continue // Continue reading
						}
						
						if err2 == io.EOF && pw.downloaded < totalSize {
							// Really incomplete
							err := fmt.Errorf("download incomplete: got %d bytes, expected %d bytes", pw.downloaded, totalSize)
							if c.logger != nil {
								c.logger.Printf("Download incomplete: %d/%d bytes (%.1f%%)", pw.downloaded, totalSize, float64(pw.downloaded)*100/float64(totalSize))
							}
							c.notifyDownloadHandler(-2, nil, err)
							return err
						}
					}
					
					// Download complete
					break
				} else {
					// Other errors
					c.notifyDownloadHandler(-1, nil, err)
					return fmt.Errorf("failed to read response: %w", err)
				}
			}
		}
		
		if err == io.EOF {
			break
		}
	}
	
	// Verify digest
	if !isPartialDownload && hasher != nil {
		digest := fmt.Sprintf("%x", hasher.Sum(nil))
		if digest != task.ExpectDigest {
			err := fmt.Errorf("digest mismatch: expected %s, got %s", task.ExpectDigest, digest)
			c.notifyDownloadHandler(-3, nil, err)
			return err
		}
	}
	
	// Download completed successfully
	c.notifyDownloadHandler(100, nil, nil)
	return nil
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