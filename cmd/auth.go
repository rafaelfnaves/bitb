package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/rafaelfnaves/bitb/internal/api"
	"github.com/rafaelfnaves/bitb/internal/config"
	"github.com/rafaelfnaves/bitb/internal/ui"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Bitbucket with an API token",
	RunE:  runAuthLogin,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE:  runAuthStatus,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials",
	RunE:  runAuthLogout,
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd, authStatusCmd, authLogoutCmd)
}

func runAuthLogin(cmd *cobra.Command, _ []string) error {
	fmt.Println(ui.StyleTitle.Render("Bitbucket CLI — Login"))
	fmt.Println()
	fmt.Println("Generate your API token at:")
	fmt.Println(ui.StyleCyan.Render("  https://id.atlassian.com/manage-profile/security/api-tokens"))
	fmt.Println()

	detectedWS, _, detected := config.DetectRepo()

	var workspace, email, token string

	if detected {
		workspace = detectedWS
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Workspace slug").
				Description("Your Bitbucket workspace (e.g. mycompany)").
				Value(&workspace).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("workspace is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Account email").
				Description("Your Atlassian account email").
				Value(&email).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("email is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("API token").
				Description("From id.atlassian.com/manage-profile/security/api-tokens").
				EchoMode(huh.EchoModePassword).
				Value(&token).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("token is required")
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("login cancelled")
	}

	fmt.Print("\nValidating credentials... ")
	client := api.NewClient(email, token)
	_, err := client.Get(fmt.Sprintf("/repositories/%s", workspace), nil)
	if err != nil {
		fmt.Println(ui.StyleRed.Render("FAILED"))
		return err
	}
	fmt.Println(ui.StyleGreen.Render("OK"))

	if err := config.Save(workspace, email, token); err != nil {
		return err
	}

	fmt.Printf("\n%s Logged in as %s (workspace: %s)\n",
		ui.StyleGreen.Render("✓"),
		ui.StyleBold.Render(email),
		ui.StyleBold.Render(workspace),
	)
	fmt.Println(ui.StyleDim.Render("Config saved to " + config.ConfigPath()))
	return nil
}

func runAuthStatus(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println(ui.StyleRed.Render("✗ Not logged in"))
		fmt.Println("Run: bb auth login")
		return nil
	}

	t := ui.NewTable("Key", "Value")
	t.AddRow("Workspace", cfg.Workspace)
	t.AddRow("Email", cfg.Email)
	t.AddRow("Token", ui.MaskToken(cfg.Token))
	t.AddRow("Config", config.ConfigPath())
	fmt.Print(t.Render())

	fmt.Print("\nConnectivity... ")
	client := api.NewClient(cfg.Email, cfg.Token)
	_, connErr := client.Get(fmt.Sprintf("/repositories/%s", cfg.Workspace), nil)
	if connErr != nil {
		fmt.Println(ui.StyleRed.Render("FAILED"))
		fmt.Println(ui.StyleRed.Render("  " + connErr.Error()))
	} else {
		fmt.Println(ui.StyleGreen.Render("Connected"))
	}
	return nil
}

func runAuthLogout(cmd *cobra.Command, _ []string) error {
	if err := config.Remove(); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Not logged in.")
			return nil
		}
		return err
	}
	fmt.Println(ui.StyleGreen.Render("✓ Logged out"))
	return nil
}
