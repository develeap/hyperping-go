// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

// ==================== Status & Reporting Models ====================

type StatusSummary struct {
	Total int `json:"total"`
	Up    int `json:"up"`
	Down  int `json:"down"`
}

type ResponseTimeReport struct {
	UUID string  `json:"uuid"`
	Avg  float64 `json:"avg"`
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
}

type MttaReport struct {
	UUID      string  `json:"uuid,omitempty"`
	AvgWait   float64 `json:"avg_wait"`
	MinWait   float64 `json:"min_wait"`
	MaxWait   float64 `json:"max_wait"`
	TotalAlerts int   `json:"total_alerts"`
	Acknowledged int `json:"acknowledged"`
}

type MttrReport struct {
	UUID      string  `json:"uuid,omitempty"`
	AvgResolve float64 `json:"avg_resolve"`
	MinResolve float64 `json:"min_resolve"`
	MaxResolve float64 `json:"max_resolve"`
	TotalOutages int   `json:"total_outages"`
	Resolved    int   `json:"resolved"`
}

// ==================== Observability Models ====================

type MonitorAnomaly struct {
	UUID        string `json:"uuid"`
	MonitorUUID string `json:"monitor_uuid"`
	DetectedAt  string `json:"detected_at"`
	Score      float64 `json:"score"`
	Type       string `json:"type"`
}

type ProbeLogResponse struct {
	UUID   string      `json:"uuid"`
	Status int        `json:"status"`
	Logs   []ProbeLog  `json:"logs"`
}

type ProbeLog struct {
	Timestamp string `json:"timestamp"`
	Status   int     `json:"status"`
	Response int    `json:"response"`
}

// ==================== Alert Models ====================

type AlertHistory struct {
	Alerts []Alert `json:"alerts"`
	Total  int     `json:"total"`
}

type Alert struct {
	UUID        string `json:"uuid"`
	MonitorUUID string `json:"monitor_uuid"`
	Status     string `json:"status"`
	TriggeredAt string `json:"triggered_at"`
	AcknowledgedBy string `json:"acknowledged_by,omitempty"`
	ResolvedAt string `json:"resolved_at,omitempty"`
}

// ==================== On-Call Models ====================

type OnCallSchedule struct {
	UUID   string          `json:"uuid"`
	Name  string          `json:"name"`
	Team  string          `json:"team"`
	CurrentOncall string   `json:"current_oncall"`
	NextOncall   string   `json:"next_oncall"`
	RotationStart string   `json:"rotation_start"`
	RotationEnd string   `json:"rotation_end"`
}

// ==================== Escalation Models ====================

type EscalationPolicy struct {
	UUID   string             `json:"uuid"`
	Name  string             `json:"name"`
	Team  string             `json:"team"`
	Steps []EscalationStep  `json:"steps"`
}

type EscalationStep struct {
	Delay    int    `json:"delay"`
	TargetType string `json:"target_type"` // user, schedule, integration
	TargetID string `json:"target_id"`
}

// ==================== Team Models ====================

type TeamMember struct {
	UUID    string `json:"uuid"`
	Email   string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
	Status string `json:"status"`
}

// ==================== Integration Models ====================

type Integration struct {
	UUID         string `json:"uuid"`
	Name        string `json:"name"`
	Type        string `json:"type"` // email, slack, webhook, pagerduty
	Enabled     bool   `json:"enabled"`
	LastTestAt  string `json:"last_test_at,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// ==================== Outage Models ====================

type OutageTimeline struct {
	UUID      string      `json:"uuid"`
	MonitorUUID string    `json:"monitor_uuid"`
	Events   []OutageEvent `json:"events"`
}

type OutageEvent struct {
	Timestamp string `json:"timestamp"`
	Status   string `json:"status"` // started, acknowledged, resolved
	User      string `json:"user,omitempty"`
	Note     string `json:"note,omitempty"`
}

// ==================== Uptime Models ====================

type UptimeReport struct {
	UUID       string  `json:"uuid,omitempty"`
	Uptime     float64 `json:"uptime"`      // percentage
	TotalDays  int     `json:"total_days"`
	UptimeDays int     `json:"uptime_days"`
}

// ==================== Outage Models ====================

type MCPOutage struct {
	UUID          string `json:"uuid"`
	MonitorUUID   string `json:"monitor_uuid"`
	Status       string `json:"status"` // active, resolved
	StartedAt    string `json:"started_at"`
	AcknowledgedAt string `json:"acknowledged_at,omitempty"`
	ResolvedAt   string `json:"resolved_at,omitempty"`
	Duration     int    `json:"duration"` // seconds
}

type OutageList struct {
	Outages []MCPOutage `json:"outages"`
	Total  int      `json:"total"`
}

// ==================== Monitor List Models ====================

type MonitorList struct {
	Monitors []Monitor `json:"monitors"`
	Total   int       `json:"total"`
	Page    int      `json:"page"`
	Limit   int      `json:"limit"`
}

// ==================== Monitor Detail Models ====================

type MonitorDetails struct {
	UUID            string              `json:"uuid"`
	Name            string              `json:"name"`
	URL             string              `json:"url"`
	Status         string              `json:"status"` // up, down, paused, ssl_expiring
	Method         string              `json:"method"`
	Frequency      int                `json:"frequency"`
	ExpectedStatus int                `json:"expected_status"`
	Regions        []string           `json:"regions"`
	CreatedAt      string             `json:"created_at"`
	UpdatedAt      string             `json:"updated_at"`
	Tags           []string           `json:"tags,omitempty"`
	EscalationPolicy *string          `json:"escalation_policy,omitempty"`
	CustomHeaders  map[string]string `json:"custom_headers,omitempty"`
}

// ==================== Write Request Models ====================

type MCPCreateMonitorRequest struct {
	Name            string            `json:"name"`
	URL             string            `json:"url,omitempty"`
	Method          string            `json:"method,omitempty"` // HTTP, ICMP, PORT, DNS
	Frequency       int               `json:"frequency,omitempty"` // seconds (10s to 24h)
	ExpectedStatus  int               `json:"expected_status,omitempty"`
	Regions         []string         `json:"regions,omitempty"`
	Keyword         string            `json:"keyword,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	EscalationPolicy string           `json:"escalation_policy,omitempty"`
}

type MCPUpdateMonitorRequest struct {
	Name            string            `json:"name,omitempty"`
	URL             string            `json:"url,omitempty"`
	Method          string            `json:"method,omitempty"`
	Frequency       int               `json:"frequency,omitempty"`
	ExpectedStatus  int               `json:"expected_status,omitempty"`
	Regions         []string         `json:"regions,omitempty"`
	Keyword         string            `json:"keyword,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	EscalationPolicy string           `json:"escalation_policy,omitempty"`
}

// ==================== Monitor Models (MCP-specific) ====================
// Uses existing Monitor from models_monitor.go