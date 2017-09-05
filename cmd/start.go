package cmd

import (
	"fmt"

	"github.com/object88/langd/client"
	"github.com/spf13/cobra"
)

func createStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the server",
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Printf("Starting...\n")
			c, err := client.NewClient()
			if err != nil {
				fmt.Printf("Failed to create client: %s\n", err.Error())
				return err
			}
			defer c.DestroyClient()

			fmt.Printf("Requesting startup...\n")
			err = c.RequestStartup()
			if err == nil {
				fmt.Printf("Running\n")
				return nil
			}
			fmt.Printf("Not yet started: %s\n", err.Error())

			if err = c.RequestNewService(); err != nil {
				fmt.Printf("Request for new service failed: %s\n", err.Error())
				return err
			}

			fmt.Printf("Started\n")
			return nil
		},
	}

	return cmd
}
