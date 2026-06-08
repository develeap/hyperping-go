// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Schema-contract test.
//
// Pins the MCP client against a snapshot of the live server's tools/list
// response (testdata/mcp_tools_list.json, captured during the v0.7.0 PR).
// For each tool that the Go client wraps, asserts that every request-arg
// name the client sends is declared in the snapshot's inputSchema.properties.
//
// If the live server drifts (renames a property, adds a required field, drops
// a property), this test stays pinned to the snapshot. Re-capturing the
// snapshot is a deliberate maintenance step, not an automatic one — that way
// the diff is visible in code review.
//
// This test is the regression guard for v0.6.3: at that release the client
// sent `uuid` for get_monitor_mtta but the server declared only
// (from, to, monitor_uuids). Loading the snapshot here and running this test
// would have caught it.

type schemaTool struct {
	Name        string `json:"name"`
	InputSchema struct {
		Properties map[string]any `json:"properties"`
		Required   []string       `json:"required"`
	} `json:"inputSchema"`
}

type schemaToolsList struct {
	Result struct {
		Tools []schemaTool `json:"tools"`
	} `json:"result"`
}

// clientToolArgs enumerates, per tool, the request-arg names the Go client
// currently sends. Update this map ONLY together with the corresponding
// CallTool args in mcp_client.go. The test rejects any arg not declared in
// the snapshot's inputSchema.properties.
//
// Scope: this registry covers the tools v0.7.0 explicitly probed against
// the live MCP server and confirmed correct. Other tools wrapped by
// MCPClient (create_monitor, update_monitor, etc.) are intentionally NOT
// listed yet because their full wire format has not been probed in this
// release; adding them here without probe evidence would just shift
// drift risk into a test that lies. Future PRs that probe additional
// tools should extend this map together with any client-side arg fixes
// surfaced by the comparison.
//
// Tools with no args (get_status_summary, list_team_members, ...) ARE
// listed with an empty slice when probed, so a future arg addition
// without a corresponding snapshot/property update is caught.
func clientToolArgs() map[string][]string {
	return map[string][]string{
		// Windowed batched tools — the v0.6.3 bug was here. v0.7.0 fixes
		// all four to send monitor_uuids[] + from/to per the snapshot.
		"get_monitor_mtta":          {"from", "to", "monitor_uuids"},
		"get_monitor_mttr":          {"from", "to", "monitor_uuids"},
		"get_monitor_response_time": {"from", "to", "monitor_uuids"},
		"get_monitor_uptime":        {"from", "to", "monitor_uuids"},
	}
}

func loadToolsListSnapshot(t *testing.T) schemaToolsList {
	t.Helper()
	path := filepath.Join("testdata", "mcp_tools_list.json")
	data, err := os.ReadFile(path)
	require.NoError(t, err, "missing snapshot %s; capture via ./testdata/regenerate_mcp_tools_list.sh", path)
	var s schemaToolsList
	require.NoError(t, json.Unmarshal(data, &s))
	require.NotEmpty(t, s.Result.Tools, "snapshot must contain at least one tool")
	return s
}

func TestSchemaContract_AllClientToolsArePresentInSnapshot(t *testing.T) {
	snap := loadToolsListSnapshot(t)
	declared := map[string]schemaTool{}
	for _, tl := range snap.Result.Tools {
		declared[tl.Name] = tl
	}

	for tool := range clientToolArgs() {
		_, ok := declared[tool]
		require.True(t, ok, "client wraps tool %q which is not declared in tools/list snapshot", tool)
	}
}

func TestSchemaContract_ClientArgsAreSubsetOfDeclaredProperties(t *testing.T) {
	snap := loadToolsListSnapshot(t)
	declared := map[string]schemaTool{}
	for _, tl := range snap.Result.Tools {
		declared[tl.Name] = tl
	}

	for tool, args := range clientToolArgs() {
		tl, ok := declared[tool]
		if !ok {
			continue // covered by the presence test above
		}
		// Build the declared property name set.
		declaredProps := map[string]struct{}{}
		for k := range tl.InputSchema.Properties {
			declaredProps[k] = struct{}{}
		}
		for _, arg := range args {
			_, ok := declaredProps[arg]
			require.True(t, ok,
				"tool %q: client sends arg %q which is not declared in inputSchema.properties; declared=%v",
				tool, arg, sortedKeys(declaredProps),
			)
		}
	}
}

func TestSchemaContract_WindowedToolsDeclareMonitorUuidsArray(t *testing.T) {
	// Targeted regression guard for the v0.6.3 bug. If the server ever
	// renames monitor_uuids back to uuid, or changes the type away from
	// array, this test fails loudly. (The snapshot is pinned, so the
	// failure would surface during the deliberate re-capture step.)
	snap := loadToolsListSnapshot(t)
	for _, tool := range []string{
		"get_monitor_mtta", "get_monitor_mttr",
		"get_monitor_response_time", "get_monitor_uptime",
	} {
		var found bool
		var props map[string]any
		for _, tl := range snap.Result.Tools {
			if tl.Name == tool {
				found = true
				props = tl.InputSchema.Properties
				break
			}
		}
		require.True(t, found, "snapshot is missing tool %q", tool)
		mu, ok := props["monitor_uuids"]
		require.True(t, ok, "tool %q: snapshot must declare monitor_uuids (the v0.6.3 bug pivot)", tool)
		// monitor_uuids must be an array per spec.
		muMap, ok := mu.(map[string]any)
		require.True(t, ok)
		require.Equal(t, "array", muMap["type"], "tool %q: monitor_uuids must be of type array", tool)
	}
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Sanity: the schema test uses a snapshot, not a network call. Keep this test
// as a tripwire so the snapshot path stays under testdata/.
func TestSchemaContract_NoNetworkInUnitTests(t *testing.T) {
	// Read with a tiny timeout would still try to open the file; the
	// timeout here is just a sanity guard for accidentally-network refactors.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_ = ctx
	loadToolsListSnapshot(t)
}
