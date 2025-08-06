package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/iot-go-sdk/pkg/config"
	"github.com/iot-go-sdk/pkg/mqtt"
	"github.com/iot-go-sdk/pkg/ota"
)

// Current firmware version - should be read from device config in production
const currentVersion = "1.0.2"

// File to save downloaded firmware - in production, write to flash
const firmwareFile = "firmware.bin"

func main() {
	cfg := config.NewConfig()

	// Configure device credentials
	cfg.Device.ProductKey = "QLTMkOfW"
	cfg.Device.DeviceName = "WjJjXbP0X1"
	cfg.Device.DeviceSecret = "Vt1OU489RAylT8MV"

	// Configure MQTT connection
	cfg.MQTT.Host = "121.41.43.80"
	cfg.MQTT.Port = 8883
	cfg.MQTT.UseTLS = true
	cfg.TLS.SkipVerify = true // Skip certificate verification for self-signed cert

	// Create MQTT client
	mqttClient := mqtt.NewClient(cfg)

	// Connect to MQTT broker
	if err := mqttClient.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer mqttClient.Disconnect()

	log.Println("Connected to MQTT broker successfully")

	// Create OTA client
	otaClient := ota.NewClient(mqttClient, cfg.Device.ProductKey, cfg.Device.DeviceName)

	// Variable to store firmware data during download
	var firmwareData []byte
	var lastPercent int

	// Set OTA message handler
	otaClient.SetRecvHandler(func(client *ota.Client, recvType ota.RecvType, task *ota.TaskDesc) {
		switch recvType {
		case ota.RecvTypeFOTA:
			log.Printf("Received FOTA task:")
			log.Printf("  Version: %s", task.Version)
			log.Printf("  Size: %d bytes", task.Size)
			log.Printf("  URL: %s", task.URL)
			log.Printf("  Digest: %s", task.ExpectDigest)

			if task.ExtraData != "" {
				log.Printf("  Extra data: %s", task.ExtraData)
			}

			// Reset firmware data
			firmwareData = make([]byte, 0, task.Size)
			lastPercent = 0

			// Start downloading firmware
			go func() {
				ctx := context.Background()

				// Download complete firmware at once
				log.Printf("Downloading firmware (%d bytes)...", task.Size)

				if err := client.Download(ctx, task, 0, 0); err != nil {
					log.Printf("Failed to download firmware: %v", err)
					// Report failure to cloud
					client.ReportProgress("-2", "Download failed", -2, task.Module)
					return
				}

				log.Printf("Download completed successfully")

				// Report 100% download complete
				if err := client.ReportProgress("100", "Download completed", 100, task.Module); err != nil {
					log.Printf("Failed to report download completion: %v", err)
				}

				// Save firmware to file (in production, write to flash)
				if err := os.WriteFile(firmwareFile, firmwareData, 0644); err != nil {
					log.Printf("Failed to save firmware: %v", err)
					// Report burn failure (-4 means burn failed according to C SDK)
					client.ReportProgress("-4", "Burn failed", -4, task.Module)
					return
				}

				log.Printf("Firmware saved to %s", firmwareFile)

				// In production, you would:
				// 1. Verify the firmware integrity
				// 2. Save current state for rollback
				// 3. Reboot device with new firmware
				// 4. Boot with new firmware
				// 5. After successful boot, report new version

				// Simulate upgrade process
				log.Printf("Simulating firmware upgrade...")

				// For demo, immediately report the new version (in production, this would be after reboot)
				log.Printf("Upgrade successful, reporting new version: %s (module: %s)", task.Version, task.Module)
				if err := client.ReportVersionWithModule(task.Version, task.Module); err != nil {
					log.Printf("Failed to report new version: %v", err)
					// Report upgrade failure (-1 means upgrade failed according to C SDK)
					client.ReportProgress("-1", "Upgrade failed", -1, task.Module)
				} else {
					log.Printf("Successfully reported new version to IoT platform")
				}
			}()

		case ota.RecvTypeCOTA:
			log.Printf("Received COTA (configuration) task")
			// Handle remote configuration update
		}
	})

	// Set download progress handler
	otaClient.SetDownloadHandler(func(percent int, data []byte, err error) {
		if err != nil {
			log.Printf("Download error: %v", err)
			// Don't report error here, let the main download function handle it
			return
		}

		// Append data to firmware buffer
		if data != nil {
			firmwareData = append(firmwareData, data...)
		}

		// Report progress every 5% or at 100%
		if percent-lastPercent >= 5 {
			log.Printf("Download progress: %d%% (%d bytes)", percent, len(firmwareData))

			// Report progress to cloud (C SDK reports during download, not at 100%)
			if err := otaClient.ReportProgress(fmt.Sprintf("%d", percent), "Downloading", percent, ""); err != nil {
				log.Printf("Failed to report progress: %v", err)
			}

			lastPercent = percent
		}
	})

	// Start OTA client
	if err := otaClient.Start(); err != nil {
		log.Fatalf("Failed to start OTA client: %v", err)
	}
	defer otaClient.Stop()

	log.Println("OTA client started")

	// Report current version
	log.Printf("Reporting current version: %s", currentVersion)
	if err := otaClient.ReportVersion(currentVersion); err != nil {
		log.Printf("Failed to report version: %v", err)
	}

	// Query for available firmware updates
	log.Println("Querying for firmware updates...")
	if err := otaClient.QueryFirmware(); err != nil {
		log.Printf("Failed to query firmware: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
}
