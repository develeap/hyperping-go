// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"strconv"

	hyperping "github.com/develeap/hyperping-go"
	"github.com/spf13/cobra"
)

func newMonitorCmd(state *cliState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Manage monitors",
	}
	cmd.AddCommand(newMonitorListCmd(state))
	cmd.AddCommand(newMonitorPauseCmd(state))
	cmd.AddCommand(newMonitorResumeCmd(state))
	return cmd
}

func newMonitorListCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all monitors",
		RunE: func(cmd *cobra.Command, args []string) error {
			monitors, err := state.client.ListMonitors(context.Background())
			if err != nil {
				return fmt.Errorf("list monitors: %w", err)
			}
			if len(monitors) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no monitors found")
				return nil
			}
			if state.output == "json" {
				writeJSON(cmd.OutOrStdout(), monitors)
				return nil
			}
			printMonitorTable(cmd, monitors)
			return nil
		},
	}
}

func printMonitorTable(cmd *cobra.Command, monitors []hyperping.Monitor) {
	headers := []string{"UUID", "NAME", "URL", "STATUS", "PAUSED"}
	rows := make([][]string, len(monitors))
	for i, m := range monitors {
		rows[i] = []string{m.UUID, m.Name, m.URL, m.Status, strconv.FormatBool(m.Paused)}
	}
	writeTable(cmd.OutOrStdout(), headers, rows)
}

func newMonitorPauseCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "pause <uuid>",
		Short: "Pause a monitor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid := args[0]
			m, err := state.client.PauseMonitor(context.Background(), uuid)
			if err != nil {
				return fmt.Errorf("pause monitor: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "monitor %s paused (name: %s)\n", m.UUID, m.Name)
			return nil
		},
	}
}

func newMonitorResumeCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "resume <uuid>",
		Short: "Resume a paused monitor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid := args[0]
			m, err := state.client.ResumeMonitor(context.Background(), uuid)
			if err != nil {
				return fmt.Errorf("resume monitor: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "monitor %s resumed (name: %s)\n", m.UUID, m.Name)
			return nil
		},
	}
}
