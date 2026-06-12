// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMethods_GetTimezone_Healthcheck verifies that GetTimezone abstracts over
// the Hyperping API's inconsistency where POST responses use "timezone" and
// GET responses use "tz".
func TestMethods_GetTimezone_Healthcheck(t *testing.T) {
	t.Run("returns Timezone when set", func(t *testing.T) {
		hc := Healthcheck{Timezone: "America/New_York"}
		require.Equal(t, "America/New_York", hc.GetTimezone())
	})

	t.Run("falls back to Tz when Timezone is empty", func(t *testing.T) {
		hc := Healthcheck{Tz: "Europe/Paris"}
		require.Equal(t, "Europe/Paris", hc.GetTimezone())
	})

	t.Run("Timezone takes precedence over Tz when both set", func(t *testing.T) {
		hc := Healthcheck{Timezone: "America/New_York", Tz: "Europe/Paris"}
		require.Equal(t, "America/New_York", hc.GetTimezone())
	})

	t.Run("returns empty string when neither set", func(t *testing.T) {
		hc := Healthcheck{}
		require.Equal(t, "", hc.GetTimezone())
	})
}
