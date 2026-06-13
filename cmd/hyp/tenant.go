// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	hyperping "github.com/develeap/hyperping-go"
	"github.com/spf13/cobra"
)

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts a name to a DNS-safe subdomain label:
// lowercase, non-alphanumeric runs replaced with hyphens, trimmed, truncated to 63 chars.
func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugNonAlnum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 63 {
		s = strings.TrimRight(s[:63], "-")
	}
	return s
}

func newTenantCmd(state *cliState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tenant",
		Short: "Tenant management",
	}
	cmd.AddCommand(newTenantOnboardCmd(state))
	return cmd
}

func newTenantOnboardCmd(state *cliState) *cobra.Command {
	var monitorURLs []string
	var protocol string

	cmd := &cobra.Command{
		Use:   "onboard <name>",
		Short: "Onboard a new tenant: create status page and monitors",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			subdomain := slugify(name)

			sp, err := state.client.CreateStatusPage(context.Background(), hyperping.CreateStatusPageRequest{
				Name:      name,
				Subdomain: &subdomain,
			})
			if err != nil {
				return fmt.Errorf("create status page: %w", err)
			}

			monitorUUIDs := make([]string, 0, len(monitorURLs))
			for _, u := range monitorURLs {
				mon, err := state.client.CreateMonitor(context.Background(), hyperping.CreateMonitorRequest{
					Name:     name + " - " + u,
					URL:      u,
					Protocol: protocol,
				})
				if err != nil {
					return fmt.Errorf("create monitor for %s: %w", u, err)
				}
				monitorUUIDs = append(monitorUUIDs, mon.UUID)
			}

			if len(monitorUUIDs) > 0 {
				if _, err := state.client.UpdateStatusPage(context.Background(), sp.UUID, hyperping.UpdateStatusPageRequest{
					Monitors: monitorUUIDs,
				}); err != nil {
					return fmt.Errorf("associate monitors with status page: %w", err)
				}
			}

			if state.output == "json" {
				result := map[string]interface{}{
					"uuid":      sp.UUID,
					"name":      sp.Name,
					"subdomain": sp.HostedSubdomain,
					"monitors":  monitorUUIDs,
				}
				writeJSON(cmd.OutOrStdout(), result)
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "status page created: %s (%s)\n", sp.UUID, sp.HostedSubdomain)
			for _, id := range monitorUUIDs {
				fmt.Fprintf(cmd.OutOrStdout(), "monitor created: %s\n", id)
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&monitorURLs, "monitor-url", nil, "Monitor URL to add (repeatable)")
	cmd.Flags().StringVar(&protocol, "protocol", "https", "Monitor protocol (default: https)")

	return cmd
}
