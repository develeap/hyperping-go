// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"context"
	"encoding/json"
)

// MCPTransport defines the interface for MCP transport
type MCPTransport interface {
	Initialize(ctx context.Context) (map[string]any, error)
	CallTool(ctx context.Context, toolName string, args map[string]any) (any, error)
}

// MCPClient is a high-level client for Hyperping MCP server tools
type MCPClient struct {
	transport MCPTransport
}

// NewMCPClient creates a new MCP client
func NewMCPClient(transport MCPTransport) *MCPClient {
	return &MCPClient{
		transport: transport,
	}
}

// ==================== Status & Reporting ====================

// GetStatusSummary returns aggregate monitor status counts
func (c *MCPClient) GetStatusSummary(ctx context.Context) (*StatusSummary, error) {
	result, err := c.transport.CallTool(ctx, "get_status_summary", nil)
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var summary StatusSummary
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &summary)
	}
	return &summary, nil
}

// GetMonitorResponseTime returns response time metrics for a monitor
func (c *MCPClient) GetMonitorResponseTime(ctx context.Context, monitorUUID string) (*ResponseTimeReport, error) {
	result, err := c.transport.CallTool(ctx, "get_monitor_response_time", map[string]any{"uuid": monitorUUID})
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var report ResponseTimeReport
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &report)
	}
	return &report, nil
}

// GetMonitorMtta returns mean time to acknowledge metrics
func (c *MCPClient) GetMonitorMtta(ctx context.Context, monitorUUID string) (*MttaReport, error) {
	args := map[string]any{}
	if monitorUUID != "" {
		args["uuid"] = monitorUUID
	}

	result, err := c.transport.CallTool(ctx, "get_monitor_mtta", args)
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var report MttaReport
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &report)
	}
	return &report, nil
}

// GetMonitorMttr returns mean time to resolve metrics
func (c *MCPClient) GetMonitorMttr(ctx context.Context, monitorUUID string) (*MttrReport, error) {
	args := map[string]any{}
	if monitorUUID != "" {
		args["uuid"] = monitorUUID
	}

	result, err := c.transport.CallTool(ctx, "get_monitor_mttr", args)
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var report MttrReport
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &report)
	}
	return &report, nil
}

// ==================== Observability ====================

// GetMonitorAnomalies returns anomalies detected for a monitor
func (c *MCPClient) GetMonitorAnomalies(ctx context.Context, monitorUUID string) ([]MonitorAnomaly, error) {
	result, err := c.transport.CallTool(ctx, "get_monitor_anomalies", map[string]any{"uuid": monitorUUID})
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	anomaliesData, ok := data["anomalies"].([]any)
	if !ok {
		return nil, nil
	}

	var anomalies []MonitorAnomaly
	for _, a := range anomaliesData {
		amap, ok := a.(map[string]any)
		if !ok {
			continue
		}
		var anomaly MonitorAnomaly
		if bytes, err := json.Marshal(amap); err == nil {
			json.Unmarshal(bytes, &anomaly)
		}
		anomalies = append(anomalies, anomaly)
	}
	return anomalies, nil
}

// GetMonitorHttpLogs returns HTTP probe logs for a monitor
func (c *MCPClient) GetMonitorHttpLogs(ctx context.Context, monitorUUID string) (*ProbeLogResponse, error) {
	result, err := c.transport.CallTool(ctx, "get_monitor_http_logs", map[string]any{"uuid": monitorUUID})
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var response ProbeLogResponse
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &response)
	}
	return &response, nil
}

// ==================== Alerts ====================

// ListRecentAlerts returns recent alert notifications
func (c *MCPClient) ListRecentAlerts(ctx context.Context) (*AlertHistory, error) {
	result, err := c.transport.CallTool(ctx, "list_recent_alerts", nil)
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var history AlertHistory
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &history)
	}
	return &history, nil
}

// ==================== On-Call ====================

// ListOnCallSchedules returns all on-call schedules
func (c *MCPClient) ListOnCallSchedules(ctx context.Context) ([]OnCallSchedule, error) {
	result, err := c.transport.CallTool(ctx, "list_on_call_schedules", nil)
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	schedulesData, ok := data["schedules"].([]any)
	if !ok {
		return nil, nil
	}

	var schedules []OnCallSchedule
	for _, s := range schedulesData {
		smap, ok := s.(map[string]any)
		if !ok {
			continue
		}
		var schedule OnCallSchedule
		if bytes, err := json.Marshal(smap); err == nil {
			json.Unmarshal(bytes, &schedule)
		}
		schedules = append(schedules, schedule)
	}
	return schedules, nil
}

// GetOnCallSchedule returns a single on-call schedule
func (c *MCPClient) GetOnCallSchedule(ctx context.Context, uuid string) (*OnCallSchedule, error) {
	result, err := c.transport.CallTool(ctx, "get_on_call_schedule", map[string]any{"uuid": uuid})
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var schedule OnCallSchedule
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &schedule)
	}
	return &schedule, nil
}

// ==================== Escalation Policies ====================

// ListEscalationPolicies returns all escalation policies
func (c *MCPClient) ListEscalationPolicies(ctx context.Context) ([]EscalationPolicy, error) {
	result, err := c.transport.CallTool(ctx, "list_escalation_policies", nil)
	if err != nil {
		return nil, err
	}

	list, ok := result.([]any)
	if !ok {
		return nil, nil
	}

	var policies []EscalationPolicy
	for _, p := range list {
		pmap, ok := p.(map[string]any)
		if !ok {
			continue
		}
		var policy EscalationPolicy
		if bytes, err := json.Marshal(pmap); err == nil {
			json.Unmarshal(bytes, &policy)
		}
		policies = append(policies, policy)
	}
	return policies, nil
}

// GetEscalationPolicy returns a single escalation policy
func (c *MCPClient) GetEscalationPolicy(ctx context.Context, uuid string) (*EscalationPolicy, error) {
	result, err := c.transport.CallTool(ctx, "get_escalation_policy", map[string]any{"uuid": uuid})
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var policy EscalationPolicy
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &policy)
	}
	return &policy, nil
}

// ==================== Team ====================

// ListTeamMembers returns all team members
func (c *MCPClient) ListTeamMembers(ctx context.Context) ([]TeamMember, error) {
	result, err := c.transport.CallTool(ctx, "list_team_members", nil)
	if err != nil {
		return nil, err
	}

	list, ok := result.([]any)
	if !ok {
		return nil, nil
	}

	var members []TeamMember
	for _, m := range list {
		mmap, ok := m.(map[string]any)
		if !ok {
			continue
		}
		var member TeamMember
		if bytes, err := json.Marshal(mmap); err == nil {
			json.Unmarshal(bytes, &member)
		}
		members = append(members, member)
	}
	return members, nil
}

// ==================== Integrations ====================

// ListIntegrations returns all notification channel integrations
func (c *MCPClient) ListIntegrations(ctx context.Context) ([]Integration, error) {
	result, err := c.transport.CallTool(ctx, "list_integrations", nil)
	if err != nil {
		return nil, err
	}

	list, ok := result.([]any)
	if !ok {
		return nil, nil
	}

	var integrations []Integration
	for _, i := range list {
		imap, ok := i.(map[string]any)
		if !ok {
			continue
		}
		var integration Integration
		if bytes, err := json.Marshal(imap); err == nil {
			json.Unmarshal(bytes, &integration)
		}
		integrations = append(integrations, integration)
	}
	return integrations, nil
}

// GetIntegration returns a single integration
func (c *MCPClient) GetIntegration(ctx context.Context, uuid string) (*Integration, error) {
	result, err := c.transport.CallTool(ctx, "get_integration", map[string]any{"uuid": uuid})
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var integration Integration
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &integration)
	}
	return &integration, nil
}

// ==================== Outages ====================

// ListMonitors returns paginated list of monitors with optional status filter
func (c *MCPClient) ListMonitors(ctx context.Context, status string, page, limit int) (*MonitorList, error) {
	args := map[string]any{}
	if status != "" {
		args["status"] = status
	}
	if page > 0 {
		args["page"] = page
	}
	if limit > 0 {
		args["limit"] = limit
	}

	result, err := c.transport.CallTool(ctx, "list_monitors", args)
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var list MonitorList
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &list)
	}
	return &list, nil
}

// GetMonitor returns a single monitor by UUID
func (c *MCPClient) GetMonitor(ctx context.Context, uuid string) (*MonitorDetails, error) {
	result, err := c.transport.CallTool(ctx, "get_monitor", map[string]any{"uuid": uuid})
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var monitor MonitorDetails
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &monitor)
	}
	return &monitor, nil
}

// SearchMonitorsByName searches monitors by name
func (c *MCPClient) SearchMonitorsByName(ctx context.Context, query string) ([]Monitor, error) {
	result, err := c.transport.CallTool(ctx, "search_monitors_by_name", map[string]any{"query": query})
	if err != nil {
		return nil, err
	}

	list, ok := result.([]any)
	if !ok {
		return nil, nil
	}

	var monitors []Monitor
	for _, m := range list {
		mmap, ok := m.(map[string]any)
		if !ok {
			continue
		}
		var monitor Monitor
		if bytes, err := json.Marshal(mmap); err == nil {
			json.Unmarshal(bytes, &monitor)
		}
		monitors = append(monitors, monitor)
	}
	return monitors, nil
}

// CreateMonitor creates a new monitor
func (c *MCPClient) CreateMonitor(ctx context.Context, req MCPCreateMonitorRequest) (*MonitorDetails, error) {
	args := map[string]any{}
	if req.Name != "" {
		args["name"] = req.Name
	}
	if req.URL != "" {
		args["url"] = req.URL
	}
	if req.Method != "" {
		args["method"] = req.Method
	}
	if req.Frequency > 0 {
		args["frequency"] = req.Frequency
	}
	if req.ExpectedStatus > 0 {
		args["expected_status"] = req.ExpectedStatus
	}
	if len(req.Regions) > 0 {
		args["regions"] = req.Regions
	}
	if req.Keyword != "" {
		args["keyword"] = req.Keyword
	}
	if len(req.Headers) > 0 {
		args["headers"] = req.Headers
	}
	if req.EscalationPolicy != "" {
		args["escalation_policy"] = req.EscalationPolicy
	}

	result, err := c.transport.CallTool(ctx, "create_monitor", args)
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var monitor MonitorDetails
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &monitor)
	}
	return &monitor, nil
}

// UpdateMonitor updates an existing monitor
func (c *MCPClient) UpdateMonitor(ctx context.Context, uuid string, req MCPUpdateMonitorRequest) (*MonitorDetails, error) {
	args := map[string]any{"uuid": uuid}
	if req.Name != "" {
		args["name"] = req.Name
	}
	if req.URL != "" {
		args["url"] = req.URL
	}
	if req.Method != "" {
		args["method"] = req.Method
	}
	if req.Frequency > 0 {
		args["frequency"] = req.Frequency
	}
	if req.ExpectedStatus > 0 {
		args["expected_status"] = req.ExpectedStatus
	}
	if len(req.Regions) > 0 {
		args["regions"] = req.Regions
	}
	if req.Keyword != "" {
		args["keyword"] = req.Keyword
	}
	if len(req.Headers) > 0 {
		args["headers"] = req.Headers
	}
	if req.EscalationPolicy != "" {
		args["escalation_policy"] = req.EscalationPolicy
	}

	result, err := c.transport.CallTool(ctx, "update_monitor", args)
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var monitor MonitorDetails
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &monitor)
	}
	return &monitor, nil
}

// PauseMonitor pauses a monitor
func (c *MCPClient) PauseMonitor(ctx context.Context, uuid string) error {
	_, err := c.transport.CallTool(ctx, "pause_monitor", map[string]any{"uuid": uuid})
	return err
}

// ResumeMonitor resumes a paused monitor
func (c *MCPClient) ResumeMonitor(ctx context.Context, uuid string) error {
	_, err := c.transport.CallTool(ctx, "resume_monitor", map[string]any{"uuid": uuid})
	return err
}

// ==================== Uptime ====================

// GetMonitorUptime returns SLA uptime metrics
func (c *MCPClient) GetMonitorUptime(ctx context.Context, monitorUUID string) (*UptimeReport, error) {
	args := map[string]any{}
	if monitorUUID != "" {
		args["monitor_uuid"] = monitorUUID
	}

	result, err := c.transport.CallTool(ctx, "get_monitor_uptime", args)
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var report UptimeReport
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &report)
	}
	return &report, nil
}

// ==================== Outages ====================

// ListOutages returns paginated outage list
func (c *MCPClient) ListOutages(ctx context.Context, page int) (*OutageList, error) {
	args := map[string]any{}
	if page > 0 {
		args["page"] = page
	}

	result, err := c.transport.CallTool(ctx, "list_outages", args)
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var list OutageList
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &list)
	}
	return &list, nil
}

// GetOutage returns a single outage by UUID
func (c *MCPClient) GetOutage(ctx context.Context, uuid string) (*MCPOutage, error) {
	result, err := c.transport.CallTool(ctx, "get_outage", map[string]any{"uuid": uuid})
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var outage MCPOutage
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &outage)
	}
	return &outage, nil
}

// GetMonitorOutages returns outages for a specific monitor
func (c *MCPClient) GetMonitorOutages(ctx context.Context, monitorUUID string, page int) (*OutageList, error) {
	args := map[string]any{"monitor_uuid": monitorUUID}
	if page > 0 {
		args["page"] = page
	}

	result, err := c.transport.CallTool(ctx, "get_monitor_outages", args)
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var list OutageList
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &list)
	}
	return &list, nil
}

// GetOutageTimeline returns the lifecycle timeline for an outage
func (c *MCPClient) GetOutageTimeline(ctx context.Context, outageUUID string) (*OutageTimeline, error) {
	result, err := c.transport.CallTool(ctx, "get_outage_timeline", map[string]any{"uuid": outageUUID})
	if err != nil {
		return nil, err
	}

	data, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}

	var timeline OutageTimeline
	if bytes, err := json.Marshal(data); err == nil {
		json.Unmarshal(bytes, &timeline)
	}
	return &timeline, nil
}
