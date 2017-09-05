package cmd

import (
	"github.com/object88/langd/server"
	"github.com/spf13/cobra"
)

func createServeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "serve",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.InitializeService()
		},
	}

	return cmd
}
