// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a sample config file",
	Long:  "Write a sample config file to ~/.bmm/config.yaml (or the path specified by --config). Will not overwrite an existing file unless --force is used.",
	RunE:  runInit,
}

func init() {
	initCmd.Flags().Bool("force", false, "overwrite existing config file")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	cfgPath := configFilePath()

	if !force {
		if _, err := os.Stat(cfgPath); err == nil {
			return fmt.Errorf("config file already exists at %s (use --force to overwrite)", cfgPath)
		}
	}

	// Write defaults via saveConfig which uses viper defaults
	if err := saveConfig(); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("Config written to %s\n", cfgPath)
	return nil
}
