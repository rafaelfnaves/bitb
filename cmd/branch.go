package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rafaelfnaves/bitb/internal/api"
	"github.com/rafaelfnaves/bitb/internal/config"
	"github.com/rafaelfnaves/bitb/internal/ui"
)

var branchCmd = &cobra.Command{
	Use:   "branch",
	Short: "Manage branches",
}

var branchListCmd = &cobra.Command{
	Use:   "list",
	Short: "List remote branches",
	RunE:  runBranchList,
}

var branchDeleteCmd = &cobra.Command{
	Use:   "delete <branch>",
	Short: "Delete a remote branch",
	Args:  cobra.ExactArgs(1),
	RunE:  runBranchDelete,
}

func init() {
	rootCmd.AddCommand(branchCmd)
	branchCmd.AddCommand(branchListCmd, branchDeleteCmd)

	branchListCmd.Flags().StringP("workspace", "w", "", "Workspace slug")
	branchListCmd.Flags().StringP("repo", "r", "", "Repository (workspace/slug or slug)")
	branchListCmd.Flags().StringP("search", "s", "", "Filter by branch name")
	branchListCmd.Flags().StringP("author", "a", "", "Filter by author name or email")
	branchListCmd.Flags().IntP("limit", "n", 50, "Maximum number of branches to show")
	branchListCmd.Flags().Bool("json", false, "Output raw JSON")

	branchDeleteCmd.Flags().StringP("workspace", "w", "", "Workspace slug")
	branchDeleteCmd.Flags().StringP("repo", "r", "", "Repository (workspace/slug or slug)")
	branchDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation")
}

type branchTarget struct {
	Date    string `json:"date"`
	Message string `json:"message"`
	Author  struct {
		Raw  string `json:"raw"`
		User struct {
			DisplayName string `json:"display_name"`
		} `json:"user"`
	} `json:"author"`
}

type branch struct {
	Name   string       `json:"name"`
	Target branchTarget `json:"target"`
}

func runBranchList(cmd *cobra.Command, _ []string) error {
	cfg := configFromCtx(cmd)
	client := clientFromCtx(cmd)

	wsFlag, _ := cmd.Flags().GetString("workspace")
	repoFlag, _ := cmd.Flags().GetString("repo")
	search, _ := cmd.Flags().GetString("search")
	author, _ := cmd.Flags().GetString("author")
	limit, _ := cmd.Flags().GetInt("limit")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	ws, slug, err := config.ResolveRepo(wsFlag, repoFlag, cfg)
	if err != nil {
		return err
	}

	params := url.Values{}
	if search != "" {
		params.Set("q", fmt.Sprintf(`name ~ "%s"`, search))
	}

	branches, err := api.Paginate[branch](client, fmt.Sprintf("/repositories/%s/%s/refs/branches", ws, slug), params, limit)
	if err != nil {
		return err
	}

	if author != "" {
		filtered := branches[:0]
		for _, b := range branches {
			if strings.Contains(strings.ToLower(b.Target.Author.Raw), strings.ToLower(author)) {
				filtered = append(filtered, b)
			}
		}
		branches = filtered
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(branches, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(branches) == 0 {
		fmt.Println("No branches found.")
		return nil
	}

	fmt.Println(ui.StyleTitle.Render(fmt.Sprintf("Branches — %s/%s", ws, slug)))
	fmt.Println()

	t := ui.NewTable("Branch", "Author", "Updated", "Last Commit")
	for _, b := range branches {
		authorName := b.Target.Author.Raw
		if idx := strings.Index(authorName, " <"); idx > 0 {
			authorName = authorName[:idx]
		}
		message := strings.SplitN(b.Target.Message, "\n", 2)[0]
		t.AddRow(
			ui.StyleCyan.Render(b.Name),
			ui.Truncate(authorName, 25),
			ui.FormatDate(b.Target.Date),
			ui.Truncate(message, 50),
		)
	}
	fmt.Print(t.Render())
	fmt.Printf("\n%s\n", ui.StyleDim.Render(fmt.Sprintf("%d branch(es)", len(branches))))
	return nil
}

func runBranchDelete(cmd *cobra.Command, args []string) error {
	cfg := configFromCtx(cmd)
	client := clientFromCtx(cmd)

	wsFlag, _ := cmd.Flags().GetString("workspace")
	repoFlag, _ := cmd.Flags().GetString("repo")
	yes, _ := cmd.Flags().GetBool("yes")

	ws, slug, err := config.ResolveRepo(wsFlag, repoFlag, cfg)
	if err != nil {
		return err
	}

	branchName := args[0]

	if !yes {
		fmt.Printf("Delete branch %s from %s/%s? [y/N] ", ui.StyleRed.Render(branchName), ws, slug)
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	if err := client.Delete(fmt.Sprintf("/repositories/%s/%s/refs/branches/%s", ws, slug, url.PathEscape(branchName))); err != nil {
		return err
	}

	fmt.Printf("%s Deleted branch %s\n", ui.StyleGreen.Render("✓"), ui.StyleBold.Render(branchName))
	return nil
}
