// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// =============================================================================
// FlexibleString
// =============================================================================

// FlexibleString is a string type that can unmarshal from both JSON strings and numbers.
// This handles API inconsistencies where a field might be returned as either type.
type FlexibleString string

// maxFlexibleStringBytes is the maximum allowed input size for FlexibleString.
// Prevents memory exhaustion from malicious or malformed numeric strings (VULN-004).
const maxFlexibleStringBytes = 100

// UnmarshalJSON implements json.Unmarshaler for FlexibleString.
func (fs *FlexibleString) UnmarshalJSON(data []byte) error {
	if len(data) > maxFlexibleStringBytes {
		return fmt.Errorf("FlexibleString input exceeds maximum size of %d bytes", maxFlexibleStringBytes)
	}

	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*fs = FlexibleString(s)
		return nil
	}

	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*fs = FlexibleString(n.String())
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into FlexibleString", string(data))
}

// MarshalJSON implements json.Marshaler for FlexibleString.
func (fs FlexibleString) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(fs))
}

// String returns the string value.
func (fs FlexibleString) String() string {
	return string(fs)
}

// =============================================================================
// Validation helpers
// =============================================================================

// Input length limits to prevent resource exhaustion (VULN-007).
const (
	maxNameLength    = 255
	maxURLLength     = 2048
	maxMessageLength = 10000
)

// validateStringLength checks that a string does not exceed the given max length
// measured in Unicode code points (runes), not bytes (VULN-018).
func validateStringLength(field, value string, maxLen int) error {
	runeCount := utf8.RuneCountInString(value)
	if runeCount > maxLen {
		return fmt.Errorf("field %q exceeds maximum length of %d characters (got %d)", field, maxLen, runeCount)
	}
	return nil
}

// localizedField pairs a language code with its value for deterministic iteration.
type localizedField struct {
	lang string
	val  string
}

// localizedFields returns all locale fields in a stable, deterministic order.
func localizedFields(text LocalizedText) []localizedField {
	return []localizedField{
		{"en", text.En}, {"fr", text.Fr}, {"de", text.De}, {"ru", text.Ru},
		{"nl", text.Nl}, {"es", text.Es}, {"it", text.It}, {"pt", text.Pt},
		{"ja", text.Ja}, {"zh", text.Zh},
	}
}

// validateLocalizedText validates all non-empty locale fields of a LocalizedText value.
func validateLocalizedText(prefix string, text LocalizedText, maxLen int) error {
	for _, f := range localizedFields(text) {
		if f.val != "" {
			if err := validateStringLength(prefix+"."+f.lang, f.val, maxLen); err != nil {
				return err
			}
		}
	}
	return nil
}

// =============================================================================
// Allowed values
// =============================================================================

const (
	// DefaultMonitorFrequency is the default check frequency for monitors in seconds.
	DefaultMonitorFrequency = 60

	// DefaultMonitorTimeout is the default timeout for monitor checks in seconds.
	DefaultMonitorTimeout = 10

	// DefaultNotifyBeforeMinutes is the default number of minutes before maintenance
	// to notify subscribers.
	DefaultNotifyBeforeMinutes = 60
)

var (
	// AllowedFrequencies contains valid monitor check frequencies in seconds.
	AllowedFrequencies = []int{10, 20, 30, 60, 120, 180, 300, 600, 1800, 3600, 21600, 43200, 86400}

	// AllowedTimeouts contains valid monitor timeout values in seconds.
	AllowedTimeouts = []int{5, 10, 15, 20}

	// AllowedProtocols contains valid monitor protocols.
	AllowedProtocols = []string{"http", "port", "icmp", "dns"}

	// AllowedDNSRecordTypes contains valid DNS record types for DNS-protocol monitors.
	AllowedDNSRecordTypes = []string{"A", "AAAA", "CNAME", "MX", "NS", "TXT", "SOA", "SRV", "CAA", "PTR"}

	// AllowedMethods contains valid HTTP methods for monitors.
	AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

	// AllowedRegions contains valid monitor check regions.
	AllowedRegions = []string{
		"london", "frankfurt", "paris", "amsterdam",
		"singapore", "sydney", "tokyo", "seoul", "mumbai", "bangalore",
		"virginia", "california", "sanfrancisco", "nyc", "toronto",
		"saopaulo",
		"bahrain",
		"capetown",
	}

	// AllowedIncidentTypes contains valid incident type values.
	AllowedIncidentTypes = []string{"outage", "incident"}

	// AllowedIncidentUpdateTypes contains valid incident update type values.
	AllowedIncidentUpdateTypes = []string{"investigating", "identified", "update", "monitoring", "resolved"}

	// AllowedNotificationOptions contains valid maintenance notification options.
	AllowedNotificationOptions = []string{"none", "scheduled", "immediate"}

	// AllowedPeriodTypes contains valid healthcheck period type values.
	AllowedPeriodTypes = []string{"seconds", "minutes", "hours", "days"}

	// AllowedStatusPageThemes contains valid status page theme values.
	AllowedStatusPageThemes = []string{"light", "dark", "system"}

	// AllowedStatusPageFonts contains valid status page font values.
	AllowedStatusPageFonts = []string{
		"system-ui", "Lato", "Manrope", "Inter", "Open Sans",
		"Montserrat", "Poppins", "Roboto", "Raleway", "Nunito",
		"Merriweather", "DM Sans", "Work Sans",
	}

	// AllowedLanguages contains valid language codes for status page configuration.
	AllowedLanguages = []string{"en", "fr", "de", "ru", "nl", "pl", "sv"}

	// AllowedSubscriberTypes contains valid subscriber type values.
	AllowedSubscriberTypes = []string{"email", "sms", "teams"}
)

// =============================================================================
// Monitor (hand-coded: polymorphic escalation_policy field)
// =============================================================================

// Monitor represents a Hyperping monitor.
// API: GET /v1/monitors, GET /v1/monitors/{uuid}
//
// The escalation_policy field is polymorphic: the read API returns an object
// {"uuid":"...","name":"..."}, while POST/PUT send a plain UUID string.
// UnmarshalJSON handles both shapes and normalises to *EscalationPolicyRef.
type Monitor struct {
	ID                 int                  `json:"id"`
	UUID               string               `json:"uuid"`
	Name               string               `json:"name"`
	URL                string               `json:"url"`
	Protocol           string               `json:"protocol"`
	ProjectUUID        string               `json:"projectUuid,omitempty"`
	HTTPMethod         string               `json:"http_method"`
	Regions            []string             `json:"regions"`
	CheckFrequency     int                  `json:"check_frequency"`
	RequestHeaders     []RequestHeader      `json:"request_headers"`
	RequestBody        string               `json:"request_body,omitempty"`
	FollowRedirects    bool                 `json:"follow_redirects"`
	ExpectedStatusCode FlexibleString       `json:"expected_status_code"`
	RequiredKeyword    *string              `json:"required_keyword,omitempty"`
	Paused             bool                 `json:"paused"`
	Port               *int                 `json:"port,omitempty"`
	AlertsWait         int                  `json:"alerts_wait,omitempty"`
	EscalationPolicy   *EscalationPolicyRef `json:"escalation_policy,omitempty"`
	DNSRecordType      *string              `json:"dns_record_type,omitempty"`
	DNSNameserver      *string              `json:"dns_nameserver,omitempty"`
	DNSExpectedAnswer  *string              `json:"dns_expected_answer,omitempty"`
	Status             string               `json:"status,omitempty"`
	SSLExpiration      *int                 `json:"ssl_expiration,omitempty"`
}

// escalationPolicyShape is used internally to unmarshal escalation_policy from
// the read API, which returns an object {"uuid":"...","name":"..."}.
type escalationPolicyShape struct {
	UUID string `json:"uuid"`
	Name string `json:"name,omitempty"`
}

// monitorAlias is used to prevent infinite recursion in Monitor.UnmarshalJSON.
type monitorAlias Monitor

// monitorWire is the raw JSON shape for Monitor, with escalation_policy as a
// raw message so we can handle both the string and object forms.
type monitorWire struct {
	monitorAlias
	EscalationPolicy json.RawMessage `json:"escalation_policy,omitempty"`
}

// UnmarshalJSON implements json.Unmarshaler for Monitor.
// It handles the escalation_policy field being either a plain UUID string or an
// object {"uuid":"...","name":"..."} as returned by the Hyperping read API.
// A nil EscalationPolicy indicates no policy is set.
func (m *Monitor) UnmarshalJSON(data []byte) error {
	var wire monitorWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	*m = Monitor(wire.monitorAlias)

	if len(wire.EscalationPolicy) == 0 || string(wire.EscalationPolicy) == "null" {
		m.EscalationPolicy = nil
		return nil
	}

	var uuidStr string
	if err := json.Unmarshal(wire.EscalationPolicy, &uuidStr); err == nil {
		if uuidStr == "" {
			m.EscalationPolicy = nil
		} else {
			m.EscalationPolicy = &EscalationPolicyRef{UUID: uuidStr}
		}
		return nil
	}

	var obj escalationPolicyShape
	if err := json.Unmarshal(wire.EscalationPolicy, &obj); err == nil {
		if obj.UUID == "" {
			m.EscalationPolicy = nil
		} else {
			m.EscalationPolicy = &EscalationPolicyRef{UUID: obj.UUID, Name: obj.Name}
		}
		return nil
	}

	return fmt.Errorf("cannot unmarshal escalation_policy: %s", string(wire.EscalationPolicy))
}

// =============================================================================
// Methods on generated types
// =============================================================================

// GetTimezone returns the timezone value regardless of which JSON field was populated.
// The Hyperping API is inconsistent: POST responses use "timezone" while GET responses
// use "tz". This method abstracts over that inconsistency.
func (h Healthcheck) GetTimezone() string {
	if h.Timezone != "" {
		return h.Timezone
	}
	return h.Tz
}

// Validate checks input lengths on CreateMonitorRequest fields.
func (r CreateMonitorRequest) Validate() error {
	if err := validateStringLength("name", r.Name, maxNameLength); err != nil {
		return err
	}
	if err := validateStringLength("url", r.URL, maxURLLength); err != nil {
		return err
	}
	return nil
}

// Validate checks input lengths on CreateHealthcheckRequest fields.
func (r CreateHealthcheckRequest) Validate() error {
	return validateStringLength("name", r.Name, maxNameLength)
}

// Validate checks input lengths on CreateIncidentRequest fields.
func (r CreateIncidentRequest) Validate() error {
	if err := validateLocalizedText("title", r.Title, maxNameLength); err != nil {
		return err
	}
	return validateLocalizedText("text", r.Text, maxMessageLength)
}

// Validate checks input lengths on CreateMaintenanceRequest fields.
func (r CreateMaintenanceRequest) Validate() error {
	if err := validateStringLength("name", r.Name, maxNameLength); err != nil {
		return err
	}
	if err := validateLocalizedText("title", r.Title, maxNameLength); err != nil {
		return err
	}
	return validateLocalizedText("text", r.Text, maxMessageLength)
}

// Validate checks input lengths on CreateOutageRequest fields.
func (r CreateOutageRequest) Validate() error {
	return validateStringLength("description", r.Description, maxMessageLength)
}

// Validate checks input lengths on CreateStatusPageRequest fields.
func (r CreateStatusPageRequest) Validate() error {
	if err := validateStringLength("name", r.Name, maxNameLength); err != nil {
		return err
	}
	if r.Website != nil && *r.Website != "" {
		if err := validateStringLength("website", *r.Website, maxURLLength); err != nil {
			return err
		}
	}
	return nil
}

// =============================================================================
// Hand-coded types referenced via x-go-type in openapi.yaml
// =============================================================================

// StatusPageSubscribeSettings represents subscription settings for a status page.
type StatusPageSubscribeSettings struct {
	Enabled bool `json:"enabled"`
	Email   bool `json:"email"`
	Slack   bool `json:"slack"`
	Teams   bool `json:"teams"`
	SMS     bool `json:"sms"`
}

// StatusPageAuthenticationSettings represents authentication settings for a status page.
type StatusPageAuthenticationSettings struct {
	PasswordProtection bool     `json:"password_protection"`
	GoogleSSO          bool     `json:"google_sso"`
	SAMLSSO            bool     `json:"saml_sso"`
	AllowedDomains     []string `json:"google_allowed_domains"`
	SSOConnectionUUID  *string  `json:"sso_connection_uuid"`
}

// CreateStatusPageSubscribeSettings represents subscription settings in create/update requests.
type CreateStatusPageSubscribeSettings struct {
	Enabled *bool `json:"enabled,omitempty"`
	Email   *bool `json:"email,omitempty"`
	Slack   *bool `json:"slack,omitempty"`
	Teams   *bool `json:"teams,omitempty"`
	SMS     *bool `json:"sms,omitempty"`
}

// CreateStatusPageAuthenticationSettings represents authentication settings in create/update requests.
type CreateStatusPageAuthenticationSettings struct {
	PasswordProtection *bool    `json:"password_protection,omitempty"`
	GoogleSSO          *bool    `json:"google_sso,omitempty"`
	SAMLSSO            *bool    `json:"saml_sso,omitempty"`
	AllowedDomains     []string `json:"google_allowed_domains,omitempty"`
	SSOConnectionUUID  *string  `json:"sso_connection_uuid,omitempty"`
}

// CreateStatusPageSection represents a section in create/update requests.
type CreateStatusPageSection struct {
	Name     string                    `json:"name"`
	IsSplit  *bool                     `json:"is_split,omitempty"`
	Services []CreateStatusPageService `json:"services,omitempty"`
}

// CreateStatusPageService represents a service in create/update requests.
// Top-level monitor entries use MonitorUUID ("monitor_uuid").
// Nested child services inside groups use UUID ("uuid").
// Group header entries omit both UUID fields and use only NameShown + IsGroup + Services.
type CreateStatusPageService struct {
	MonitorUUID       *string                   `json:"monitor_uuid,omitempty"`
	UUID              *string                   `json:"uuid,omitempty"`
	NameShown         *string                   `json:"name_shown,omitempty"`
	Name              map[string]string         `json:"name,omitempty"`
	ShowUptime        *bool                     `json:"show_uptime,omitempty"`
	ShowResponseTimes *bool                     `json:"show_response_times,omitempty"`
	Description       *string                   `json:"description,omitempty"`
	IsGroup           *bool                     `json:"is_group,omitempty"`
	Services          []CreateStatusPageService `json:"services,omitempty"`
}

// =============================================================================
// Action types (not in OAS spec)
// =============================================================================

// HealthcheckAction represents an action performed on a healthcheck.
// Used for pause, resume responses.
type HealthcheckAction struct {
	Message string `json:"message"`
	UUID    string `json:"uuid"`
}

// OutageAction represents an action performed on an outage.
// Used for acknowledge, resolve, escalate responses.
type OutageAction struct {
	Message string `json:"message"`
	UUID    string `json:"uuid"`
}

// SubscriberPaginatedResponse represents a paginated list of subscribers.
// API: /v2/statuspages/{uuid}/subscribers with pagination
type SubscriberPaginatedResponse struct {
	Subscribers    []StatusPageSubscriber `json:"subscribers"`
	HasNextPage    bool                   `json:"hasNextPage"`
	Total          int                    `json:"total"`
	Page           int                    `json:"page"`
	ResultsPerPage int                    `json:"resultsPerPage"`
}

// =============================================================================
// Subscriber validation
// =============================================================================

func validateEmailSubscriber(req AddSubscriberRequest) error {
	if req.Email == nil || *req.Email == "" {
		return fmt.Errorf("email is required when type is 'email'")
	}
	return nil
}

func validateSMSSubscriber(req AddSubscriberRequest) error {
	if req.Phone == nil || *req.Phone == "" {
		return fmt.Errorf("phone is required when type is 'sms'")
	}
	return nil
}

func validateTeamsSubscriber(req AddSubscriberRequest) error {
	if req.TeamsWebhookURL == nil || *req.TeamsWebhookURL == "" {
		return fmt.Errorf("teams_webhook_url is required when type is 'teams'")
	}
	return nil
}

func validateSubscriberType(subscriberType string) error {
	for _, t := range AllowedSubscriberTypes {
		if subscriberType == t {
			return nil
		}
	}
	return fmt.Errorf("invalid subscriber type %q, must be one of: %v", subscriberType, AllowedSubscriberTypes)
}

func validateSubscriberLanguage(language string) error {
	for _, lang := range AllowedLanguages {
		if language == lang {
			return nil
		}
	}
	return fmt.Errorf("invalid language %q, must be one of: %v", language, AllowedLanguages)
}

// typeValidators maps subscriber type names to their per-type validation functions.
var typeValidators = map[string]func(AddSubscriberRequest) error{
	"email": validateEmailSubscriber,
	"sms":   validateSMSSubscriber,
	"teams": validateTeamsSubscriber,
}

// Validate checks that the required fields are present based on the subscriber type.
func (r AddSubscriberRequest) Validate() error {
	if err := validateSubscriberType(r.Type); err != nil {
		return err
	}
	if validator, ok := typeValidators[r.Type]; ok {
		if err := validator(r); err != nil {
			return err
		}
	}
	if r.Language != nil && *r.Language != "" {
		return validateSubscriberLanguage(*r.Language)
	}
	return nil
}
