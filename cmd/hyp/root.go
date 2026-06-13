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
	output  string
	baseURL string
	apiKey  string
}

func newRootCmd(version, commit, date string) *cobra.Command {
	state := &cliState{}

	root := &cobra.Command{
		Use:   "hyp",
		Short: "hyp - Hyperping CLI",
		Long:  "hyp is a command-line interface for the Hyperping uptime monitoring platform.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			key := state.apiKey
			if key == "" {
				key = os.Getenv("HYPERPING_API_KEY")
			}
			if key == "" {
				return fmt.Errorf("API key required: set --api-key flag or HYPERPING_API_KEY environment variable")
			}
			opts := []hyperping.Option{
				hyperping.WithNoCircuitBreaker(),
			}
			if state.baseURL != "" {
				opts = append(opts, hyperping.WithBaseURL(state.baseURL))
			}
			state.client = hyperping.NewClient(key, opts...)
			return nil
		},
	}

	root.PersistentFlags().StringVar(&state.apiKey, "api-key", "", "Hyperping API key (overrides HYPERPING_API_KEY)")
	root.PersistentFlags().StringVarP(&state.output, "output", "o", "table", "Output format: table or json")
	root.PersistentFlags().StringVar(&state.baseURL, "base-url", "", "Override Hyperping API base URL")

	root.AddCommand(newVersionCmd(version, commit, date))
	root.AddCommand(newMonitorCmd(state))
	root.AddCommand(newIncidentCmd(state))
	root.AddCommand(newStatuspageCmd(state))
	root.AddCommand(newTenantCmd(state))

	return root
}
