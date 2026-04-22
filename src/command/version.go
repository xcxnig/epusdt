package command

import (
	"fmt"

	"github.com/GMWalletApp/epusdt/config"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print build version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := fmt.Fprintf(cmd.OutOrStdout(),
			"version: %s\ncommit: %s\nbuilt: %s\n",
			config.GetAppVersion(),
			config.GetBuildCommit(),
			config.GetBuildDate(),
		)
		return err
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
