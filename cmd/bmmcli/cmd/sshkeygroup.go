// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/nvidia/bare-metal-manager-rest/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var sshKeyGroupCmd = &cobra.Command{
	Use:   "ssh-key-group",
	Short: "SSH key group operations",
}

var sshKeyGroupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all SSH key groups",
	RunE:  runSSHKeyGroupList,
}

var sshKeyGroupGetCmd = &cobra.Command{
	Use:   "get <ssh-key-group-id>",
	Short: "Get SSH key group details",
	Args:  cobra.ExactArgs(1),
	RunE:  runSSHKeyGroupGet,
}

var sshKeyGroupDeleteCmd = &cobra.Command{
	Use:   "delete <ssh-key-group-id>",
	Short: "Delete an SSH key group",
	Args:  cobra.ExactArgs(1),
	RunE:  runSSHKeyGroupDelete,
}

func init() {
	sshKeyGroupListCmd.Flags().Bool("json", false, "output raw JSON")

	rootCmd.AddCommand(sshKeyGroupCmd)
	sshKeyGroupCmd.AddCommand(sshKeyGroupListCmd)
	sshKeyGroupCmd.AddCommand(sshKeyGroupGetCmd)
	sshKeyGroupCmd.AddCommand(sshKeyGroupDeleteCmd)
}

func runSSHKeyGroupList(cmd *cobra.Command, args []string) error {
	org := viper.GetString("api.org")
	if org == "" {
		return fmt.Errorf("org is required: set api.org in config or pass --org")
	}

	apiClient := newAPIClient()
	ctx, err := apiContext()
	if err != nil {
		return err
	}

	groups, resp, err := apiClient.SSHKeyGroupAPI.GetAllSshKeyGroup(ctx, org).Execute()
	if err != nil {
		if resp != nil {
			body := tryReadBody(resp.Body)
			return fmt.Errorf("listing SSH key groups (HTTP %d): %v\n%s", resp.StatusCode, err, body)
		}
		return fmt.Errorf("listing SSH key groups: %v", err)
	}

	jsonFlag, _ := cmd.Flags().GetBool("json")
	outputFlag, _ := cmd.Root().PersistentFlags().GetString("output")
	switch {
	case jsonFlag || outputFlag == "json":
		return printJSON(os.Stdout, groups)
	case outputFlag == "yaml":
		return printYAML(os.Stdout, groups)
	default:
		return printSSHKeyGroupTable(os.Stdout, groups)
	}
}

func runSSHKeyGroupGet(cmd *cobra.Command, args []string) error {
	org := viper.GetString("api.org")
	if org == "" {
		return fmt.Errorf("org is required: set api.org in config or pass --org")
	}

	apiClient := newAPIClient()
	ctx, err := apiContext()
	if err != nil {
		return err
	}

	group, resp, err := apiClient.SSHKeyGroupAPI.GetSshKeyGroup(ctx, org, args[0]).Execute()
	if err != nil {
		if resp != nil {
			body := tryReadBody(resp.Body)
			return fmt.Errorf("getting SSH key group (HTTP %d): %v\n%s", resp.StatusCode, err, body)
		}
		return fmt.Errorf("getting SSH key group: %v", err)
	}

	outputFlag, _ := cmd.Root().PersistentFlags().GetString("output")
	switch outputFlag {
	case "yaml":
		return printYAML(os.Stdout, group)
	default:
		return printJSON(os.Stdout, group)
	}
}

func runSSHKeyGroupDelete(cmd *cobra.Command, args []string) error {
	org := viper.GetString("api.org")
	if org == "" {
		return fmt.Errorf("org is required: set api.org in config or pass --org")
	}

	apiClient := newAPIClient()
	ctx, err := apiContext()
	if err != nil {
		return err
	}

	resp, err := apiClient.SSHKeyGroupAPI.DeleteSshKeyGroup(ctx, org, args[0]).Execute()
	if err != nil {
		if resp != nil {
			body := tryReadBody(resp.Body)
			return fmt.Errorf("deleting SSH key group (HTTP %d): %v\n%s", resp.StatusCode, err, body)
		}
		return fmt.Errorf("deleting SSH key group: %v", err)
	}

	fmt.Fprintf(os.Stderr, "SSH key group deleted: %s\n", args[0])
	return nil
}

func printSSHKeyGroupTable(w io.Writer, groups []client.SshKeyGroup) error {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "NAME\tSTATUS\tAGE\tID")

	for _, g := range groups {
		name := ptrStr(g.Name)
		id := ptrStr(g.Id)
		status := ""
		if g.Status != nil {
			status = string(*g.Status)
		}
		age := "<unknown>"
		if g.Created != nil {
			age = formatAge(time.Since(*g.Created))
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", name, status, age, id)
	}

	return tw.Flush()
}
