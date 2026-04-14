package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	appName = "forgejo-cli"
)

// Config holds all CLI configuration
type Config struct {
	Server string
	Token  string
	Owner  string
	Proxy  string
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", appName), nil
}

// Load reads config from file + env + flags
func Load() (*Config, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(dir)
	viper.AddConfigPath(".")

	// Env overrides
	viper.SetEnvPrefix("FORGEJO_CLI")
	viper.AutomaticEnv()
	viper.BindEnv("server", "FORGEJO_SERVER")
	viper.BindEnv("token", "FORGEJO_TOKEN")
	viper.BindEnv("owner", "FORGEJO_OWNER")
	viper.BindEnv("proxy", "FORGEJO_PROXY")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config: %w", err)
		}
	}

	cfg := &Config{
		Server: viper.GetString("server"),
		Token:  viper.GetString("token"),
		Owner:  viper.GetString("owner"),
		Proxy:  viper.GetString("proxy"),
	}

	if cfg.Server == "" {
		return nil, fmt.Errorf("server URL is required (set in config or FORGEJO_CLI_SERVER env)")
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("API token is required (set in config or FORGEJO_CLI_TOKEN env)")
	}

	return cfg, nil
}

// InitConfig creates a default config file with restricted permissions
func InitConfig(server, token, owner, proxy string) error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	viper.Set("server", server)
	viper.Set("token", token)
	viper.Set("owner", owner)
	viper.Set("proxy", proxy)

	configPath := filepath.Join(dir, "config.yaml")
	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	// Restrict permissions: only owner can read/write
	if err := os.Chmod(configPath, 0600); err != nil {
		return fmt.Errorf("setting config permissions: %w", err)
	}

	fmt.Printf("Config written to %s\n", configPath)
	return nil
}
