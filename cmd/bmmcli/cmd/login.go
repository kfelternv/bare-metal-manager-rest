// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

// tokenResponse represents an OIDC token endpoint response
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the Carbide API",
	Long:  "Authenticate with the API using an OIDC provider and save the token to config.",
	RunE:  runLogin,
}

func init() {
	loginCmd.Flags().String("username", "", "username for authentication")
	loginCmd.Flags().String("password", "", "password for authentication")
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	if !hasAuthProviderConfig() {
		return fmt.Errorf(authGuidance, configFilePath())
	}

	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")

	// Fall back to config values
	if username == "" {
		username = viper.GetString("auth.oidc.username")
	}
	if password == "" {
		password = viper.GetString("auth.oidc.password")
	}

	// Prompt for username if still empty
	if username == "" {
		fmt.Print("Username: ")
		fmt.Scanln(&username)
	}

	// Prompt for password if still empty (hidden input)
	if password == "" {
		fmt.Print("Password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}
		fmt.Println() // newline after hidden input
		password = string(passwordBytes)
	}

	tokenURL := viper.GetString("auth.oidc.token_url")
	clientID := viper.GetString("auth.oidc.client_id")
	clientSecret := viper.GetString("auth.oidc.client_secret")

	formData := url.Values{
		"grant_type":    {"password"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"username":      {username},
		"password":      {password},
	}

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("requesting token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("parsing token response: %w", err)
	}

	// Save tokens to config
	viper.Set("auth.token", tokenResp.AccessToken)
	viper.Set("auth.refresh_token", tokenResp.RefreshToken)

	if err := saveConfig(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Login successful. Token saved to %s\n", configFilePath())
	return nil
}
