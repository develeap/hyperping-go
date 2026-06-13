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

var nonAlphanumRe = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts a tenant name into a URL-safe subdomain slug:
// lowercase, collapse non-alphanumeric runs to hyphens, trim surrounding
// hyphens, truncate to 63 characters (DNS label limit).
func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 63 {
		s = s[:63]
		s = strings.TrimRight(s, "-")
	}
	return s
}

func newTenantCmd(state *cliState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tenant",
		Short: "Manage tenants",
	}
	cmd.AddCommand(newTenantOnboardCmd(state))
	return cmd
}

func newTenantOnboardCmd(state *cliState) *cobra.Command {
	var monitorURLs []string
	var protocol string

	cmd := &cobra.Command{
		Use:   "onboard <name>",
		Short: "Onboard a new tenant: create a status page and optional monitors",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			name := args[0]
			subdomain := slugify(name)

			page, err := state.client.CreateStatusPage(ctx, hyperping.CreateStatusPageRequest{
				Name:      name,
				Subdomain: &subdomain,
			})
			if err != nil {
				return fmt.Errorf("create status page: %w", err)
			}

			var monitorUUIDs []string
			for _, url := range monitorURLs {
				mon, err := state.client.CreateMonitor(ctx, hyperping.CreateMonitorRequest{
					Name:     fmt.Sprintf("%s - %s", name, url),
					URL:      url,
					Protocol: protocol,
				})
				if err != nil {
					return fmt.Errorf("create monitor %q: %w", url, err)
				}
				monitorUUIDs = append(monitorUUIDs, mon.UUID)
			}

			if len(monitorUUIDs) > 0 {
				page, err = state.client.UpdateStatusPage(ctx, page.UUID, hyperping.UpdateStatusPageRequest{
					Monitors: monitorUUIDs,
				})
				if err != nil {
					return fmt.Errorf("associate monitors: %w", err)
				}
			}

			if state.output == "json" {
				type result struct {
					UUID      string   `json:"uuid"`
					Name      string   `json:"name"`
					Subdomain string   `json:"subdomain"`
					Monitors  []string `json:"monitors,omitempty"`
				}
				return writeJSON(cmd.OutOrStdout(), result{
					UUID:      page.UUID,
					Name:      page.Name,
					Subdomain: page.HostedSubdomain,
					Monitors:  monitorUUIDs,
				})
			}

			writeTable(cmd.OutOrStdout(),
				[]string{"UUID", "NAME", "SUBDOMAIN", "MONITORS"},
				[][]string{{page.UUID, page.Name, page.HostedSubdomain, fmt.Sprintf("%d", len(monitorUUIDs))}},
			)
			return nil
		},
	}

	cmd.Flags().StringArrayVar(&monitorURLs, "monitor-url", nil, "Monitor URL to create and associate (repeatable)")
	cmd.Flags().StringVar(&protocol, "protocol", "https", "Protocol for created monitors")
	return cmd
}
