package ota

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

// BinaryUpdater implements binary file update with self-replacement
type BinaryUpdater struct {
	executablePath string
	backupPath     string
	tempPath       string
	logger         *log.Logger
}

// NewBinaryUpdater creates a new binary updater
func NewBinaryUpdater(logger *log.Logger) Updater {
	// Get the path of the current executable
	execPath, err := os.Executable()
	if err != nil {
		if logger != nil {
			logger.Printf("Warning: Failed to get executable path: %v", err)
		}
		execPath = "./app"
	}
	
	// Resolve symbolic links to get the real path
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		if logger != nil {
			logger.Printf("Warning: Failed to resolve executable path: %v", err)
		}
	}
	
	return &BinaryUpdater{
		executablePath: execPath,
		backupPath:     execPath + ".backup",
		tempPath:       execPath + ".new",
		logger:         logger,
	}
}

// CanUpdate checks if update is possible
func (u *BinaryUpdater) CanUpdate() bool {
	// Check if we have write permission to the executable directory
	dir := filepath.Dir(u.executablePath)
	
	// Try to create a test file
	testFile := filepath.Join(dir, ".ota_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		if u.logger != nil {
			u.logger.Printf("Cannot update: no write permission in %s", dir)
		}
		return false
	}
	os.Remove(testFile)
	
	return true
}

// PrepareUpdate prepares the update by saving the new binary
func (u *BinaryUpdater) PrepareUpdate(data []byte) error {
	// Backup current executable
	if err := u.backupCurrentExecutable(); err != nil {
		return fmt.Errorf("failed to backup current executable: %v", err)
	}
	
	// Write new executable to temp file
	if err := os.WriteFile(u.tempPath, data, 0755); err != nil {
		return fmt.Errorf("failed to write new executable: %v", err)
	}
	
	if u.logger != nil {
		u.logger.Printf("New firmware saved to %s (%d bytes)", u.tempPath, len(data))
	}
	
	return nil
}

// ExecuteUpdate executes the update and restarts the process
func (u *BinaryUpdater) ExecuteUpdate() error {
	if u.logger != nil {
		u.logger.Println("=== EXECUTING UPDATE ===")
	}
	
	// Platform-specific update
	if runtime.GOOS == "windows" {
		return u.executeUpdateWindows()
	}
	
	return u.executeUpdateUnix()
}

// Rollback rolls back to the previous version
func (u *BinaryUpdater) Rollback() error {
	// Check if backup exists
	if _, err := os.Stat(u.backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist")
	}
	
	// Remove failed update
	os.Remove(u.tempPath)
	
	// Restore backup
	if err := os.Rename(u.backupPath, u.executablePath); err != nil {
		// Try to copy instead
		data, readErr := os.ReadFile(u.backupPath)
		if readErr != nil {
			return fmt.Errorf("failed to read backup: %v", readErr)
		}
		
		if writeErr := os.WriteFile(u.executablePath, data, 0755); writeErr != nil {
			return fmt.Errorf("failed to restore backup: %v", writeErr)
		}
	}
	
	if u.logger != nil {
		u.logger.Println("Rolled back to previous version")
	}
	
	return nil
}

// backupCurrentExecutable creates a backup of the current executable
func (u *BinaryUpdater) backupCurrentExecutable() error {
	// Remove old backup if exists
	os.Remove(u.backupPath)
	
	// Read current executable
	data, err := os.ReadFile(u.executablePath)
	if err != nil {
		return fmt.Errorf("failed to read current executable: %v", err)
	}
	
	// Write backup
	if err := os.WriteFile(u.backupPath, data, 0755); err != nil {
		return fmt.Errorf("failed to write backup: %v", err)
	}
	
	if u.logger != nil {
		u.logger.Printf("Backed up current executable to %s", u.backupPath)
	}
	
	return nil
}

// executeUpdateUnix executes update on Unix-like systems
func (u *BinaryUpdater) executeUpdateUnix() error {
	// Remove current executable (Unix allows this while running)
	if err := os.Remove(u.executablePath); err != nil {
		if u.logger != nil {
			u.logger.Printf("Warning: Failed to remove old executable: %v", err)
		}
	}
	
	// Move new executable to the correct location
	if err := os.Rename(u.tempPath, u.executablePath); err != nil {
		// Try to copy instead
		data, readErr := os.ReadFile(u.tempPath)
		if readErr != nil {
			return fmt.Errorf("failed to read new executable: %v", readErr)
		}
		
		if writeErr := os.WriteFile(u.executablePath, data, 0755); writeErr != nil {
			return fmt.Errorf("failed to write new executable: %v", writeErr)
		}
		
		os.Remove(u.tempPath)
	}
	
	// Ensure executable permissions
	os.Chmod(u.executablePath, 0755)
	
	if u.logger != nil {
		u.logger.Println("=== RESTARTING WITH NEW VERSION ===")
	}
	
	// Use syscall.Exec to replace the current process
	return syscall.Exec(u.executablePath, os.Args, os.Environ())
}

// executeUpdateWindows executes update on Windows
func (u *BinaryUpdater) executeUpdateWindows() error {
	// Create a batch script to replace the executable
	scriptPath := u.executablePath + "_update.bat"
	script := fmt.Sprintf(`@echo off
echo Waiting for process to exit...
timeout /t 2 /nobreak > nul
echo Updating executable...
move /y "%s" "%s"
echo Starting new version...
start "" "%s"
del "%%~f0"
`, u.tempPath, u.executablePath, u.executablePath)
	
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		return fmt.Errorf("failed to create update script: %v", err)
	}
	
	if u.logger != nil {
		u.logger.Println("Starting update script...")
	}
	
	// Execute the batch script
	cmd := exec.Command("cmd", "/c", scriptPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start update script: %v", err)
	}
	
	// Exit the current process
	os.Exit(0)
	return nil
}

// ConfigUpdater implements configuration file update
type ConfigUpdater struct {
	configPath string
	backupPath string
	logger     *log.Logger
}

// NewConfigUpdater creates a new configuration updater
func NewConfigUpdater(configPath string, logger *log.Logger) Updater {
	return &ConfigUpdater{
		configPath: configPath,
		backupPath: configPath + ".backup",
		logger:     logger,
	}
}

// CanUpdate checks if update is possible
func (u *ConfigUpdater) CanUpdate() bool {
	// Check if we can write to the config file
	dir := filepath.Dir(u.configPath)
	
	testFile := filepath.Join(dir, ".config_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return false
	}
	os.Remove(testFile)
	
	return true
}

// PrepareUpdate prepares the configuration update
func (u *ConfigUpdater) PrepareUpdate(data []byte) error {
	// Backup current config if it exists
	if _, err := os.Stat(u.configPath); err == nil {
		configData, err := os.ReadFile(u.configPath)
		if err != nil {
			return fmt.Errorf("failed to read current config: %v", err)
		}
		
		if err := os.WriteFile(u.backupPath, configData, 0644); err != nil {
			return fmt.Errorf("failed to backup config: %v", err)
		}
	}
	
	// Write new config
	if err := os.WriteFile(u.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write new config: %v", err)
	}
	
	if u.logger != nil {
		u.logger.Printf("Configuration updated: %s", u.configPath)
	}
	
	return nil
}

// ExecuteUpdate applies the configuration update
func (u *ConfigUpdater) ExecuteUpdate() error {
	// Configuration is already updated in PrepareUpdate
	// This could trigger a configuration reload if needed
	if u.logger != nil {
		u.logger.Println("Configuration update applied successfully")
	}
	return nil
}

// Rollback rolls back the configuration
func (u *ConfigUpdater) Rollback() error {
	if _, err := os.Stat(u.backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist")
	}
	
	data, err := os.ReadFile(u.backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %v", err)
	}
	
	if err := os.WriteFile(u.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to restore backup: %v", err)
	}
	
	if u.logger != nil {
		u.logger.Println("Configuration rolled back to previous version")
	}
	
	return nil
}