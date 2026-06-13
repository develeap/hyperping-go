// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"

	hyperping "github.com/develeap/hyperping-go"
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
				return writeJSON(cmd.OutOrStdout(), resp.StatusPages)
			}
			writeTable(cmd.OutOrStdout(),
				[]string{"UUID", "NAME", "URL"},
				statuspagesToRows(resp.StatusPages))
			return nil
		},
	}
}

func statuspagesToRows(pages []hyperping.StatusPage) [][]string {
	rows := make([][]string, len(pages))
	for i, p := range pages {
		rows[i] = []string{p.UUID, p.Name, p.URL}
	}
	return rows
}

func newStatuspageShowCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "show <uuid>",
		Short: "Show a status page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			page, err := state.client.GetStatusPage(context.Background(), args[0])
			if err != nil {
				return fmt.Errorf("get status page: %w", err)
			}
			if state.output == "json" {
				return writeJSON(cmd.OutOrStdout(), page)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "UUID:  %s\n", page.UUID)
			fmt.Fprintf(cmd.OutOrStdout(), "Name:  %s\n", page.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "URL:   %s\n", page.URL)
			if page.Hostname != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Host:  %s\n", *page.Hostname)
			}
			return nil
		},
	}
}
