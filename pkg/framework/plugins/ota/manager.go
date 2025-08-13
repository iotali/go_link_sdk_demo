package ota

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/iot-go-sdk/pkg/mqtt"
	"github.com/iot-go-sdk/pkg/ota"
)

// ManagerImpl implements the Manager interface
type ManagerImpl struct {
	mqttClient      *mqtt.Client
	otaClient       *ota.Client
	productKey      string
	deviceName      string
	versionProvider VersionProvider
	downloader      Downloader
	updater         Updater
	
	currentVersion  string
	status          Status
	statusCallback  StatusCallback
	autoUpdate      bool
	
	mu              sync.RWMutex
	logger          *log.Logger
	stopCh          chan struct{}
	wg              sync.WaitGroup
}

// NewManager creates a new OTA manager
func NewManager(mqttClient *mqtt.Client, productKey, deviceName string, versionProvider VersionProvider) Manager {
	manager := &ManagerImpl{
		mqttClient:      mqttClient,
		productKey:      productKey,
		deviceName:      deviceName,
		versionProvider: versionProvider,
		status:          StatusIdle,
		autoUpdate:      true,
		logger:          log.New(os.Stdout, fmt.Sprintf("[OTA-%s] ", deviceName), log.LstdFlags),
		stopCh:          make(chan struct{}),
	}
	
	// Create OTA client
	manager.otaClient = ota.NewClient(mqttClient, productKey, deviceName)
	
	// Get current version
	manager.currentVersion = versionProvider.GetVersion()
	
	// Create default downloader and updater
	manager.downloader = NewSimpleDownloader()
	manager.updater = NewBinaryUpdater(manager.logger)
	
	return manager
}

// Start starts the OTA manager
func (m *ManagerImpl) Start() error {
	m.logger.Printf("Starting OTA manager, current version: %s", m.currentVersion)
	
	// Set up OTA handlers
	m.setupHandlers()
	
	// Start OTA client
	if err := m.otaClient.Start(); err != nil {
		return fmt.Errorf("failed to start OTA client: %v", err)
	}
	
	// Report current version
	m.reportVersion()
	
	// Start periodic update check
	m.wg.Add(1)
	go m.updateCheckLoop()
	
	return nil
}

// Stop stops the OTA manager
func (m *ManagerImpl) Stop() error {
	m.logger.Println("Stopping OTA manager")
	
	// Signal stop
	select {
	case <-m.stopCh:
		// Already closed
	default:
		close(m.stopCh)
	}
	
	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// All goroutines finished
	case <-time.After(2 * time.Second):
		m.logger.Println("Warning: Timeout waiting for OTA manager goroutines to stop")
	}
	
	// Stop OTA client
	if m.otaClient != nil {
		if err := m.otaClient.Stop(); err != nil {
			m.logger.Printf("Warning: Failed to stop OTA client: %v", err)
			// Don't return error, continue cleanup
		}
	}
	
	return nil
}

// GetCurrentVersion returns the current firmware version
func (m *ManagerImpl) GetCurrentVersion() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentVersion
}

// CheckUpdate checks for available updates
func (m *ManagerImpl) CheckUpdate() (*UpdateInfo, error) {
	m.logger.Println("Checking for updates...")
	
	// Query update by reporting current version
	// This will trigger the platform to send update info if available
	m.reportVersion()
	
	// Updates are handled asynchronously via callback
	return nil, nil
}

// PerformUpdate performs the firmware update
func (m *ManagerImpl) PerformUpdate(info *UpdateInfo) (*UpdateResult, error) {
	m.mu.Lock()
	if m.status != StatusIdle {
		m.mu.Unlock()
		return &UpdateResult{
			Success: false,
			Message: "Update already in progress",
			Code:    -1,
		}, nil
	}
	m.status = StatusDownloading
	m.mu.Unlock()
	
	// Notify status change
	m.notifyStatus(StatusDownloading, 0, "Starting download")
	
	// Download firmware
	ctx := context.Background()
	data, err := m.downloader.Download(ctx, info, func(current, total int64, percentage float64) {
		m.notifyStatus(StatusDownloading, int32(percentage), fmt.Sprintf("Downloading: %d/%d bytes", current, total))
	})
	
	if err != nil {
		m.setStatus(StatusFailed)
		m.notifyStatus(StatusFailed, 0, fmt.Sprintf("Download failed: %v", err))
		return &UpdateResult{
			Success: false,
			Message: fmt.Sprintf("Download failed: %v", err),
			Code:    -2,
		}, nil
	}
	
	// Verify firmware
	m.setStatus(StatusVerifying)
	m.notifyStatus(StatusVerifying, 50, "Verifying firmware")
	
	if err := m.downloader.Verify(data, info); err != nil {
		m.setStatus(StatusFailed)
		m.notifyStatus(StatusFailed, 0, fmt.Sprintf("Verification failed: %v", err))
		return &UpdateResult{
			Success: false,
			Message: fmt.Sprintf("Verification failed: %v", err),
			Code:    -3,
		}, nil
	}
	
	// Prepare update
	m.setStatus(StatusUpdating)
	m.notifyStatus(StatusUpdating, 75, "Preparing update")
	
	if err := m.updater.PrepareUpdate(data); err != nil {
		m.setStatus(StatusFailed)
		m.notifyStatus(StatusFailed, 0, fmt.Sprintf("Update preparation failed: %v", err))
		return &UpdateResult{
			Success: false,
			Message: fmt.Sprintf("Update preparation failed: %v", err),
			Code:    -4,
		}, nil
	}
	
	// Update version
	if err := m.versionProvider.SetVersion(info.Version); err != nil {
		m.logger.Printf("Failed to save version: %v", err)
	}
	
	// Report progress
	m.otaClient.ReportProgress("download", "Update prepared", 100, "")
	
	// Execute update (this may restart the process)
	m.setStatus(StatusRestarting)
	m.notifyStatus(StatusRestarting, 100, "Restarting with new version")
	
	if err := m.updater.ExecuteUpdate(); err != nil {
		// If we're here, update failed
		m.setStatus(StatusFailed)
		m.notifyStatus(StatusFailed, 0, fmt.Sprintf("Update execution failed: %v", err))
		
		// Try to rollback
		if rollbackErr := m.updater.Rollback(); rollbackErr != nil {
			m.logger.Printf("Rollback failed: %v", rollbackErr)
		}
		
		return &UpdateResult{
			Success: false,
			Message: fmt.Sprintf("Update execution failed: %v", err),
			Code:    -5,
		}, nil
	}
	
	// If we reach here, update was successful but didn't restart
	m.setStatus(StatusIdle)
	m.currentVersion = info.Version
	m.notifyStatus(StatusIdle, 100, "Update completed")
	
	return &UpdateResult{
		Success: true,
		Message: "Update completed successfully",
		Code:    0,
	}, nil
}

// SetStatusCallback sets the status callback
func (m *ManagerImpl) SetStatusCallback(callback StatusCallback) {
	m.mu.Lock()
	m.statusCallback = callback
	m.mu.Unlock()
}

// SetAutoUpdate enables or disables auto-update
func (m *ManagerImpl) SetAutoUpdate(enabled bool) {
	m.mu.Lock()
	m.autoUpdate = enabled
	m.mu.Unlock()
}

// GetStatus returns the current OTA status
func (m *ManagerImpl) GetStatus() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// setupHandlers sets up OTA message handlers
func (m *ManagerImpl) setupHandlers() {
	// Handle OTA message
	m.otaClient.SetRecvHandler(func(client *ota.Client, recvType ota.RecvType, task *ota.TaskDesc) {
		m.logger.Printf("=== OTA Update Available ===")
		m.logger.Printf("  Current version: %s", m.currentVersion)
		m.logger.Printf("  New version: %s", task.Version)
		m.logger.Printf("  Size: %d bytes", task.Size)
		
		// Check if it's a new version
		if task.Version == m.currentVersion {
			m.logger.Printf("Already on version %s, skipping update", task.Version)
			m.otaClient.ReportProgress("download", "Already on latest version", 100, "")
			return
		}
		
		// Convert to UpdateInfo
		digestMethod := "MD5"
		if task.DigestMethod == ota.DigestSHA256 {
			digestMethod = "SHA256"
		}
		
		info := &UpdateInfo{
			Version:      task.Version,
			URL:          task.URL,
			Size:         task.Size,
			Digest:       task.ExpectDigest,
			DigestMethod: digestMethod,
		}
		
		// Perform update if auto-update is enabled
		if m.autoUpdate {
			go func() {
				result, _ := m.PerformUpdate(info)
				if !result.Success {
					m.logger.Printf("Auto-update failed: %s", result.Message)
				}
			}()
		}
	})
}

// reportVersion reports the current version to the platform
func (m *ManagerImpl) reportVersion() {
	module := "default"
	if m.versionProvider != nil {
		module = m.versionProvider.GetModule()
	}
	m.logger.Printf("Reporting version to platform: %s (module: %s)", m.currentVersion, module)
	m.otaClient.ReportVersionWithModule(m.currentVersion, module)
}

// updateCheckLoop periodically checks for updates
func (m *ManagerImpl) updateCheckLoop() {
	defer m.wg.Done()
	
	// Initial check after 30 seconds
	select {
	case <-time.After(30 * time.Second):
		m.CheckUpdate()
	case <-m.stopCh:
		return
	}
	
	// Periodic checks every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			m.CheckUpdate()
		case <-m.stopCh:
			return
		}
	}
}

// setStatus sets the OTA status
func (m *ManagerImpl) setStatus(status Status) {
	m.mu.Lock()
	m.status = status
	m.mu.Unlock()
}

// notifyStatus notifies status change
func (m *ManagerImpl) notifyStatus(status Status, progress int32, message string) {
	m.mu.RLock()
	callback := m.statusCallback
	m.mu.RUnlock()
	
	if callback != nil {
		callback(status, progress, message)
	}
	
	// Report to platform
	if status == StatusFailed {
		m.otaClient.ReportProgress("download", message, -1, "")
	} else if status == StatusDownloading || status == StatusVerifying || status == StatusUpdating {
		m.otaClient.ReportProgress("download", message, int(progress), "")
	}
}

// VersionInfo stores version and module information
type VersionInfo struct {
	Version string `json:"version"`
	Module  string `json:"module"`
}

// FileVersionProvider provides version from a file
type FileVersionProvider struct {
	versionFile string
	cache       *VersionInfo
	mu          sync.RWMutex
}

// NewFileVersionProvider creates a new file-based version provider
func NewFileVersionProvider(versionFile string) *FileVersionProvider {
	p := &FileVersionProvider{
		versionFile: versionFile,
	}
	// Load initial version
	p.load()
	return p
}

// load reads version info from file
func (p *FileVersionProvider) load() {
	data, err := os.ReadFile(p.versionFile)
	if err != nil {
		p.cache = &VersionInfo{
			Version: "1.0.0",
			Module:  "default",
		}
		return
	}
	
	// Try to parse as JSON first
	var info VersionInfo
	if err := json.Unmarshal(data, &info); err == nil {
		p.cache = &info
		return
	}
	
	// Fallback to plain text (backward compatibility)
	version := strings.TrimSpace(string(data))
	p.cache = &VersionInfo{
		Version: version,
		Module:  "default",
	}
}

// save writes version info to file
func (p *FileVersionProvider) save() error {
	data, err := json.MarshalIndent(p.cache, "", "  ")
	if err != nil {
		return err
	}
	
	dir := filepath.Dir(p.versionFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	return os.WriteFile(p.versionFile, data, 0644)
}

// GetVersion gets the version from file
func (p *FileVersionProvider) GetVersion() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.cache == nil {
		p.load()
	}
	return p.cache.Version
}

// SetVersion saves the version to file
func (p *FileVersionProvider) SetVersion(version string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.cache == nil {
		p.load()
	}
	p.cache.Version = version
	return p.save()
}

// GetModule gets the module name
func (p *FileVersionProvider) GetModule() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if p.cache == nil {
		p.load()
	}
	if p.cache.Module == "" {
		return "default"
	}
	return p.cache.Module
}

// SetModule sets the module name
func (p *FileVersionProvider) SetModule(module string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.cache == nil {
		p.load()
	}
	p.cache.Module = module
	return p.save()
}