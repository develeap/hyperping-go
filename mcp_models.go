// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

// ==================== Status & Reporting Models ====================

// StatusSummary mirrors the get_status_summary response shape probed
// against /v1/mcp tools/call on 2026-06-09. The pre-v0.7.1 struct
// declared only Total/Up/Down; Paused, Unknown, DownMonitors, and
// PausedMonitors were silently dropped at decode time because the
// fields did not exist on the Go type. Adding them is a wire-level
// additive change but counts as a feature surface change against the
// pre-v0.7.1 SDK because consumers now see them on the response.
type StatusSummary struct {
	Total           int      `json:"total"`
	Up              int      `json:"up"`
	Down            int      `json:"down"`
	Paused          int      `json:"paused"`
	Unknown         int      `json:"unknown"`
	DownMonitors    []string `json:"down_monitors"`
	PausedMonitors  []string `json:"paused_monitors"`
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
//
// v0.7.1 BREAKING: replaces the pre-v0.7.1 AlertHistory{Alerts, Total}
// declaration. The server's list_recent_alerts shape is bucketed counts
// plus a flat rawAlerts list:
//
//   {
//     "timeGroups": [{"time": "...", "count": N}, ...],
//     "totalAlerts": N,
//     "downAlerts": N,
//     "upAlerts": N,
//     "rawAlerts": [ ... ]
//   }
//
// The pre-v0.7.1 fields were never populated because the server never
// returned the keys "alerts" or "total". Probed against
// /v1/mcp tools/call list_recent_alerts on 2026-06-09 to capture the
// shape above.
//
// Total() is retained as a method so downstream consumers that read the
// scalar alert count keep compiling, but the underlying source is now
// the server's TotalAlerts field (zero-valued when rawAlerts is empty).
type AlertHistory struct {
	TimeGroups  []AlertTimeGroup `json:"timeGroups"`
	TotalAlerts int              `json:"totalAlerts"`
	DownAlerts  int              `json:"downAlerts"`
	UpAlerts    int              `json:"upAlerts"`
	RawAlerts   []Alert          `json:"rawAlerts"`
}

// Total reports the aggregate alert count for the response window. It
// mirrors the server's totalAlerts field rather than summing TimeGroups
// because the server already does that aggregation and downstream
// callers should not need to know whether the field exists at the top
// level or only as a derived sum.
func (a *AlertHistory) Total() int {
	if a == nil {
		return 0
	}
	return a.TotalAlerts
}

// AlertTimeGroup is one bucket of the timeGroups histogram returned by
// list_recent_alerts. Buckets stay populated even when rawAlerts is
// empty (e.g., the server returns zero-count buckets for every interval
// in a long window), so Count==0 buckets are valid data, not absence.
type AlertTimeGroup struct {
	Time  string `json:"time"`
	Count int    `json:"count"`
}

// Alert is one entry in the AlertHistory.RawAlerts slice. The pre-v0.7.1
// field tags reflect the historical typing; future probes against
// non-empty rawAlerts arrays should refine these to match the live
// shape. omitempty is preserved on fields whose population depends on
// the alert lifecycle (acknowledged-by, resolved-at).
type Alert struct {
	UUID           string `json:"uuid"`
	MonitorUUID    string `json:"monitor_uuid"`
	Status         string `json:"status"`
	TriggeredAt    string `json:"triggered_at"`
	AcknowledgedBy string `json:"acknowledged_by,omitempty"`
	ResolvedAt     string `json:"resolved_at,omitempty"`
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
//
// v0.7.1 BREAKING: replaces the pre-v0.7.1 TeamMember{Role,Status}
// declaration. The server returns accountRole (not role), no status
// field at all, plus phone/profilePictureUrl/ssoPictureUrl that were
// silently dropped at decode time. SsoPictureUrl is *string because
// the server returns JSON null for members who never signed in via
// SSO; differentiating "no SSO photo" from "" lets consumers render
// a placeholder accurately. Probed against /v1/mcp tools/call
// list_team_members on 2026-06-09.
type TeamMember struct {
	UUID              string  `json:"uuid"`
	Email             string  `json:"email"`
	Name              string  `json:"name"`
	Phone             string  `json:"phone,omitempty"`
	ProfilePictureURL string  `json:"profilePictureUrl,omitempty"`
	SsoPictureURL     *string `json:"ssoPictureUrl"`
	AccountRole       string  `json:"accountRole"`
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

// MCPRequestHeader is a single custom HTTP header sent with an MCP
// create_monitor or update_monitor request. The server declares this
// as an array of {name, value} objects under the request_headers arg.
type MCPRequestHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// MCPCreateMonitorRequest carries arguments for the MCP create_monitor tool.
// Field names and types match the server's inputSchema exactly. Required
// fields are Name and URL; all others are pointer types so nil values are
// omitted from the args map and the server applies its own defaults.
type MCPCreateMonitorRequest struct {
	Name               string             `json:"name"`
	URL                string             `json:"url"`
	Protocol           *string            `json:"protocol,omitempty"`           // "http", "icmp", "port", "dns"
	Port               *int               `json:"port,omitempty"`               // required when protocol="port"
	HTTPMethod         *string            `json:"http_method,omitempty"`         // "GET", "POST", etc.
	Regions            []string           `json:"regions,omitempty"`
	CheckFrequency     *float64           `json:"check_frequency,omitempty"`     // seconds
	FollowRedirects    *bool              `json:"follow_redirects,omitempty"`
	Timeout            *int               `json:"timeout,omitempty"`             // 1-60 seconds
	ExpectedStatusCode *string            `json:"expected_status_code,omitempty"` // "200", "2xx", "1xx-3xx"
	RequestBody        *string            `json:"request_body,omitempty"`
	RequestHeaders     []MCPRequestHeader `json:"request_headers,omitempty"`
	RequiredKeyword    *string            `json:"required_keyword,omitempty"`
	Paused             *bool              `json:"paused,omitempty"`
	AlertsWait         *float64           `json:"alerts_wait,omitempty"`         // minutes; -1 disables, 0 is immediate
	DNSRecordType      *string            `json:"dns_record_type,omitempty"`
	DNSNameserver      *string            `json:"dns_nameserver,omitempty"`
	DNSExpectedAnswer  *string            `json:"dns_expected_answer,omitempty"`
	EscalationPolicy   *string            `json:"escalation_policy,omitempty"`
	GroupID            any                `json:"group_id,omitempty"` // int or string
}

// MCPUpdateMonitorRequest carries arguments for the MCP update_monitor tool.
// The UUID is passed as a separate method argument, not embedded here. All
// fields are optional; nil values are omitted so only explicitly changed
// fields reach the server.
type MCPUpdateMonitorRequest struct {
	Name               *string            `json:"name,omitempty"`
	URL                *string            `json:"url,omitempty"`
	Protocol           *string            `json:"protocol,omitempty"`
	Port               *int               `json:"port,omitempty"`
	HTTPMethod         *string            `json:"http_method,omitempty"`
	Regions            []string           `json:"regions,omitempty"`
	CheckFrequency     *float64           `json:"check_frequency,omitempty"`
	FollowRedirects    *bool              `json:"follow_redirects,omitempty"`
	Timeout            *int               `json:"timeout,omitempty"`
	ExpectedStatusCode *string            `json:"expected_status_code,omitempty"`
	RequestBody        *string            `json:"request_body,omitempty"`
	RequestHeaders     []MCPRequestHeader `json:"request_headers,omitempty"`
	RequiredKeyword    *string            `json:"required_keyword,omitempty"`
	Paused             *bool              `json:"paused,omitempty"`
	AlertsWait         *float64           `json:"alerts_wait,omitempty"`
	DNSRecordType      *string            `json:"dns_record_type,omitempty"`
	DNSNameserver      *string            `json:"dns_nameserver,omitempty"`
	DNSExpectedAnswer  *string            `json:"dns_expected_answer,omitempty"`
	EscalationPolicy   *string            `json:"escalation_policy,omitempty"`
	GroupID            any                `json:"group_id,omitempty"`
}

// ==================== Monitor Models (MCP-specific) ====================
// Uses existing Monitor from models_monitor.go