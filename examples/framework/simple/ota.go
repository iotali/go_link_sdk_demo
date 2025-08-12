package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/iot-go-sdk/pkg/mqtt"
	"github.com/iot-go-sdk/pkg/ota"
)

// OTAManager handles firmware updates with self-update capability
type OTAManager struct {
	otaClient      *ota.Client
	mqttClient     *mqtt.Client
	oven           *ElectricOven
	currentVersion string
	versionFile    string
	executablePath string
	backupPath     string
	tempPath       string
	logger         *log.Logger
	isUpdating     bool
}

// NewOTAManager creates a new OTA manager
func NewOTAManager(mqttClient *mqtt.Client, productKey, deviceName string, oven *ElectricOven) *OTAManager {
	// Get the path of the current executable
	execPath, err := os.Executable()
	if err != nil {
		log.Printf("[OTA] Warning: Failed to get executable path: %v", err)
		execPath = "./oven" // Fallback
	}

	// Resolve symbolic links to get the real path
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		log.Printf("[OTA] Warning: Failed to resolve executable path: %v", err)
	}

	dir := filepath.Dir(execPath)

	// Create OTA client
	otaClient := ota.NewClient(mqttClient, productKey, deviceName)

	// Get initial version from oven or use default
	currentVersion := "1.0.0"
	if oven != nil {
		currentVersion = oven.firmwareVersion
	}

	manager := &OTAManager{
		otaClient:      otaClient,
		mqttClient:     mqttClient,
		oven:           oven,
		currentVersion: currentVersion,
		versionFile:    filepath.Join(dir, "version.txt"),
		executablePath: execPath,
		backupPath:     execPath + ".backup",
		tempPath:       execPath + ".new",
		logger:         log.New(os.Stdout, "[OTA] ", log.LstdFlags),
		isUpdating:     false,
	}

	return manager
}

// Start begins OTA monitoring and management
func (m *OTAManager) Start() error {
	m.logger.Printf("Starting OTA manager")
	m.logger.Printf("Executable path: %s", m.executablePath)
	m.logger.Printf("Current version: %s", m.currentVersion)

	// Load version from file if exists
	if savedVersion := m.loadVersion(); savedVersion != "" {
		m.currentVersion = savedVersion
		if m.oven != nil {
			m.oven.SetFirmwareVersion(savedVersion)
		}
	} else {
		// Save current version
		m.saveVersion(m.currentVersion)
	}

	// Set up OTA handlers
	m.setupHandlers()

	// Start OTA client
	if err := m.otaClient.Start(); err != nil {
		return fmt.Errorf("failed to start OTA client: %v", err)
	}

	// Report current version
	m.logger.Printf("Reporting version to platform: %s", m.currentVersion)
	if err := m.otaClient.ReportVersion(m.currentVersion); err != nil {
		m.logger.Printf("Failed to report version: %v", err)
	}

	// Query for updates after a short delay
	go func() {
		time.Sleep(5 * time.Second)
		m.logger.Printf("Checking for firmware updates...")
		if err := m.otaClient.QueryFirmware(); err != nil {
			m.logger.Printf("Failed to query firmware: %v", err)
		}
	}()

	// Start periodic update checks
	go m.queryUpdatesLoop()

	return nil
}

// setupHandlers configures OTA event handlers
func (m *OTAManager) setupHandlers() {
	var downloadedData []byte
	var lastPercent int

	m.otaClient.SetRecvHandler(func(client *ota.Client, recvType ota.RecvType, task *ota.TaskDesc) {
		if recvType != ota.RecvTypeFOTA {
			m.logger.Printf("Ignoring non-FOTA task type: %v", recvType)
			return
		}

		m.logger.Printf("=== OTA Update Available ===")
		m.logger.Printf("  Current version: %s", m.currentVersion)
		m.logger.Printf("  New version: %s", task.Version)
		m.logger.Printf("  Size: %d bytes", task.Size)
		m.logger.Printf("  URL: %s", task.URL)
		m.logger.Printf("  Digest: %s", task.ExpectDigest)

		// Check if this is actually a newer version or empty (test)
		if task.Version == "" {
			m.logger.Printf("Empty version, treating as test update")
		} else if task.Version == m.currentVersion {
			m.logger.Printf("Same version, skipping update")
			client.ReportProgress("-1", "Same version", -1, task.Module)
			return
		}

		// Update oven status
		if m.oven != nil {
			m.oven.UpdateOTAStatus("downloading", 0)
		}

		// Reset download buffer
		downloadedData = make([]byte, 0, task.Size)
		lastPercent = 0

		// Start update process
		go m.performUpdate(task, &downloadedData)
	})

	// Set download progress handler
	m.otaClient.SetDownloadHandler(func(percent int, data []byte, err error) {
		if err != nil {
			m.logger.Printf("Download error: %v", err)
			if m.oven != nil {
				m.oven.UpdateOTAStatus("failed", int32(percent))
			}
			return
		}

		if data != nil {
			downloadedData = append(downloadedData, data...)
		}

		// Update oven OTA progress
		if m.oven != nil {
			m.oven.UpdateOTAStatus("downloading", int32(percent))
		}

		// Report progress every 10%
		if percent-lastPercent >= 10 || percent == 100 {
			m.logger.Printf("Download progress: %d%% (%d bytes)", percent, len(downloadedData))

			if err := m.otaClient.ReportProgress(
				fmt.Sprintf("%d", percent),
				"Downloading",
				percent,
				"",
			); err != nil {
				m.logger.Printf("Failed to report progress: %v", err)
			}

			lastPercent = percent
		}

		// When download completes, save the data
		if percent == 100 && len(downloadedData) > 0 {
			m.saveTempFirmware(downloadedData)
		}
	})
}

// performUpdate handles the complete update process
func (m *OTAManager) performUpdate(task *ota.TaskDesc, downloadedData *[]byte) {
	if m.isUpdating {
		m.logger.Printf("Update already in progress")
		return
	}

	m.isUpdating = true
	defer func() { m.isUpdating = false }()

	m.logger.Printf("Starting firmware update to version %s", task.Version)

	// Step 1: Download firmware using simple method
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	m.logger.Printf("Downloading firmware using simple method...")
	
	// Report download start
	if m.oven != nil {
		m.oven.UpdateOTAStatus("downloading", 0)
	}
	m.otaClient.ReportProgress("0", "Starting download", 0, task.Module)
	
	// Use simple download for reliability
	firmwareData, err := m.otaClient.SimpleDownload(ctx, task)
	if err != nil {
		m.logger.Printf("Download failed: %v", err)
		m.otaClient.ReportProgress("-2", "Download failed", -2, task.Module)
		if m.oven != nil {
			m.oven.UpdateOTAStatus("failed", 0)
		}
		return
	}
	
	m.logger.Printf("Downloaded %d bytes successfully", len(firmwareData))
	*downloadedData = firmwareData
	
	// Report download complete
	if m.oven != nil {
		m.oven.UpdateOTAStatus("downloading", 100)
	}
	m.otaClient.ReportProgress("100", "Download complete", 100, task.Module)

	// Update status to verifying (already verified in SimpleDownload)
	if m.oven != nil {
		m.oven.UpdateOTAStatus("verifying", 100)
	}
	
	// SimpleDownload already verified the firmware
	m.logger.Printf("Firmware verification already completed")

	// Update status to updating
	if m.oven != nil {
		m.oven.UpdateOTAStatus("updating", 100)
	}

	// Step 2: Save firmware to temp file
	if err := os.WriteFile(m.tempPath, *downloadedData, 0755); err != nil {
		m.logger.Printf("Failed to save firmware: %v", err)
		m.otaClient.ReportProgress("-4", "Save failed", -4, task.Module)
		if m.oven != nil {
			m.oven.UpdateOTAStatus("failed", 100)
		}
		return
	}

	// Step 3: Backup current executable
	if err := m.backupCurrentExecutable(); err != nil {
		m.logger.Printf("Failed to backup current executable: %v", err)
		m.otaClient.ReportProgress("-4", "Backup failed", -4, task.Module)
		if m.oven != nil {
			m.oven.UpdateOTAStatus("failed", 100)
		}
		return
	}

	// Step 4: Replace executable
	if err := m.replaceExecutable(); err != nil {
		m.logger.Printf("Failed to replace executable: %v", err)
		m.restoreBackup()
		m.otaClient.ReportProgress("-4", "Update failed", -4, task.Module)
		if m.oven != nil {
			m.oven.UpdateOTAStatus("failed", 100)
		}
		return
	}

	// Step 5: Update version file
	newVersion := task.Version
	if newVersion == "" {
		// For test updates without version, increment version
		newVersion = m.incrementVersion(m.currentVersion)
	}
	m.saveVersion(newVersion)

	// Step 6: Report success and prepare for restart
	m.logger.Printf("Update successful, preparing to restart with version %s...", newVersion)
	m.otaClient.ReportProgress("100", "Update complete", 100, task.Module)

	// Report new version (will be confirmed after restart)
	m.otaClient.ReportVersion(newVersion)

	// Update oven status
	if m.oven != nil {
		m.oven.UpdateOTAStatus("restarting", 100)
	}

	// Step 7: Trigger restart
	time.Sleep(2 * time.Second) // Give time for reports to be sent
	m.triggerRestart()
}

// incrementVersion increments the version number
func (m *OTAManager) incrementVersion(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return "1.0.1"
	}

	// Try to increment patch version
	var major, minor, patch int
	fmt.Sscanf(version, "%d.%d.%d", &major, &minor, &patch)
	return fmt.Sprintf("%d.%d.%d", major, minor, patch+1)
}

// saveTempFirmware saves downloaded firmware to temporary file
func (m *OTAManager) saveTempFirmware(data []byte) {
	if err := os.WriteFile(m.tempPath, data, 0755); err != nil {
		m.logger.Printf("Failed to save firmware: %v", err)
	} else {
		m.logger.Printf("Firmware saved to %s", m.tempPath)
	}
}

// verifyFirmware checks the integrity of downloaded firmware
func (m *OTAManager) verifyFirmware(data []byte, expectedDigest string) bool {
	if expectedDigest == "" {
		m.logger.Printf("No digest provided, skipping verification")
		return true
	}

	// Calculate MD5 hash
	hash := md5.Sum(data)
	actualDigest := hex.EncodeToString(hash[:])

	// Compare with expected (case-insensitive)
	if strings.EqualFold(actualDigest, expectedDigest) {
		m.logger.Printf("Firmware verification successful")
		return true
	}

	m.logger.Printf("Firmware verification failed: expected %s, got %s", expectedDigest, actualDigest)
	return false
}

// backupCurrentExecutable creates a backup of the current executable
func (m *OTAManager) backupCurrentExecutable() error {
	m.logger.Printf("Backing up current executable to %s", m.backupPath)

	// Remove old backup if exists
	os.Remove(m.backupPath)

	// Copy current to backup
	return copyFile(m.executablePath, m.backupPath)
}

// replaceExecutable replaces the current executable with the new one
func (m *OTAManager) replaceExecutable() error {
	m.logger.Printf("Replacing executable with new version")

	// On Unix systems, we can replace a running executable
	// On Windows, this would require a different approach
	if runtime.GOOS == "windows" {
		return m.replaceExecutableWindows()
	}

	return m.replaceExecutableUnix()
}

// replaceExecutableUnix handles Unix-like systems (Linux, macOS)
func (m *OTAManager) replaceExecutableUnix() error {
	// Remove the current executable (Unix allows this while running)
	if err := os.Remove(m.executablePath); err != nil {
		// If removal fails, try renaming instead
		tempOld := m.executablePath + ".old"
		if err := os.Rename(m.executablePath, tempOld); err != nil {
			return fmt.Errorf("failed to remove/rename old executable: %v", err)
		}
		defer os.Remove(tempOld) // Clean up later
	}

	// Move new executable to the correct location
	if err := os.Rename(m.tempPath, m.executablePath); err != nil {
		return fmt.Errorf("failed to move new executable: %v", err)
	}

	// Ensure executable permissions
	if err := os.Chmod(m.executablePath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %v", err)
	}

	return nil
}

// replaceExecutableWindows handles Windows systems
func (m *OTAManager) replaceExecutableWindows() error {
	// Windows doesn't allow replacing a running executable
	// We need to use a batch script or scheduled task

	// Create a batch script to replace the executable after exit
	batchScript := `@echo off
timeout /t 2 /nobreak > nul
move /y "%s" "%s"
start "" "%s"
del "%%~f0"
`
	scriptPath := m.executablePath + ".update.bat"
	script := fmt.Sprintf(batchScript, m.tempPath, m.executablePath, m.executablePath)

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to create update script: %v", err)
	}

	// Execute the batch script
	cmd := exec.Command("cmd", "/c", scriptPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start update script: %v", err)
	}

	return nil
}

// restoreBackup restores the backup executable if update fails
func (m *OTAManager) restoreBackup() {
	m.logger.Printf("Restoring backup executable")

	if _, err := os.Stat(m.backupPath); err == nil {
		os.Remove(m.executablePath)
		os.Rename(m.backupPath, m.executablePath)
		os.Chmod(m.executablePath, 0755)
	}
}

// triggerRestart restarts the application with the new executable
func (m *OTAManager) triggerRestart() {
	m.logger.Printf("=== RESTARTING WITH NEW VERSION ===")

	// Get current command line arguments
	args := os.Args

	if runtime.GOOS == "windows" {
		// On Windows, just exit and let the batch script restart
		os.Exit(0)
	}

	// On Unix systems, we can use exec to replace the current process
	m.logger.Printf("Executing new binary: %s", m.executablePath)

	// Use syscall.Exec to replace the current process
	env := os.Environ()
	err := syscall.Exec(m.executablePath, args, env)
	if err != nil {
		// If exec fails, try using exec.Command
		m.logger.Printf("syscall.Exec failed: %v, trying exec.Command", err)

		cmd := exec.Command(m.executablePath, args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Start(); err != nil {
			m.logger.Printf("Failed to restart: %v", err)
			return
		}

		// Exit current process
		os.Exit(0)
	}
}

// queryUpdatesLoop periodically queries for updates
func (m *OTAManager) queryUpdatesLoop() {
	// Query every 5 minutes
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if !m.isUpdating {
			m.logger.Printf("Periodic update check...")
			if err := m.otaClient.QueryFirmware(); err != nil {
				m.logger.Printf("Failed to query firmware: %v", err)
			}
		}
	}
}

// loadVersion loads version from file
func (m *OTAManager) loadVersion() string {
	data, err := os.ReadFile(m.versionFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// saveVersion saves version to file
func (m *OTAManager) saveVersion(version string) {
	if err := os.WriteFile(m.versionFile, []byte(version), 0644); err != nil {
		m.logger.Printf("Failed to save version: %v", err)
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// Stop stops the OTA manager
func (m *OTAManager) Stop() {
	m.logger.Printf("Stopping OTA manager")
	if m.otaClient != nil {
		m.otaClient.Stop()
	}
}