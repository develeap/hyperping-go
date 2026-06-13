// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newStatuspageCmd(state *cliState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "statuspage",
		Short: "Manage status pages",
	}
	cmd.AddCommand(newStatuspageListCmd(state))
	cmd.AddCommand(newStatuspageShowCmd(state))
	return cmd
}

func newStatuspageListCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all status pages",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := state.client.ListStatusPages(context.Background(), nil, nil)
			if err != nil {
				return fmt.Errorf("list status pages: %w", err)
			}
			if len(resp.StatusPages) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no status pages found")
				return nil
			}
			if state.output == "json" {
				writeJSON(cmd.OutOrStdout(), resp.StatusPages)
				return nil
			}
			headers := []string{"UUID", "NAME", "SUBDOMAIN", "URL"}
			rows := make([][]string, len(resp.StatusPages))
			for i, sp := range resp.StatusPages {
				rows[i] = []string{sp.UUID, sp.Name, sp.HostedSubdomain, sp.URL}
			}
			writeTable(cmd.OutOrStdout(), headers, rows)
			return nil
		},
	}
}

func newStatuspageShowCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "show <uuid>",
		Short: "Show a single status page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid := args[0]
			sp, err := state.client.GetStatusPage(context.Background(), uuid)
			if err != nil {
				return fmt.Errorf("get status page: %w", err)
			}
			if state.output == "json" {
				writeJSON(cmd.OutOrStdout(), sp)
				return nil
			}
			headers := []string{"UUID", "NAME", "SUBDOMAIN", "URL"}
			rows := [][]string{{sp.UUID, sp.Name, sp.HostedSubdomain, sp.URL}}
			writeTable(cmd.OutOrStdout(), headers, rows)
			return nil
		},
	}
}
