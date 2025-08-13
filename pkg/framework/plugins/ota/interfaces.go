package ota

import (
	"context"
)

// Status represents OTA update status
type Status string

const (
	StatusIdle        Status = "idle"
	StatusDownloading Status = "downloading"
	StatusVerifying   Status = "verifying"
	StatusUpdating    Status = "updating"
	StatusRestarting  Status = "restarting"
	StatusFailed      Status = "failed"
)

// UpdateInfo contains firmware update information
type UpdateInfo struct {
	Version      string `json:"version"`
	URL          string `json:"url"`
	Size         uint32 `json:"size"`
	Digest       string `json:"sign"`
	DigestMethod string `json:"signMethod"`
	Description  string `json:"desc,omitempty"`
}

// UpdateResult represents the result of an OTA update
type UpdateResult struct {
	Success bool
	Message string
	Code    int
}

// ProgressCallback is called during download progress
type ProgressCallback func(current, total int64, percentage float64)

// StatusCallback is called when OTA status changes
type StatusCallback func(status Status, progress int32, message string)

// VersionProvider provides current firmware version
type VersionProvider interface {
	GetVersion() string
	SetVersion(version string) error
	GetModule() string
	SetModule(module string) error
}

// Downloader handles firmware download
type Downloader interface {
	Download(ctx context.Context, info *UpdateInfo, progress ProgressCallback) ([]byte, error)
	Verify(data []byte, info *UpdateInfo) error
}

// Updater handles firmware update execution
type Updater interface {
	CanUpdate() bool
	PrepareUpdate(data []byte) error
	ExecuteUpdate() error
	Rollback() error
}

// Manager manages the complete OTA process
type Manager interface {
	Start() error
	Stop() error
	GetCurrentVersion() string
	CheckUpdate() (*UpdateInfo, error)
	PerformUpdate(info *UpdateInfo) (*UpdateResult, error)
	SetStatusCallback(callback StatusCallback)
	SetAutoUpdate(enabled bool)
	GetStatus() Status
}