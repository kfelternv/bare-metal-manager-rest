// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var cfgFile string

// Config represents the CLI configuration stored in ~/.bmm/config.yaml
type Config struct {
	API  APIConfig  `yaml:"api"`
	Auth AuthConfig `yaml:"auth"`
}

// APIConfig holds server connection settings
type APIConfig struct {
	Base string `yaml:"base"`
	Org  string `yaml:"org"`
	Name string `yaml:"name"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	OIDC  OIDCConfig `yaml:"oidc"`
	Token string     `yaml:"token"`
}

// OIDCConfig holds OIDC provider settings used by bmm login
type OIDCConfig struct {
	TokenURL     string `yaml:"token_url"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
}

var rootCmd = &cobra.Command{
	Use:   "bmm",
	Short: "Bare Metal Manager CLI",
	Long:  "CLI for interacting with the Carbide REST API.",
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ~/.bmm/config.yaml)")
	rootCmd.PersistentFlags().String("org", "", "override org from config")
	rootCmd.PersistentFlags().String("base", "", "override API base URL")
	rootCmd.PersistentFlags().StringP("output", "o", "", "output format: json, yaml (default from config)")

	viper.BindPFlag("api.org", rootCmd.PersistentFlags().Lookup("org"))
	viper.BindPFlag("api.base", rootCmd.PersistentFlags().Lookup("base"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error finding home directory:", err)
			os.Exit(1)
		}

		configDir := filepath.Join(home, ".bmm")
		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// Set defaults targeting local kind cluster
	viper.SetDefault("api.base", "http://localhost:8388")
	viper.SetDefault("api.org", "test-org")
	viper.SetDefault("api.name", "carbide")
	viper.SetDefault("auth.oidc.token_url", "http://localhost:8080/realms/carbide-dev/protocol/openid-connect/token")
	viper.SetDefault("auth.oidc.client_id", "carbide-api")
	viper.SetDefault("auth.oidc.client_secret", "carbide-local-secret")
	viper.SetDefault("auth.oidc.username", "admin@example.com")
	viper.SetDefault("auth.oidc.password", "adminpassword")

	viper.SetEnvPrefix("BMM")
	viper.AutomaticEnv()

	// Read config file if it exists; ignore error if not found
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintln(os.Stderr, "Error reading config:", err)
		}
	}
}

// configFilePath returns the path to the config file being used, or the default path
func configFilePath() string {
	if cfgFile != "" {
		return cfgFile
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".bmm", "config.yaml")
}

const configHeader = `# BMM CLI configuration
#
# API connection:
#
#   api.base -- server URL (e.g. http://localhost:8388)
#   api.org  -- organization name used in API paths
#   api.name -- the API path segment (default: carbide). This value is baked
#               into the generated Go client. If your server uses a different
#               api.name, you must update the OpenAPI spec and regenerate the
#               client with: make set-api-name API_NAME=myname generate-client
#               then rebuild: make build-cli
#
# Authentication (choose one):
#
#   Option 1 -- OIDC provider (e.g. Keycloak)
#
#     Configure the auth.oidc section below, then run: bmm login
#
#     Example for Keycloak:
#       auth:
#         oidc:
#           token_url: http://localhost:8080/realms/my-realm/protocol/openid-connect/token
#           client_id: my-client
#           client_secret: my-secret
#           username: user@example.com
#           password: mypassword
#
#   Option 2 -- API key / Bearer token
#
#     If you already have a token (e.g. from NGC or another provider),
#     set auth.token directly. No bmm login needed.
#
#     Example:
#       auth:
#         token: eyJhbGciOiJSUzI1NiIs...
#
`

const authGuidance = `No authentication token found. Set up auth in %s using one of:

  1) OIDC provider: configure the auth.oidc section
     (token_url, client_id, client_secret, and optionally username/password),
     then run: bmm login

  2) API key: set auth.token directly in your config file, e.g.
       auth:
         token: <your-bearer-token>

Run "bmm init" to generate a sample config.`

// hasAuthProviderConfig returns true when OIDC provider settings are configured
func hasAuthProviderConfig() bool {
	return viper.GetString("auth.oidc.token_url") != "" &&
		viper.GetString("auth.oidc.client_id") != "" &&
		viper.GetString("auth.oidc.client_secret") != ""
}

// getAuthToken returns the stored auth token or an error with guidance
func getAuthToken() (string, error) {
	token := viper.GetString("auth.token")
	if token != "" {
		return token, nil
	}
	return "", fmt.Errorf(authGuidance, configFilePath())
}

// saveConfig writes the current viper config to the config file
func saveConfig() error {
	cfgPath := configFilePath()

	// Ensure directory exists
	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	cfg := Config{
		API: APIConfig{
			Base: viper.GetString("api.base"),
			Org:  viper.GetString("api.org"),
			Name: viper.GetString("api.name"),
		},
		Auth: AuthConfig{
			OIDC: OIDCConfig{
				TokenURL:     viper.GetString("auth.oidc.token_url"),
				ClientID:     viper.GetString("auth.oidc.client_id"),
				ClientSecret: viper.GetString("auth.oidc.client_secret"),
				Username:     viper.GetString("auth.oidc.username"),
				Password:     viper.GetString("auth.oidc.password"),
			},
			Token: viper.GetString("auth.token"),
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	output := configHeader + string(data)

	if err := os.WriteFile(cfgPath, []byte(output), 0600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
