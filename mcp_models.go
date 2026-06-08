// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

// ==================== Status & Reporting Models ====================

type StatusSummary struct {
	Total int `json:"total"`
	Up    int `json:"up"`
	Down  int `json:"down"`
}

// ==================== MTTA (windowed) ====================
//
// v0.7.0 BREAKING: replaces MttaReport. The pre-v0.7.0 struct modeled
// fields that the server never returned, so every call decoded zero
// values silently. The shape below matches the live MCP server
// (probed 2026-06-08 against /v1/mcp tools/call get_monitor_mtta):
//
//   {
//     "monitors": [ ... per-monitor entries when alerts present ... ],
//     "totalAcknowledged": N,
//     "mtta": <seconds>
//   }
//
// Per-monitor MTTA entries mirror the MTTR per-monitor shape for
// forward-compatibility; the server returns an empty array when there
// are no acknowledged alerts in the window.
type MonitorMttaResponse struct {
	Monitors          []MonitorMttaEntry `json:"monitors"`
	TotalAcknowledged int                `json:"totalAcknowledged"`
	Mtta              float64            `json:"mtta"`
}

type MonitorMttaEntry struct {
	UUID         string  `json:"uuid"`
	Name         string  `json:"name"`
	Protocol     string  `json:"protocol"`
	Acknowledged int     `json:"acknowledged"`
	Mtta         float64 `json:"mtta"`
}

// ==================== MTTR (windowed) ====================
//
// v0.7.0 BREAKING: replaces MttrReport.
type MonitorMttrResponse struct {
	Monitors           []MonitorMttrEntry `json:"monitors"`
	TotalOutages       int                `json:"totalOutages"`
	TotalOutagesLength int                `json:"totalOutagesLength"`
	Mttr               float64            `json:"mttr"`
	Mtta               float64            `json:"mtta"`
}

type MonitorMttrEntry struct {
	UUID          string  `json:"uuid"`
	Name          string  `json:"name"`
	Protocol      string  `json:"protocol"`
	OutageCount   int     `json:"outageCount"`
	TotalDowntime float64 `json:"totalDowntime"`
	Mttr          float64 `json:"mttr"`
	Mtta          float64 `json:"mtta"`
	LongestOutage float64 `json:"longestOutage"`
}

// ==================== Response time (windowed) ====================
//
// v0.7.0 BREAKING: replaces ResponseTimeReport.
type MonitorResponseTimeResponse struct {
	TimeGroups      []ResponseTimeGroup           `json:"timeGroups"`
	AvgResponseTime float64                       `json:"avgResponseTime"`
	P95ResponseTime float64                       `json:"p95ResponseTime"`
	Monitors        []MonitorResponseTimeEntry    `json:"monitors"`
}

type ResponseTimeGroup struct {
	Time            string  `json:"time"`
	AvgResponseTime float64 `json:"avgResponseTime"`
	Count           int     `json:"count"`
}

// AvgResponseTimeByRegion uses *float64 because the server returns null for
// regions that recorded no probes in the window. Differentiating "no data"
// from "0 ms" matters for downstream metric emission.
type MonitorResponseTimeEntry struct {
	UUID                    string              `json:"uuid"`
	Name                    string              `json:"name"`
	Protocol                string              `json:"protocol"`
	AvgResponseTime         float64             `json:"avgResponseTime"`
	AvgResponseTimeByRegion map[string]*float64 `json:"avgResponseTimeByRegion"`
	Count                   int                 `json:"count"`
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

// ==================== Uptime (windowed) ====================
//
// v0.7.0 BREAKING: replaces UptimeReport. Note that the server returns
// the top-level AverageUptime as a string ("100%") and the per-monitor
// AverageUptime as a number — the types here reflect that asymmetry.
type MonitorUptimeResponse struct {
	Monitors           []MonitorUptimeEntry `json:"monitors"`
	PeriodAverages     []UptimePeriod       `json:"periodAverages"`
	TotalOutages       int                  `json:"totalOutages"`
	TotalOutagesLength int                  `json:"totalOutagesLength"`
	MTTR               float64              `json:"MTTR"`
	AverageUptime      string               `json:"averageUptime"`
}

type MonitorUptimeEntry struct {
	UUID          string         `json:"uuid"`
	Name          string         `json:"name"`
	Protocol      string         `json:"protocol"`
	UptimePeriods []UptimePeriod `json:"uptimePeriods"`
	AverageUptime float64        `json:"averageUptime"`
	OutageCount   int            `json:"outageCount"`
	TotalDowntime float64        `json:"totalDowntime"`
	Mttr          float64        `json:"mttr"`
	LongestOutage float64        `json:"longestOutage"`
}

type UptimePeriod struct {
	Date   string  `json:"date"`
	Uptime float64 `json:"uptime"`
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