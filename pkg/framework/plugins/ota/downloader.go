package ota

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SimpleDownloader implements simple HTTP download
type SimpleDownloader struct {
	client *http.Client
}

// NewSimpleDownloader creates a new simple downloader
func NewSimpleDownloader() Downloader {
	return &SimpleDownloader{
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// Download downloads firmware from URL
func (d *SimpleDownloader) Download(ctx context.Context, info *UpdateInfo, progress ProgressCallback) ([]byte, error) {
	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", info.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Execute request
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	// Get content length
	contentLength := resp.ContentLength
	if contentLength < 0 {
		contentLength = int64(info.Size)
	}
	
	// Read data with progress reporting
	data := make([]byte, 0, contentLength)
	buffer := make([]byte, 32*1024) // 32KB buffer
	totalRead := int64(0)
	
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			data = append(data, buffer[:n]...)
			totalRead += int64(n)
			
			// Report progress
			if progress != nil && contentLength > 0 {
				percentage := float64(totalRead) / float64(contentLength) * 100
				progress(totalRead, contentLength, percentage)
			}
		}
		
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
	}
	
	// Verify size
	if uint32(len(data)) != info.Size {
		return nil, fmt.Errorf("size mismatch: got %d bytes, expected %d bytes", len(data), info.Size)
	}
	
	return data, nil
}

// Verify verifies the downloaded firmware
func (d *SimpleDownloader) Verify(data []byte, info *UpdateInfo) error {
	// Calculate MD5
	hash := md5.Sum(data)
	digest := fmt.Sprintf("%x", hash)
	
	// Compare with expected digest
	if digest != info.Digest {
		return fmt.Errorf("digest mismatch: expected %s, got %s", info.Digest, digest)
	}
	
	return nil
}

// ChunkedDownloader implements chunked download with resume support
type ChunkedDownloader struct {
	client    *http.Client
	chunkSize int64
}

// NewChunkedDownloader creates a new chunked downloader
func NewChunkedDownloader(chunkSize int64) Downloader {
	return &ChunkedDownloader{
		client: &http.Client{
			Timeout: 30 * time.Second, // Per-chunk timeout
		},
		chunkSize: chunkSize,
	}
}

// Download downloads firmware in chunks
func (d *ChunkedDownloader) Download(ctx context.Context, info *UpdateInfo, progress ProgressCallback) ([]byte, error) {
	data := make([]byte, info.Size)
	totalSize := int64(info.Size)
	downloaded := int64(0)
	
	for downloaded < totalSize {
		// Calculate chunk range
		start := downloaded
		end := start + d.chunkSize - 1
		if end >= totalSize {
			end = totalSize - 1
		}
		
		// Download chunk
		chunk, err := d.downloadChunk(ctx, info.URL, start, end)
		if err != nil {
			return nil, fmt.Errorf("failed to download chunk %d-%d: %w", start, end, err)
		}
		
		// Copy chunk to data
		copy(data[start:], chunk)
		downloaded += int64(len(chunk))
		
		// Report progress
		if progress != nil {
			percentage := float64(downloaded) / float64(totalSize) * 100
			progress(downloaded, totalSize, percentage)
		}
	}
	
	return data, nil
}

// downloadChunk downloads a specific chunk
func (d *ChunkedDownloader) downloadChunk(ctx context.Context, url string, start, end int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	// Set range header
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	// Check status code (206 for partial content)
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	return io.ReadAll(resp.Body)
}

// Verify verifies the downloaded firmware
func (d *ChunkedDownloader) Verify(data []byte, info *UpdateInfo) error {
	// Same verification as SimpleDownloader
	hash := md5.Sum(data)
	digest := fmt.Sprintf("%x", hash)
	
	if digest != info.Digest {
		return fmt.Errorf("digest mismatch: expected %s, got %s", info.Digest, digest)
	}
	
	return nil
}