package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rafaelfnaves/bitb/internal/api"
	"github.com/rafaelfnaves/bitb/internal/config"
)

type contextKey string

const clientKey contextKey = "client"
const configKey contextKey = "config"

var version = "dev"

var rootCmd = &cobra.Command{
	Use:           "bitb",
	Short:         "Bitbucket CLI",
	Long: `bitb — Bitbucket Cloud CLI

A command-line interface for Bitbucket Cloud, inspired by GitHub's gh CLI.
Manage branches, pull requests, issues, and pipelines directly from your terminal.

AUTHENTICATION
  Run 'bitb auth login' to get started. You will need an API token from:
  https://id.atlassian.com/manage-profile/security/api-tokens

REPOSITORY AUTO-DETECTION
  bitb detects the current repository from the git remote automatically.
  Both SSH and HTTPS Bitbucket remotes are supported:
    git@bitbucket.org:myworkspace/myrepo.git
    https://bitbucket.org/myworkspace/myrepo

  Use --repo workspace/slug to target a specific repository explicitly.

EXAMPLES
  bitb auth login
  bitb pr list
  bitb repo list
  bitb pipeline list --branch main`,
	SilenceErrors: true,
	SilenceUsage:  true,
	Version:       version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Auth subcommands don't need a loaded client
		if cmd.Parent() != nil && cmd.Parent().Use == "auth" {
			return nil
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		client := api.NewClient(cfg.Email, cfg.Token)
		ctx := context.WithValue(cmd.Context(), clientKey, client)
		ctx = context.WithValue(ctx, configKey, cfg)
		cmd.SetContext(ctx)
		return nil
	}
}

func clientFromCtx(cmd *cobra.Command) *api.Client {
	return cmd.Context().Value(clientKey).(*api.Client)
}

func configFromCtx(cmd *cobra.Command) *config.Config {
	return cmd.Context().Value(configKey).(*config.Config)
}
