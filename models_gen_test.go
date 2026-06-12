// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGenerated_TypesCompile instantiates one generated type per domain to
// ensure models_gen.go compiles and field names are accessible. If oapi-codegen
// changes a field name or type incompatibly, this test fails to compile.
func TestGenerated_TypesCompile(t *testing.T) {
	_ = Healthcheck{UUID: "tok_x", Name: "n", PingURL: "u", Period: 60, GracePeriod: 30}
	_ = Incident{UUID: "inc_x", Type: "outage", StatusPages: []string{"sp_x"}}
	_ = Maintenance{UUID: "mw_x", Name: "m", Monitors: []string{"mon_x"}}
	_ = Outage{UUID: "out_x", StartDate: "2026-01-01T00:00:00Z", OutageType: "automatic"}
	_ = MonitorReport{UUID: "mon_x", Name: "n", SLA: 99.9}
	_ = StatusPage{UUID: "sp_x", Name: "n", HostedSubdomain: "x.hyperping.app"}
	_ = StatusPageSubscriber{ID: 1, Type: "email", Value: "a@b.com"}
	_ = LocalizedText{En: "hello"}
	_ = RequestHeader{Name: "X-Key", Value: "v"}
	_ = CreateMonitorRequest{Name: "n", URL: "https://x.example.com"}
	_ = CreateHealthcheckRequest{Name: "hc", GracePeriodValue: 5, GracePeriodType: "minutes"}
	_ = CreateIncidentRequest{Type: "outage", StatusPages: []string{"sp_x"}}
	_ = CreateMaintenanceRequest{Name: "mw", StartDate: "2026-01-01T00:00:00Z", EndDate: "2026-01-02T00:00:00Z", Monitors: []string{"mon_x"}}
	_ = CreateOutageRequest{MonitorUUID: "mon_x", StartDate: "2026-01-01T00:00:00Z", StatusCode: 503, Description: "d", OutageType: "manual"}
}

// TestGenerated_JSONRoundtrip_Healthcheck marshals and unmarshals a Healthcheck
// through the generated struct and verifies all fields survive the round-trip.
// This is the regression guard for json-tag drift on the Healthcheck schema.
func TestGenerated_JSONRoundtrip_Healthcheck(t *testing.T) {
	periodVal := 1
	hc := Healthcheck{
		UUID:             "tok_abc123def456",
		Name:             "Daily backup check",
		PingURL:          "https://hping.io/healthchecks/check/tok_abc123",
		Cron:             "0 0 * * *",
		Timezone:         "America/New_York",
		PeriodValue:      &periodVal,
		PeriodType:       "days",
		Period:           86400,
		GracePeriod:      3600,
		GracePeriodValue: 1,
		GracePeriodType:  "hours",
		IsDown:           false,
		IsPaused:         false,
		LastPing:         "2026-06-13T00:00:00Z",
		EscalationPolicy: &EscalationPolicyReference{
			UUID:         "ep_abc123",
			Name:         "Default",
			AlertedSteps: 1,
			TotalSteps:   3,
		},
	}

	data, err := json.Marshal(hc)
	require.NoError(t, err)

	var got Healthcheck
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, hc, got)
}

// TestGenerated_JSONRoundtrip_Incident marshals and unmarshals an Incident with
// nested LocalizedText and IncidentUpdate slices, verifying all fields survive.
func TestGenerated_JSONRoundtrip_Incident(t *testing.T) {
	inc := Incident{
		UUID:               "inc_xyz789",
		Date:               "2026-06-13T12:00:00Z",
		Title:              LocalizedText{En: "API Outage", Fr: "Panne API"},
		Text:               LocalizedText{En: "The API is experiencing issues"},
		Type:               "outage",
		AffectedComponents: []string{"mon_abc123"},
		StatusPages:        []string{"sp_abc123"},
		Updates: []IncidentUpdate{
			{
				UUID: "upd_001",
				Date: "2026-06-13T12:05:00Z",
				Text: LocalizedText{En: "Investigating the issue"},
				Type: "investigating",
			},
			{
				UUID: "upd_002",
				Date: "2026-06-13T12:30:00Z",
				Text: LocalizedText{En: "Issue identified and resolved"},
				Type: "resolved",
			},
		},
	}

	data, err := json.Marshal(inc)
	require.NoError(t, err)

	var got Incident
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, inc, got)
}

// TestGenerated_JSONRoundtrip_StatusPage marshals and unmarshals a StatusPage
// with nested sections and services, verifying all fields survive.
func TestGenerated_JSONRoundtrip_StatusPage(t *testing.T) {
	flexID := FlexibleString("mon_abc123")
	sp := StatusPage{
		UUID:              "sp_abc123xyz",
		Name:              "Acme Status",
		Hostname:          "status.acme.com",
		HostedSubdomain:   "acme.hyperping.app",
		URL:               "https://status.acme.com",
		PasswordProtected: false,
		Settings: StatusPageSettings{
			Name:            "Acme Status",
			Languages:       []string{"en", "fr"},
			DefaultLanguage: "en",
			Theme:           "light",
			Font:            "Inter",
			AccentColor:     "#0070f3",
			AutoRefresh:     true,
			Subscribe:       StatusPageSubscribeSettings{Enabled: true, Email: true},
			Authentication:  StatusPageAuthenticationSettings{},
		},
		Sections: []StatusPageSection{
			{
				Name:    map[string]string{"en": "API Services"},
				IsSplit: false,
				Services: []StatusPageService{
					{
						ID:      &flexID,
						UUID:    "mon_abc123",
						Name:    map[string]string{"en": "Production API"},
						IsGroup: false,
					},
				},
			},
		},
	}

	data, err := json.Marshal(sp)
	require.NoError(t, err)

	var got StatusPage
	require.NoError(t, json.Unmarshal(data, &got))
	require.Equal(t, sp, got)
}
