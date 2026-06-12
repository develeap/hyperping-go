// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package mockserver

import (
	"testing"
)

func TestSpecValidator_LoadsSpec(t *testing.T) {
	sv, err := newSpecValidator("../../openapi.yaml")
	if err != nil {
		t.Fatalf("newSpecValidator: unexpected error: %v", err)
	}
	if sv == nil {
		t.Fatal("newSpecValidator: returned nil validator")
	}
}

func TestSpecValidator_InvalidPath_Error(t *testing.T) {
	_, err := newSpecValidator("nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestSpecValidator_AllMutationOpsPresent(t *testing.T) {
	sv, err := newSpecValidator("../../openapi.yaml")
	if err != nil {
		t.Fatalf("newSpecValidator: %v", err)
	}
	for opKey := range opSchemas {
		if sv.schemas[opKey] == nil {
			t.Errorf("schema for %q is nil after load", opKey)
		}
	}
}

func TestSpecValidator_MonitorCreate_MissingURL_Error(t *testing.T) {
	sv, err := newSpecValidator("../../openapi.yaml")
	if err != nil {
		t.Fatalf("newSpecValidator: %v", err)
	}
	err = sv.Validate("POST /v1/monitors", []byte(`{"name":"x"}`))
	if err == nil {
		t.Fatal("expected validation error for missing url, got nil")
	}
}

func TestSpecValidator_MonitorCreate_FullBody_NoError(t *testing.T) {
	sv, err := newSpecValidator("../../openapi.yaml")
	if err != nil {
		t.Fatalf("newSpecValidator: %v", err)
	}
	err = sv.Validate("POST /v1/monitors", []byte(`{"name":"x","url":"https://x.com","protocol":"http"}`))
	if err != nil {
		t.Fatalf("expected no error for complete valid body, got: %v", err)
	}
}

func TestSpecValidator_HealthcheckCreate_NameOnly_Error(t *testing.T) {
	sv, err := newSpecValidator("../../openapi.yaml")
	if err != nil {
		t.Fatalf("newSpecValidator: %v", err)
	}
	err = sv.Validate("POST /v2/healthchecks", []byte(`{"name":"x"}`))
	if err == nil {
		t.Fatal("expected validation error for missing grace_period fields, got nil")
	}
}

func TestSpecValidator_StatusPageCreate_NameOnly_NoError(t *testing.T) {
	sv, err := newSpecValidator("../../openapi.yaml")
	if err != nil {
		t.Fatalf("newSpecValidator: %v", err)
	}
	err = sv.Validate("POST /v2/statuspages", []byte(`{"name":"My Page"}`))
	if err != nil {
		t.Fatalf("expected no error for valid status page body, got: %v", err)
	}
}
