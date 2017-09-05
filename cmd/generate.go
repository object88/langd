package cmd

import (
	"fmt"

	"github.com/object88/langd/client"
	"github.com/spf13/cobra"
)

func createGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "generate will create a new UUID",
		RunE:  run,
	}

	return cmd
}

func run(_ *cobra.Command, _ []string) error {
	c, err := client.NewClient()
	if err != nil {
		return err
	}

	if uuid, ok := c.GenerateUUID(); ok {
		c.DestroyClient()
		fmt.Printf("Received UUID: %s\n", uuid)
		return nil
	}

	if err = c.RequestNewService(); err != nil {
		return err
	}

	if uuid, ok := c.GenerateUUID(); ok {
		fmt.Printf("Received UUID: %s\n", uuid)
	}

	c.DestroyClient()

	return nil
}
