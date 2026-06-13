// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"

	hyperping "github.com/develeap/hyperping-go"
	"github.com/spf13/cobra"
)

func newIncidentCmd(state *cliState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "incident",
		Short: "Manage incidents",
	}
	cmd.AddCommand(newIncidentCreateCmd(state))
	cmd.AddCommand(newIncidentResolveCmd(state))
	return cmd
}

func newIncidentCreateCmd(state *cliState) *cobra.Command {
	var title, text, incType, statuspage string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new incident",
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			if incType == "" {
				return fmt.Errorf("--type is required")
			}
			if statuspage == "" {
				return fmt.Errorf("--statuspage is required")
			}
			req := hyperping.CreateIncidentRequest{
				Title:       hyperping.LocalizedText{En: title},
				Text:        hyperping.LocalizedText{En: text},
				Type:        incType,
				StatusPages: []string{statuspage},
			}
			incident, err := state.client.CreateIncident(context.Background(), req)
			if err != nil {
				return fmt.Errorf("create incident: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "incident created: %s\n", incident.UUID)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Incident title (required)")
	cmd.Flags().StringVar(&text, "text", "", "Incident description")
	cmd.Flags().StringVar(&incType, "type", "", "Incident type: outage or incident (required)")
	cmd.Flags().StringVar(&statuspage, "statuspage", "", "Status page UUID to associate (required)")

	return cmd
}

func newIncidentResolveCmd(state *cliState) *cobra.Command {
	var message string

	cmd := &cobra.Command{
		Use:   "resolve <uuid>",
		Short: "Resolve an incident",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid := args[0]
			if message == "" {
				message = "Incident resolved."
			}
			incident, err := state.client.ResolveIncident(context.Background(), uuid, message)
			if err != nil {
				return fmt.Errorf("resolve incident: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "incident %s resolved\n", incident.UUID)
			return nil
		},
	}

	cmd.Flags().StringVar(&message, "message", "", "Resolution message")

	return cmd
}
