// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/nvidia/bare-metal-manager-rest/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var siteCmd = &cobra.Command{
	Use:   "site",
	Short: "Site operations",
}

var siteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sites",
	RunE:  runSiteList,
}

var siteGetCmd = &cobra.Command{
	Use:   "get <site-id>",
	Short: "Get site details",
	Args:  cobra.ExactArgs(1),
	RunE:  runSiteGet,
}

var siteCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a site",
	RunE:  runSiteCreate,
}

var siteUpdateCmd = &cobra.Command{
	Use:   "update <site-id>",
	Short: "Update a site",
	Args:  cobra.ExactArgs(1),
	RunE:  runSiteUpdate,
}

var siteDeleteCmd = &cobra.Command{
	Use:   "delete <site-id>",
	Short: "Delete a site",
	Args:  cobra.ExactArgs(1),
	RunE:  runSiteDelete,
}

func init() {
	siteListCmd.Flags().Bool("json", false, "output raw JSON")

	siteCreateCmd.Flags().String("name", "", "name for the site (required)")
	siteCreateCmd.Flags().String("description", "", "optional description")
	siteCreateCmd.MarkFlagRequired("name")

	siteUpdateCmd.Flags().String("name", "", "new name")
	siteUpdateCmd.Flags().String("description", "", "new description")

	rootCmd.AddCommand(siteCmd)
	siteCmd.AddCommand(siteListCmd)
	siteCmd.AddCommand(siteGetCmd)
	siteCmd.AddCommand(siteCreateCmd)
	siteCmd.AddCommand(siteUpdateCmd)
	siteCmd.AddCommand(siteDeleteCmd)
}

func newAPIClient() *client.APIClient {
	cfg := client.NewConfiguration()
	cfg.Servers = client.ServerConfigurations{
		{
			URL:         viper.GetString("api.base"),
			Description: "Configured server",
		},
	}
	return client.NewAPIClient(cfg)
}

func apiContext() (context.Context, error) {
	token, err := getAuthToken()
	if err != nil {
		return nil, err
	}
	ctx := context.WithValue(context.Background(), client.ContextAccessToken, token)
	return ctx, nil
}

func runSiteList(cmd *cobra.Command, args []string) error {
	org := viper.GetString("api.org")
	if org == "" {
		return fmt.Errorf("org is required: set api.org in config or pass --org")
	}

	apiClient := newAPIClient()
	ctx, err := apiContext()
	if err != nil {
		return err
	}

	sites, resp, err := apiClient.SiteAPI.GetAllSite(ctx, org).Execute()
	if err != nil {
		if resp != nil {
			return fmt.Errorf("listing sites (HTTP %d): %v", resp.StatusCode, err)
		}
		return fmt.Errorf("listing sites: %v", err)
	}

	jsonFlag, _ := cmd.Flags().GetBool("json")
	outputFlag, _ := cmd.Root().PersistentFlags().GetString("output")
	switch {
	case jsonFlag || outputFlag == "json":
		return printJSON(os.Stdout, sites)
	case outputFlag == "yaml":
		return printYAML(os.Stdout, sites)
	default:
		return printSiteTable(os.Stdout, sites)
	}
}

func runSiteGet(cmd *cobra.Command, args []string) error {
	org := viper.GetString("api.org")
	if org == "" {
		return fmt.Errorf("org is required: set api.org in config or pass --org")
	}

	apiClient := newAPIClient()
	ctx, err := apiContext()
	if err != nil {
		return err
	}

	site, resp, err := apiClient.SiteAPI.GetSite(ctx, org, args[0]).Execute()
	if err != nil {
		if resp != nil {
			body := tryReadBody(resp.Body)
			return fmt.Errorf("getting site (HTTP %d): %v\n%s", resp.StatusCode, err, body)
		}
		return fmt.Errorf("getting site: %v", err)
	}

	outputFlag, _ := cmd.Root().PersistentFlags().GetString("output")
	switch outputFlag {
	case "yaml":
		return printYAML(os.Stdout, site)
	default:
		return printJSON(os.Stdout, site)
	}
}

func runSiteCreate(cmd *cobra.Command, args []string) error {
	org := viper.GetString("api.org")
	if org == "" {
		return fmt.Errorf("org is required: set api.org in config or pass --org")
	}

	name, _ := cmd.Flags().GetString("name")
	description, _ := cmd.Flags().GetString("description")

	apiClient := newAPIClient()
	ctx, err := apiContext()
	if err != nil {
		return err
	}

	req := client.NewSiteCreateRequest(name)
	if description != "" {
		req.SetDescription(description)
	}

	site, resp, err := apiClient.SiteAPI.CreateSite(ctx, org).SiteCreateRequest(*req).Execute()
	if err != nil {
		if resp != nil {
			body := tryReadBody(resp.Body)
			return fmt.Errorf("creating site (HTTP %d): %v\n%s", resp.StatusCode, err, body)
		}
		return fmt.Errorf("creating site: %v", err)
	}

	outputFlag, _ := cmd.Root().PersistentFlags().GetString("output")
	switch outputFlag {
	case "yaml":
		return printYAML(os.Stdout, site)
	default:
		fmt.Fprintf(os.Stderr, "Site created: %s (%s)\n", ptrStr(site.Name), ptrStr(site.Id))
		return printJSON(os.Stdout, site)
	}
}

func runSiteUpdate(cmd *cobra.Command, args []string) error {
	org := viper.GetString("api.org")
	if org == "" {
		return fmt.Errorf("org is required: set api.org in config or pass --org")
	}

	apiClient := newAPIClient()
	ctx, err := apiContext()
	if err != nil {
		return err
	}

	req := client.NewSiteUpdateRequest()
	if cmd.Flags().Changed("name") {
		name, _ := cmd.Flags().GetString("name")
		req.SetName(name)
	}
	if cmd.Flags().Changed("description") {
		description, _ := cmd.Flags().GetString("description")
		req.SetDescription(description)
	}

	site, resp, err := apiClient.SiteAPI.UpdateSite(ctx, org, args[0]).SiteUpdateRequest(*req).Execute()
	if err != nil {
		if resp != nil {
			body := tryReadBody(resp.Body)
			return fmt.Errorf("updating site (HTTP %d): %v\n%s", resp.StatusCode, err, body)
		}
		return fmt.Errorf("updating site: %v", err)
	}

	outputFlag, _ := cmd.Root().PersistentFlags().GetString("output")
	switch outputFlag {
	case "yaml":
		return printYAML(os.Stdout, site)
	default:
		fmt.Fprintf(os.Stderr, "Site updated: %s (%s)\n", ptrStr(site.Name), ptrStr(site.Id))
		return printJSON(os.Stdout, site)
	}
}

func runSiteDelete(cmd *cobra.Command, args []string) error {
	org := viper.GetString("api.org")
	if org == "" {
		return fmt.Errorf("org is required: set api.org in config or pass --org")
	}

	apiClient := newAPIClient()
	ctx, err := apiContext()
	if err != nil {
		return err
	}

	resp, err := apiClient.SiteAPI.DeleteSite(ctx, org, args[0]).Execute()
	if err != nil {
		if resp != nil {
			body := tryReadBody(resp.Body)
			return fmt.Errorf("deleting site (HTTP %d): %v\n%s", resp.StatusCode, err, body)
		}
		return fmt.Errorf("deleting site: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Site deleted: %s\n", args[0])
	return nil
}

func printJSON(w io.Writer, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling output: %v", err)
	}
	fmt.Fprintln(w, string(data))
	return nil
}

func printYAML(w io.Writer, v interface{}) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshaling output: %v", err)
	}
	fmt.Fprint(w, string(data))
	return nil
}

func printSiteTable(w io.Writer, sites []client.Site) error {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "NAME\tSTATUS\tONLINE\tAGE\tID")

	for _, s := range sites {
		name := ptrStr(s.Name)
		id := ptrStr(s.Id)
		status := ""
		if s.Status != nil {
			status = string(*s.Status)
		}
		online := "Unknown"
		if s.IsOnline != nil {
			if *s.IsOnline {
				online = "True"
			} else {
				online = "False"
			}
		}
		age := "<unknown>"
		if s.Created != nil {
			age = formatAge(time.Since(*s.Created))
		}

		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", name, status, online, age, id)
	}

	return tw.Flush()
}

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func formatAge(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	}
}
