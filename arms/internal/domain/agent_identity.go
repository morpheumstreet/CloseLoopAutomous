package domain

import (
	"strings"
	"time"
)

// AgentIdentity is the canonical representation of a gateway-facing agent profile (see docs/scan-agents.md).
type AgentIdentity struct {
	ID           string            `json:"id"`
	GatewayURL   string            `json:"gateway_url"`
	Name         string            `json:"name"`
	Driver       string            `json:"driver"`
	Version      string            `json:"version,omitempty"`
	Status       AgentStatus       `json:"status"`
	LastSeen     time.Time         `json:"last_seen"`
	Capabilities []string          `json:"capabilities,omitempty"`
	Platform     PlatformInfo      `json:"platform"`
	Metrics      Metrics           `json:"metrics"`
	SubAgents    []SubAgentRef     `json:"sub_agents,omitempty"`
	Geo          *GeoLocation      `json:"geo,omitempty"`
	Custom       map[string]any    `json:"custom,omitempty"`
}

// AgentStatus is runtime state for an identity row (synthesized or live).
type AgentStatus string

const (
	StatusOnline        AgentStatus = "online"
	StatusOffline       AgentStatus = "offline"
	StatusUnauthorized  AgentStatus = "unauthorized" // HTTP 401/403 or missing credentials when the driver expects them — not the same as offline/unreachable.
	StatusError         AgentStatus = "error"
	StatusBusy          AgentStatus = "busy"
)

// PlatformInfo describes host / OS context when reported or inferred.
type PlatformInfo struct {
	OS       string `json:"os,omitempty"`
	Arch     string `json:"arch,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	GPU      string `json:"gpu,omitempty"`
}

// Metrics holds coarse resource telemetry when available.
type Metrics struct {
	CPUPercent  float64 `json:"cpu_percent,omitempty"`
	MemUsedMB   int64   `json:"mem_used_mb,omitempty"`
	MemTotalMB  int64   `json:"mem_total_mb,omitempty"`
	DiskUsedGB  int64   `json:"disk_used_gb,omitempty"`
	DiskTotalGB int64   `json:"disk_total_gb,omitempty"`
	GPUMemMB    int64   `json:"gpu_mem_mb,omitempty"`
}

// SubAgentRef is a child or secondary agent reported by a multi-agent gateway.
type SubAgentRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role,omitempty"`
}

// GeoLocation is city-level geolocation from offline GeoIP or manual entry.
type GeoLocation struct {
	Latitude    float64   `json:"latitude,omitempty"`
	Longitude   float64   `json:"longitude,omitempty"`
	City        string    `json:"city,omitempty"`
	Region      string    `json:"region,omitempty"`
	Country     string    `json:"country,omitempty"`
	CountryISO  string    `json:"country_iso,omitempty"`
	AccuracyKM  int       `json:"accuracy_km,omitempty"`
	Source      string    `json:"source"` // maxmind_geoip2, manual, none
	LastUpdated time.Time `json:"last_updated"`
}

// StableAgentProfileID returns a stable primary key for one logical identity per gateway endpoint (MVP).
func StableAgentProfileID(endpointID, deviceID string) string {
	d := strings.TrimSpace(deviceID)
	if d == "" {
		d = "default"
	}
	return endpointID + ":" + d
}
