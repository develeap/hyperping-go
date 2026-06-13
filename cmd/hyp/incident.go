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

type incidentCreateFlags struct {
	title      string
	text       string
	incType    string
	statuspage string
}

func newIncidentCreateCmd(state *cliState) *cobra.Command {
	flags := &incidentCreateFlags{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an incident",
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.title == "" || flags.text == "" || flags.incType == "" || flags.statuspage == "" {
				return fmt.Errorf("--title, --text, --type, and --statuspage are required")
			}
			req := hyperping.CreateIncidentRequest{
				Title:       hyperping.LocalizedText{En: flags.title},
				Text:        hyperping.LocalizedText{En: flags.text},
				Type:        flags.incType,
				StatusPages: []string{flags.statuspage},
			}
			incident, err := state.client.CreateIncident(context.Background(), req)
			if err != nil {
				return fmt.Errorf("create incident: %w", err)
			}
			if state.output == "json" {
				return writeJSON(cmd.OutOrStdout(), incident)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "incident %s created\n", incident.UUID)
			return nil
		},
	}
	cmd.Flags().StringVar(&flags.title, "title", "", "Incident title (English)")
	cmd.Flags().StringVar(&flags.text, "text", "", "Incident description (English)")
	cmd.Flags().StringVar(&flags.incType, "type", "", "Incident type (e.g. monitoring, outage)")
	cmd.Flags().StringVar(&flags.statuspage, "statuspage", "", "Status page UUID to associate")
	return cmd
}

func newIncidentResolveCmd(state *cliState) *cobra.Command {
	var message string
	cmd := &cobra.Command{
		Use:   "resolve <uuid>",
		Short: "Resolve an incident",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			incident, err := state.client.ResolveIncident(context.Background(), args[0], message)
			if err != nil {
				return fmt.Errorf("resolve incident: %w", err)
			}
			if state.output == "json" {
				return writeJSON(cmd.OutOrStdout(), incident)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "incident %s resolved\n", incident.UUID)
			return nil
		},
	}
	cmd.Flags().StringVar(&message, "message", "", "Resolution message")
	return cmd
}
