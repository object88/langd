package cmd

import (
	"fmt"

	"github.com/object88/langd/client"
	"github.com/spf13/cobra"
)

func createLoadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "load",
		Short: "Report the CPU and Memory load",
		RunE: func(_ *cobra.Command, _ []string) error {
			c, err := client.NewClient()
			if err != nil {
				fmt.Printf("Failed to create client: %s\n", err.Error())
				return err
			}
			defer c.DestroyClient()

			c.RequestLoad()

			return nil
		},
	}
	return cmd
}
