package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jimboylabs/forgejocli/internal/config"
	apiclient "github.com/jimboylabs/forgejocli/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	cfg        *config.Config
	apiClient  *apiclient.Client
	flagOwner  string
	flagRepo   string
	flagOutput string // "text" or "json"
)

var rootCmd = &cobra.Command{
	Use:   "forgejo",
	Short: "CLI for self-hosted Forgejo",
	Long: `A command-line tool to interact with your self-hosted Forgejo server.
Manage repositories, issues, and pull requests from the terminal.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for 'init' command
		if cmd.Name() == "init" {
			return nil
		}

		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("config error: %w\nRun 'forgejo init' to set up configuration", err)
		}

		// Override owner from flag if provided
		if flagOwner != "" {
			cfg.Owner = flagOwner
		}

		apiClient, err = apiclient.NewClient(cfg.Server, cfg.Token, cfg.Proxy)
		if err != nil {
			return fmt.Errorf("creating API client: %w", err)
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
	rootCmd.PersistentFlags().StringVarP(&flagOwner, "owner", "o", "", "repository owner (overrides config)")
	rootCmd.PersistentFlags().StringVarP(&flagRepo, "repo", "r", "", "repository name")
	rootCmd.PersistentFlags().StringVarP(&flagOutput, "output", "O", "text", "output format: text or json")

	cobra.OnInitialize(func() {
		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
		}
	})
}

// outputJSON writes a JSON-serializable value to stdout
func outputJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// isJSON returns true if --output json is set
func isJSON() bool {
	return flagOutput == "json"
}

// outputError writes an error to stderr
func outputError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}
