package cmd

import (
	"fmt"

	"github.com/jimboylabs/forgejocli/internal/config"
	"github.com/spf13/cobra"
)

var initServer, initToken, initOwner, initProxy string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize CLI configuration",
	Long:  "Create or update the configuration file with your Forgejo server details.",
	Example: `  forgejo init \
    --server http://forgejo.example.com:8080 \
    --token YOUR_API_TOKEN \
    --owner myorg \
    --proxy socks5://127.0.0.1:1080`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if initServer == "" || initToken == "" {
			return fmt.Errorf("both --server and --token are required")
		}
		return config.InitConfig(initServer, initToken, initOwner, initProxy)
	},
}

func init() {
	initCmd.Flags().StringVar(&initServer, "server", "", "Forgejo server URL")
	initCmd.Flags().StringVar(&initToken, "token", "", "API token")
	initCmd.Flags().StringVar(&initOwner, "owner", "", "Default owner/organization")
	initCmd.Flags().StringVar(&initProxy, "proxy", "", "SOCKS5 proxy URL (optional)")

	rootCmd.AddCommand(initCmd)
}
