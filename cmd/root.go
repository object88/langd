package cmd

import "github.com/spf13/cobra"

// InitializeCommands sets up the cobra commands
func InitializeCommands() *cobra.Command {
	rootCmd := createRootCommand()

	rootCmd.AddCommand(
		createGenerateCommand(),
		createServeCommand(),
		createStartCommand(),
		createStopCommand(),
	)

	return rootCmd
}

func createRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "langd",
		Short: "langd exercises a client/server configuration as a single binary",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}

	return cmd
}
