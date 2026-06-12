// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// oasDriftSchema is a parsed schema entry from components/schemas in openapi.yaml.
type oasDriftSchema struct {
	Properties map[string]oasDriftProperty `yaml:"properties"`
	Required   []string                    `yaml:"required"`
}

// oasDriftProperty is a parsed property from an OAS schema.
type oasDriftProperty struct {
	Type    string `yaml:"type"`
	Ref     string `yaml:"$ref"`
	XGoType string `yaml:"x-go-type"`
}

// schemaTypeRegistry maps each OAS schema name in components/schemas to its Go reflect.Type.
// Expand this map in sync with openapi.yaml whenever a schema is added or renamed.
var schemaTypeRegistry = map[string]reflect.Type{
	"Monitor":                     reflect.TypeOf(Monitor{}),
	"EscalationPolicyRef":         reflect.TypeOf(EscalationPolicyRef{}),
	"CreateMonitorRequest":        reflect.TypeOf(CreateMonitorRequest{}),
	"UpdateMonitorRequest":        reflect.TypeOf(UpdateMonitorRequest{}),
	"Incident":                    reflect.TypeOf(Incident{}),
	"IncidentUpdate":              reflect.TypeOf(IncidentUpdate{}),
	"CreateIncidentRequest":       reflect.TypeOf(CreateIncidentRequest{}),
	"UpdateIncidentRequest":       reflect.TypeOf(UpdateIncidentRequest{}),
	"AddIncidentUpdateRequest":    reflect.TypeOf(AddIncidentUpdateRequest{}),
	"Maintenance":                 reflect.TypeOf(Maintenance{}),
	"MaintenanceUpdate":           reflect.TypeOf(MaintenanceUpdate{}),
	"CreateMaintenanceRequest":    reflect.TypeOf(CreateMaintenanceRequest{}),
	"UpdateMaintenanceRequest":    reflect.TypeOf(UpdateMaintenanceRequest{}),
	"Healthcheck":                 reflect.TypeOf(Healthcheck{}),
	"CreateHealthcheckRequest":    reflect.TypeOf(CreateHealthcheckRequest{}),
	"UpdateHealthcheckRequest":    reflect.TypeOf(UpdateHealthcheckRequest{}),
	"Outage":                      reflect.TypeOf(Outage{}),
	"CreateOutageRequest":         reflect.TypeOf(CreateOutageRequest{}),
	"AcknowledgedByUser":          reflect.TypeOf(AcknowledgedByUser{}),
	"MonitorReference":            reflect.TypeOf(MonitorReference{}),
	"EscalationPolicyReference":   reflect.TypeOf(EscalationPolicyReference{}),
	"MonitorReport":               reflect.TypeOf(MonitorReport{}),
	"ReportPeriod":                reflect.TypeOf(ReportPeriod{}),
	"OutageStats":                 reflect.TypeOf(OutageStats{}),
	"OutageDetail":                reflect.TypeOf(OutageDetail{}),
	"ListMonitorReportsResponse":  reflect.TypeOf(ListMonitorReportsResponse{}),
	"StatusPage":                  reflect.TypeOf(StatusPage{}),
	"StatusPageSettings":          reflect.TypeOf(StatusPageSettings{}),
	"StatusPageSection":           reflect.TypeOf(StatusPageSection{}),
	"StatusPageService":           reflect.TypeOf(StatusPageService{}),
	"StatusPagePaginatedResponse": reflect.TypeOf(StatusPagePaginatedResponse{}),
	"CreateStatusPageRequest":     reflect.TypeOf(CreateStatusPageRequest{}),
	"UpdateStatusPageRequest":     reflect.TypeOf(UpdateStatusPageRequest{}),
	"StatusPageSubscriber":        reflect.TypeOf(StatusPageSubscriber{}),
	"AddSubscriberRequest":        reflect.TypeOf(AddSubscriberRequest{}),
	"LocalizedText":               reflect.TypeOf(LocalizedText{}),
	"RequestHeader":               reflect.TypeOf(RequestHeader{}),
}

// loadOpenAPISchemas reads openapi.yaml from the repository root and returns
// the parsed components/schemas map.
func loadOpenAPISchemas(t *testing.T) map[string]oasDriftSchema {
	t.Helper()
	data, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatalf("cannot open openapi.yaml: %v", err)
	}
	var doc struct {
		Components struct {
			Schemas map[string]oasDriftSchema `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("cannot parse openapi.yaml: %v", err)
	}
	return doc.Components.Schemas
}

// jsonTagKey extracts the base key from a json struct tag, stripping options such as omitempty.
// Returns "" for json:"-" or missing tags.
func jsonTagKey(tag reflect.StructTag) string {
	s := tag.Get("json")
	if s == "" || s == "-" {
		return ""
	}
	name, _, _ := strings.Cut(s, ",")
	return name
}

// jsonTagIndex builds a map of json tag key -> struct field for a struct type.
func jsonTagIndex(t reflect.Type) map[string]reflect.StructField {
	m := make(map[string]reflect.StructField, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if k := jsonTagKey(f.Tag); k != "" {
			m[k] = f
		}
	}
	return m
}

// TestSpecDrift_AllSchemasHaveGoTypes verifies that every schema in components/schemas
// has an entry in schemaTypeRegistry. A missing entry means the Go type is missing or the
// registry is out of sync with openapi.yaml.
func TestSpecDrift_AllSchemasHaveGoTypes(t *testing.T) {
	schemas := loadOpenAPISchemas(t)
	for name := range schemas {
		if _, ok := schemaTypeRegistry[name]; !ok {
			t.Errorf("schema %q is in openapi.yaml but missing from schemaTypeRegistry", name)
		}
	}
}

// TestSpecDrift_SchemaPropertiesMapToJSONTags verifies that for each OAS schema property
// there is a struct field in the corresponding Go type whose json tag matches the property
// name exactly. This is the regression guard for the v0.6.x silent-zero MTTA bug class.
func TestSpecDrift_SchemaPropertiesMapToJSONTags(t *testing.T) {
	schemas := loadOpenAPISchemas(t)
	for schemaName, schema := range schemas {
		goType, ok := schemaTypeRegistry[schemaName]
		if !ok {
			continue
		}
		idx := jsonTagIndex(goType)
		for propName := range schema.Properties {
			if _, found := idx[propName]; !found {
				t.Errorf("schema %q: property %q has no matching json tag in Go type %s",
					schemaName, propName, goType.Name())
			}
		}
	}
}

// TestSpecDrift_RequiredPropertiesAreNotDroppedOnZero verifies that OAS required properties
// that map to non-pointer Go struct fields do not carry omitempty. A non-pointer required
// field with omitempty is silently dropped by encoding/json when the field is zero-valued.
func TestSpecDrift_RequiredPropertiesAreNotDroppedOnZero(t *testing.T) {
	schemas := loadOpenAPISchemas(t)
	for schemaName, schema := range schemas {
		goType, ok := schemaTypeRegistry[schemaName]
		if !ok {
			continue
		}
		idx := jsonTagIndex(goType)
		for _, propName := range schema.Required {
			sf, found := idx[propName]
			if !found {
				continue
			}
			if sf.Type.Kind() == reflect.Ptr {
				continue
			}
			tag := sf.Tag.Get("json")
			if strings.Contains(tag, ",omitempty") {
				t.Errorf("schema %q: required property %q maps to %s.%s (kind=%s) with omitempty — zero value is silently dropped",
					schemaName, propName, goType.Name(), sf.Name, sf.Type.Kind())
			}
		}
	}
}
