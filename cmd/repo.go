package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/rafaelfernandes/bb/internal/api"
	"github.com/rafaelfernandes/bb/internal/config"
	"github.com/rafaelfernandes/bb/internal/ui"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repositories",
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories in workspace",
	RunE:  runRepoList,
}

var repoViewCmd = &cobra.Command{
	Use:   "view",
	Short: "View current repository info",
	RunE:  runRepoView,
}

func init() {
	rootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoListCmd, repoViewCmd)

	repoListCmd.Flags().StringP("workspace", "w", "", "Workspace slug")
	repoListCmd.Flags().IntP("limit", "n", 50, "Maximum number of repos to show")
	repoListCmd.Flags().Bool("json", false, "Output raw JSON")
	repoListCmd.Flags().Bool("web", false, "Open in browser")

	repoViewCmd.Flags().StringP("workspace", "w", "", "Workspace slug")
	repoViewCmd.Flags().StringP("repo", "r", "", "Repository (workspace/slug or slug)")
	repoViewCmd.Flags().Bool("json", false, "Output raw JSON")
	repoViewCmd.Flags().Bool("web", false, "Open in browser")
}

type repository struct {
	Slug        string `json:"slug"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Language    string `json:"language"`
	IsPrivate   bool   `json:"is_private"`
	UpdatedOn   string `json:"updated_on"`
	CreatedOn   string `json:"created_on"`
	Size        int    `json:"size"`
	Mainbranch  struct {
		Name string `json:"name"`
	} `json:"mainbranch"`
	Links struct {
		HTML struct{ Href string `json:"href"` } `json:"html"`
		Clone []struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"clone"`
	} `json:"links"`
}

func runRepoList(cmd *cobra.Command, _ []string) error {
	cfg := configFromCtx(cmd)
	client := clientFromCtx(cmd)

	wsFlag, _ := cmd.Flags().GetString("workspace")
	limit, _ := cmd.Flags().GetInt("limit")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	web, _ := cmd.Flags().GetBool("web")

	ws := wsFlag
	if ws == "" {
		ws = cfg.Workspace
	}

	if web {
		openURL(fmt.Sprintf("https://bitbucket.org/%s", ws))
		return nil
	}

	params := url.Values{"sort": {"-updated_on"}}
	repos, err := api.Paginate[repository](client, fmt.Sprintf("/repositories/%s", ws), params, limit)
	if err != nil {
		return err
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(repos, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found.")
		return nil
	}

	fmt.Println(ui.StyleTitle.Render(fmt.Sprintf("Repositories — %s", ws)))
	fmt.Println()

	t := ui.NewTable("Repository", "Language", "Visibility", "Updated")
	for _, r := range repos {
		visibility := ui.StyleGreen.Render("public")
		if r.IsPrivate {
			visibility = ui.StyleDim.Render("private")
		}
		lang := r.Language
		if lang == "" {
			lang = "—"
		}
		t.AddRow(
			ui.StyleCyan.Render(r.Slug),
			lang,
			visibility,
			ui.FormatDate(r.UpdatedOn),
		)
	}
	fmt.Print(t.Render())
	fmt.Printf("\n%s\n", ui.StyleDim.Render(fmt.Sprintf("%d repository(ies)", len(repos))))
	return nil
}

func runRepoView(cmd *cobra.Command, _ []string) error {
	cfg := configFromCtx(cmd)
	client := clientFromCtx(cmd)

	wsFlag, _ := cmd.Flags().GetString("workspace")
	repoFlag, _ := cmd.Flags().GetString("repo")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	web, _ := cmd.Flags().GetBool("web")

	ws, slug, err := config.ResolveRepo(wsFlag, repoFlag, cfg)
	if err != nil {
		return err
	}

	data, err := client.Get(fmt.Sprintf("/repositories/%s/%s", ws, slug), nil)
	if err != nil {
		return err
	}

	if jsonOutput {
		fmt.Println(string(data))
		return nil
	}

	var repo repository
	if err := json.Unmarshal(data, &repo); err != nil {
		return err
	}

	if web {
		openURL(repo.Links.HTML.Href)
		return nil
	}

	visibility := ui.StyleGreen.Render("public")
	if repo.IsPrivate {
		visibility = ui.StyleYellow.Render("private")
	}

	fmt.Printf("%s  %s  %s\n\n",
		ui.StyleTitle.Render(repo.FullName),
		visibility,
		ui.StyleDim.Render(repo.Language),
	)

	if repo.Description != "" {
		fmt.Println(repo.Description)
		fmt.Println()
	}

	t := ui.NewTable("Key", "Value")
	t.AddRow("Main branch", repo.Mainbranch.Name)
	t.AddRow("Created", ui.FormatDate(repo.CreatedOn))
	t.AddRow("Updated", ui.FormatDate(repo.UpdatedOn))
	t.AddRow("Size", fmt.Sprintf("%s KB", fmt.Sprintf("%d", repo.Size/1024+1)))

	for _, clone := range repo.Links.Clone {
		t.AddRow("Clone ("+clone.Name+")", clone.Href)
	}

	fmt.Print(t.Render())
	fmt.Println()
	fmt.Println(ui.StyleDim.Render(repo.Links.HTML.Href))
	return nil
}
