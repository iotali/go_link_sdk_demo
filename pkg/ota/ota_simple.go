package ota

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SimpleDownload performs a simple, direct download of the firmware
func (c *Client) SimpleDownload(ctx context.Context, task *TaskDesc) ([]byte, error) {
	c.logger.Printf("Starting simple download from %s", task.URL)
	
	// Create HTTP client with longer timeout
	client := &http.Client{
		Timeout: 30 * time.Minute, // Very long timeout for large files
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  true,
			DisableKeepAlives:   false,
			MaxIdleConnsPerHost: 10,
		},
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", task.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add headers to help with download
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "IoT-Device-OTA/1.0")
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	c.logger.Printf("Response: %s, Content-Length: %d", resp.Status, resp.ContentLength)
	
	// Read all data at once
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	c.logger.Printf("Downloaded %d bytes", len(data))
	
	// Verify size
	if uint32(len(data)) != task.Size {
		return nil, fmt.Errorf("size mismatch: got %d bytes, expected %d bytes", len(data), task.Size)
	}
	
	// Verify MD5
	hash := md5.Sum(data)
	digest := fmt.Sprintf("%x", hash)
	if digest != task.ExpectDigest {
		return nil, fmt.Errorf("digest mismatch: expected %s, got %s", task.ExpectDigest, digest)
	}
	
	c.logger.Printf("Download successful, MD5 verified")
	return data, nil
}