package schema

import "time"

type Status string

const (
	StatusAvailable   Status = "available"
	StatusUnsupported Status = "unsupported"
	StatusUnavailable Status = "unavailable"
)

type Scope string

const (
	ScopeHost        Scope = "host"
	ScopeContainer   Scope = "container"
	ScopeUnavailable Scope = "unavailable"
)

type SectionStatus struct {
	Status Status `json:"status"`
	Scope  Scope  `json:"scope"`
	Error  string `json:"error,omitempty"`
}

type TargetIdentity struct {
	Hostname     string `json:"hostname"`
	Platform     string `json:"platform"`
	Architecture string `json:"architecture"`
	MachineID    string `json:"machine_id,omitempty"`
}

type Snapshot struct {
	Timestamp     time.Time      `json:"timestamp"`
	SchemaVersion string         `json:"schema_version"`
	Target        TargetIdentity `json:"target"`
	System        SystemSection  `json:"system"`
	CPU           CPUSection     `json:"cpu"`
	Memory        MemorySection  `json:"memory"`
	Disks         DisksSection   `json:"disks"`
	GPU           GPUSection     `json:"gpu"`
}

type Health struct {
	Status           string    `json:"status"`
	SchemaVersion    string    `json:"schema_version"`
	CollectionStatus string    `json:"collection_status"`
	Timestamp        time.Time `json:"timestamp,omitempty"`
	LastError        string    `json:"last_error,omitempty"`
}

type SystemSection struct {
	SectionStatus
	UptimeSeconds *uint64      `json:"uptime_seconds"`
	LoadAverage   *LoadAverage `json:"load_average"`
}

type LoadAverage struct {
	OneMinute      float64 `json:"one_minute"`
	FiveMinutes    float64 `json:"five_minutes"`
	FifteenMinutes float64 `json:"fifteen_minutes"`
}

type CPUSection struct {
	SectionStatus
	Model              string    `json:"model,omitempty"`
	CoreCount          int       `json:"core_count"`
	Utilization        *float64  `json:"utilization_percent"`
	PerCoreUtilization []float64 `json:"per_core_utilization_percent,omitempty"`
}

type MemorySection struct {
	SectionStatus
	TotalBytes     *uint64  `json:"total_bytes"`
	UsedBytes      *uint64  `json:"used_bytes"`
	FreeBytes      *uint64  `json:"free_bytes"`
	AvailableBytes *uint64  `json:"available_bytes"`
	PercentUsed    *float64 `json:"percent_used"`
}

type DisksSection struct {
	SectionStatus
	Volumes []Volume `json:"volumes"`
}

type Volume struct {
	MountPoint     string      `json:"mount_point"`
	FilesystemType string      `json:"filesystem_type,omitempty"`
	TotalBytes     *uint64     `json:"total_bytes"`
	UsedBytes      *uint64     `json:"used_bytes"`
	FreeBytes      *uint64     `json:"free_bytes"`
	PercentUsed    *float64    `json:"percent_used"`
	Health         *DiskHealth `json:"health,omitempty"`
}

type DiskHealth struct {
	Status      string   `json:"status"`
	Temperature *float64 `json:"temperature_celsius,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}

type GPUSection struct {
	SectionStatus
	Devices []GPUDevice `json:"devices,omitempty"`
}

type GPUDevice struct {
	Vendor             string   `json:"vendor,omitempty"`
	Model              string   `json:"model,omitempty"`
	MemoryTotalBytes   *uint64  `json:"memory_total_bytes"`
	MemoryUsedBytes    *uint64  `json:"memory_used_bytes"`
	Utilization        *float64 `json:"utilization_percent"`
	TemperatureCelsius *float64 `json:"temperature_celsius"`
}
