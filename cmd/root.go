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

var rootCmd = &cobra.Command{
	Use:           "bitb",
	Short:         "Bitbucket CLI",
	Long:          "A GitHub CLI-inspired tool for Bitbucket Cloud repositories.",
	SilenceErrors: true,
	SilenceUsage:  true,
	Version:       "0.1.0",
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
