// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"

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
				return writeJSON(cmd.OutOrStdout(), monitors)
			}
			writeTable(cmd.OutOrStdout(),
				[]string{"UUID", "NAME", "URL", "STATUS", "PAUSED"},
				monitorsToRows(monitors))
			return nil
		},
	}
}

func monitorsToRows(monitors []hyperping.Monitor) [][]string {
	rows := make([][]string, len(monitors))
	for i, m := range monitors {
		paused := "false"
		if m.Paused {
			paused = "true"
		}
		rows[i] = []string{m.UUID, m.Name, m.URL, m.Status, paused}
	}
	return rows
}

func newMonitorPauseCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "pause <uuid>",
		Short: "Pause a monitor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := state.client.PauseMonitor(context.Background(), args[0])
			if err != nil {
				return fmt.Errorf("pause monitor: %w", err)
			}
			if state.output == "json" {
				return writeJSON(cmd.OutOrStdout(), m)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "monitor %s paused\n", m.UUID)
			return nil
		},
	}
}

func newMonitorResumeCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "resume <uuid>",
		Short: "Resume a monitor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := state.client.ResumeMonitor(context.Background(), args[0])
			if err != nil {
				return fmt.Errorf("resume monitor: %w", err)
			}
			if state.output == "json" {
				return writeJSON(cmd.OutOrStdout(), m)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "monitor %s resumed\n", m.UUID)
			return nil
		},
	}
}
