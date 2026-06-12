// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package mockserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

// jsonableDoc converts the yaml.v3-unmarshalled tree to a tree whose types
// are all JSON-native (map[string]any, []any, float64, string, bool, nil).
// yaml.v3 already uses string-keyed maps and standard scalars, but integer
// values come out as int, which the jsonschema library does not accept as a
// valid JSON value type. Marshalling to JSON and unmarshalling back normalises
// everything to the types json.Unmarshal produces.
func jsonableDoc(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if err := d.Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

// opSchemas maps mux pattern keys to OAS component schema names for request bodies.
var opSchemas = map[string]string{
	"POST /v1/monitors":                       "CreateMonitorRequest",
	"PUT /v1/monitors/{uuid}":                 "UpdateMonitorRequest",
	"POST /v2/healthchecks":                   "CreateHealthcheckRequest",
	"PUT /v2/healthchecks/{uuid}":             "UpdateHealthcheckRequest",
	"POST /v3/incidents":                      "CreateIncidentRequest",
	"PUT /v3/incidents/{uuid}":                "UpdateIncidentRequest",
	"POST /v3/incidents/{uuid}/updates":       "AddIncidentUpdateRequest",
	"POST /v1/maintenance-windows":            "CreateMaintenanceRequest",
	"PUT /v1/maintenance-windows/{uuid}":      "UpdateMaintenanceRequest",
	"POST /v2/outages":                        "CreateOutageRequest",
	"POST /v2/statuspages":                    "CreateStatusPageRequest",
	"PUT /v2/statuspages/{uuid}":              "UpdateStatusPageRequest",
	"POST /v2/statuspages/{uuid}/subscribers": "AddSubscriberRequest",
}

type specValidator struct {
	schemas map[string]*jsonschema.Schema
}

// newSpecValidator loads specFile (OAS 3.1 YAML), converts it to a JSON-native
// document tree, and pre-compiles all mutation operation schemas from opSchemas.
func newSpecValidator(specFile string) (*specValidator, error) {
	raw, err := os.ReadFile(specFile)
	if err != nil {
		return nil, fmt.Errorf("read spec file: %w", err)
	}

	var yamlDoc any
	if err := yaml.Unmarshal(raw, &yamlDoc); err != nil {
		return nil, fmt.Errorf("parse spec YAML: %w", err)
	}

	doc, err := jsonableDoc(yamlDoc)
	if err != nil {
		return nil, fmt.Errorf("normalise spec to JSON types: %w", err)
	}

	c := jsonschema.NewCompiler()
	if err := c.AddResource("file:///openapi.yaml", doc); err != nil {
		return nil, fmt.Errorf("register spec resource: %w", err)
	}

	sv := &specValidator{
		schemas: make(map[string]*jsonschema.Schema, len(opSchemas)),
	}
	for opKey, schemaName := range opSchemas {
		s, err := c.Compile("file:///openapi.yaml#/components/schemas/" + schemaName)
		if err != nil {
			return nil, fmt.Errorf("compile schema %q: %w", schemaName, err)
		}
		sv.schemas[opKey] = s
	}
	return sv, nil
}

// Validate validates body against the compiled schema for opKey.
// Returns nil when opKey has no registered schema (non-mutation operations).
func (sv *specValidator) Validate(opKey string, body []byte) error {
	s, ok := sv.schemas[opKey]
	if !ok {
		return nil
	}
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	return s.Validate(v)
}

// validateBodySchema reads the request body, validates it against the spec,
// and restores the body so downstream handlers can still read it.
// Returns nil when sv is nil (structural-validation fallback path).
func validateBodySchema(r *http.Request, sv *specValidator, opKey string) error {
	if sv == nil {
		return nil
	}
	var body []byte
	if r.Body != nil {
		var err error
		body, err = io.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("read request body: %w", err)
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
	}
	return sv.Validate(opKey, body)
}
