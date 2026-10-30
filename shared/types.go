package shared

import "time"

type HostMetrics struct {
	CpuUsage       float64      `json:"cpu_percent"`
	CpuLoad        float64      `json:"cpu_load"`
	RamUsedMB      uint64       `json:"ram_used_mb"`
	RamTotalMB     uint64       `json:"ram_total_mb"`
	RamPercent     float64      `json:"ram_percent"`
	DiskMetric     []DiskMetric `json:"disk_metrics"`
	CpuTemperature *float64     `json:"cpu_temperature,omitempty"`
	GpuMetrics     *GpuMetrics  `json:"gpu_metrics,omitempty"`
	Uptime         uint64       `json:"uptime"`
}

type ContainerMetrics struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Image         string  `json:"image"`
	State         string  `json:"state"`
	Health        string  `json:"health"`
	CpuPercent    float64 `json:"cpu_percent"`
	MemoryRssMB   float64 `json:"memory_rss_mb"`
	MemoryLimitMB float64 `json:"memory_limit_mb"`
}

type IncidentEvent struct {
	IncidentID  string    `json:"incident_id"`
	Timestamp   time.Time `json:"timestamp"`
	Severity    string    `json:"severity"`
	Node        string    `json:"node"`
	Container   string    `json:"container"`
	ExitCode    int       `json:"exit_code"`
	Reason      string    `json:"reason"`
	LogsSnippet string    `json:"logs_snippet"`
}

type NodeTelemetry struct {
	NodeID     string             `json:"node_id"`
	Hostname   string             `json:"hostname"`
	Timestamp  time.Time          `json:"timestamp"`
	PowerState PowerState         `json:"power_state"`
	Host       HostMetrics        `json:"host_metrics"`
	Containers []ContainerMetrics `json:"container_metrics"`
}

type PowerState string

const (
	PowerOnline     PowerState = "ONLINE"
	PowerSuspending PowerState = "SUSPENDING"
	PowerSuspended  PowerState = "SUSPENDED"
	PowerBooting    PowerState = "BOOTING"
)

type DiskMetric struct {
	Name       string  `json:"name"`
	UsedMB     uint64  `json:"used_mb"`
	TotalMB    uint64  `json:"total_mb"`
	Percent    float64 `json:"percent"`
	MountPoint string  `json:"mount_point"`
}

type GpuMetrics struct {
	Name        string   `json:"name"`
	Temperature *float64 `json:"temperature,omitempty"`
	LoadPercent *float64 `json:"load_percent,omitempty"`
	MemoryUsed  *uint64  `json:"memory_used_mb,omitempty"`
	MemoryTotal *uint64  `json:"memory_total_mb,omitempty"`
	MemoryUsage *float64 `json:"memory_usage_percent,omitempty"`
}
