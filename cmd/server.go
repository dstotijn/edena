package cmd

import (
	"github.com/spf13/cobra"

	"github.com/dstotijn/edena/pkg/http"
)

func init() {
	rootCmd.AddCommand(serverCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Runs a server for collecting and managing HTTP, SMTP and DNS traffic.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		httpServer := http.NewServer()

		if err := httpServer.Run(ctx); err != nil {
			return err
		}

		return nil
	},
}
