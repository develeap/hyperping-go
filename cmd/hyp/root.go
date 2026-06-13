// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"
	"os"

	hyperping "github.com/develeap/hyperping-go"
	"github.com/spf13/cobra"
)

type cliState struct {
	client  *hyperping.Client
	apiKey  string
	baseURL string
	output  string
}

func newRootCmd(version, commit, date string) *cobra.Command {
	state := &cliState{}

	root := &cobra.Command{
		Use:          "hyp",
		Short:        "Hyperping CLI",
		Long:         "hyp is the command-line interface for Hyperping uptime monitoring.",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if state.apiKey == "" {
				state.apiKey = os.Getenv("HYPERPING_API_KEY")
			}
			if state.apiKey == "" {
				return fmt.Errorf("API key required: set --api-key flag or HYPERPING_API_KEY env var")
			}
			opts := []hyperping.Option{hyperping.WithMaxRetries(0)}
			if state.baseURL != "" {
				opts = append(opts, hyperping.WithBaseURL(state.baseURL))
			}
			state.client = hyperping.NewClient(state.apiKey, opts...)
			return nil
		},
	}

	root.PersistentFlags().StringVar(&state.apiKey, "api-key", "", "Hyperping API key (overrides HYPERPING_API_KEY env var)")
	root.PersistentFlags().StringVar(&state.output, "output", "table", "Output format: table or json")
	root.PersistentFlags().StringVar(&state.baseURL, "base-url", "", "Custom API base URL")

	root.AddCommand(newVersionCmd(version, commit, date))
	root.AddCommand(newMonitorCmd(state))
	root.AddCommand(newIncidentCmd(state))
	root.AddCommand(newStatuspageCmd(state))

	return root
}
