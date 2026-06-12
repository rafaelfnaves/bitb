package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/charmbracelet/glamour"

	"github.com/rafaelfnaves/bitb/internal/api"
	"github.com/rafaelfnaves/bitb/internal/config"
	"github.com/rafaelfnaves/bitb/internal/ui"
)

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Manage issues",
}

func init() {
	rootCmd.AddCommand(issueCmd)
	issueCmd.AddCommand(issueListCmd, issueViewCmd, issueCreateCmd)

	issueListCmd.Flags().StringP("workspace", "w", "", "Workspace slug")
	issueListCmd.Flags().StringP("repo", "r", "", "Repository (workspace/slug or slug)")
	issueListCmd.Flags().StringP("status", "s", "new,open", "Comma-separated statuses: new,open,resolved,on hold,invalid,duplicate,wontfix,closed")
	issueListCmd.Flags().StringP("assignee", "a", "", "Filter by assignee username")
	issueListCmd.Flags().StringP("kind", "k", "", "Filter by kind: bug, enhancement, proposal, task")
	issueListCmd.Flags().IntP("limit", "n", 30, "Maximum number of issues to show")
	issueListCmd.Flags().Bool("json", false, "Output raw JSON")

	issueViewCmd.Flags().StringP("workspace", "w", "", "Workspace slug")
	issueViewCmd.Flags().StringP("repo", "r", "", "Repository (workspace/slug or slug)")
	issueViewCmd.Flags().BoolP("comments", "c", false, "Show comments")
	issueViewCmd.Flags().Bool("json", false, "Output raw JSON")
	issueViewCmd.Flags().Bool("web", false, "Open in browser")

	issueCreateCmd.Flags().StringP("workspace", "w", "", "Workspace slug")
	issueCreateCmd.Flags().StringP("repo", "r", "", "Repository (workspace/slug or slug)")
	issueCreateCmd.Flags().StringP("title", "t", "", "Issue title")
	issueCreateCmd.Flags().StringP("body", "b", "", "Issue description")
	issueCreateCmd.Flags().StringP("kind", "k", "task", "Kind: bug, enhancement, proposal, task")
	issueCreateCmd.Flags().StringP("priority", "p", "major", "Priority: trivial, minor, major, critical, blocker")
}

type issue struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Kind     string `json:"kind"`
	Priority string `json:"priority"`
	Content  struct {
		Raw string `json:"raw"`
	} `json:"content"`
	CreatedOn string `json:"created_on"`
	UpdatedOn string `json:"updated_on"`
	Reporter  struct {
		DisplayName string `json:"display_name"`
	} `json:"reporter"`
	Assignee *struct {
		DisplayName string `json:"display_name"`
	} `json:"assignee"`
	Links struct {
		HTML struct{ Href string `json:"href"` } `json:"html"`
	} `json:"links"`
}

type issueComment struct {
	ID      int `json:"id"`
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
	CreatedOn string `json:"created_on"`
	Author    struct {
		DisplayName string `json:"display_name"`
	} `json:"author"`
}

var issueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues",
	RunE:  runIssueList,
}

var issueViewCmd = &cobra.Command{
	Use:   "view <id>",
	Short: "View an issue",
	Args:  cobra.ExactArgs(1),
	RunE:  runIssueView,
}

var issueCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an issue",
	RunE:  runIssueCreate,
}

func issueNotEnabled() error {
	msg := "this repository does not have Bitbucket Issues enabled\n" +
		ui.StyleDim.Render("  (the repo may use Jira or another tracker instead)")
	return fmt.Errorf("%s", msg)
}

func runIssueList(cmd *cobra.Command, _ []string) error {
	cfg := configFromCtx(cmd)
	client := clientFromCtx(cmd)

	wsFlag, _ := cmd.Flags().GetString("workspace")
	repoFlag, _ := cmd.Flags().GetString("repo")
	status, _ := cmd.Flags().GetString("status")
	assignee, _ := cmd.Flags().GetString("assignee")
	kind, _ := cmd.Flags().GetString("kind")
	limit, _ := cmd.Flags().GetInt("limit")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	ws, slug, err := config.ResolveRepo(wsFlag, repoFlag, cfg)
	if err != nil {
		return err
	}

	// Build query
	var qParts []string
	for _, s := range strings.Split(status, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			qParts = append(qParts, fmt.Sprintf(`status="%s"`, s))
		}
	}
	if assignee != "" {
		qParts = append(qParts, fmt.Sprintf(`assignee.username="%s"`, assignee))
	}
	if kind != "" {
		qParts = append(qParts, fmt.Sprintf(`kind="%s"`, kind))
	}

	params := url.Values{}
	if len(qParts) > 0 {
		params.Set("q", strings.Join(qParts, " AND "))
	}

	issues, err := api.Paginate[issue](client, fmt.Sprintf("/repositories/%s/%s/issues", ws, slug), params, limit)
	if err != nil {
		if apiErr, ok := err.(*api.APIError); ok && apiErr.StatusCode == 404 {
			return issueNotEnabled()
		}
		return err
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(issues, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(issues) == 0 {
		fmt.Println("No issues found.")
		return nil
	}

	fmt.Println(ui.StyleTitle.Render(fmt.Sprintf("Issues — %s/%s", ws, slug)))
	fmt.Println()

	t := ui.NewTable("ID", "Title", "Status", "Kind", "Reporter", "Updated")
	for _, i := range issues {
		assigneeName := "—"
		if i.Assignee != nil {
			assigneeName = i.Assignee.DisplayName
		}
		_ = assigneeName
		t.AddRow(
			ui.StyleID.Render(fmt.Sprintf("#%d", i.ID)),
			ui.Truncate(i.Title, 45),
			i.Status,
			i.Kind,
			i.Reporter.DisplayName,
			ui.FormatDate(i.UpdatedOn),
		)
	}
	fmt.Print(t.Render())
	fmt.Printf("\n%s\n", ui.StyleDim.Render(fmt.Sprintf("%d issue(s)", len(issues))))
	return nil
}

func runIssueView(cmd *cobra.Command, args []string) error {
	cfg := configFromCtx(cmd)
	client := clientFromCtx(cmd)

	wsFlag, _ := cmd.Flags().GetString("workspace")
	repoFlag, _ := cmd.Flags().GetString("repo")
	showComments, _ := cmd.Flags().GetBool("comments")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	web, _ := cmd.Flags().GetBool("web")

	ws, slug, err := config.ResolveRepo(wsFlag, repoFlag, cfg)
	if err != nil {
		return err
	}

	data, err := client.Get(fmt.Sprintf("/repositories/%s/%s/issues/%s", ws, slug, args[0]), nil)
	if err != nil {
		if apiErr, ok := err.(*api.APIError); ok && apiErr.StatusCode == 404 {
			return issueNotEnabled()
		}
		return err
	}

	if jsonOutput {
		fmt.Println(string(data))
		return nil
	}

	var i issue
	if err := json.Unmarshal(data, &i); err != nil {
		return err
	}

	if web {
		openURL(i.Links.HTML.Href)
		return nil
	}

	fmt.Printf("%s %s\n", ui.StyleID.Render(fmt.Sprintf("#%d", i.ID)), ui.StyleBold.Render(i.Title))
	fmt.Printf("%s  %s  %s  %s\n",
		i.Status,
		ui.StyleDim.Render(i.Kind),
		ui.StyleDim.Render("by "+i.Reporter.DisplayName),
		ui.StyleDim.Render(ui.FormatDate(i.CreatedOn)),
	)
	if i.Assignee != nil {
		fmt.Printf("%s %s\n", ui.StyleDim.Render("Assignee:"), i.Assignee.DisplayName)
	}
	fmt.Println()

	if i.Content.Raw != "" {
		renderer, err := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(100))
		if err == nil {
			rendered, err := renderer.Render(i.Content.Raw)
			if err == nil {
				fmt.Print(rendered)
			} else {
				fmt.Println(i.Content.Raw)
			}
		} else {
			fmt.Println(i.Content.Raw)
		}
	}

	fmt.Println(ui.StyleDim.Render(i.Links.HTML.Href))

	if showComments {
		commentsData, err := client.Get(fmt.Sprintf("/repositories/%s/%s/issues/%s/comments", ws, slug, args[0]), nil)
		if err == nil {
			var page api.Page[issueComment]
			if json.Unmarshal(commentsData, &page) == nil && len(page.Values) > 0 {
				fmt.Println()
				fmt.Println(ui.StyleTitle.Render(fmt.Sprintf("Comments (%d)", len(page.Values))))
				for _, c := range page.Values {
					fmt.Println()
					fmt.Printf("%s %s\n", ui.StyleBold.Render(c.Author.DisplayName), ui.StyleDim.Render(ui.FormatDate(c.CreatedOn)))
					fmt.Println(c.Content.Raw)
				}
			}
		}
	}

	return nil
}

func runIssueCreate(cmd *cobra.Command, _ []string) error {
	cfg := configFromCtx(cmd)
	client := clientFromCtx(cmd)

	wsFlag, _ := cmd.Flags().GetString("workspace")
	repoFlag, _ := cmd.Flags().GetString("repo")
	title, _ := cmd.Flags().GetString("title")
	body, _ := cmd.Flags().GetString("body")
	kind, _ := cmd.Flags().GetString("kind")
	priority, _ := cmd.Flags().GetString("priority")

	ws, slug, err := config.ResolveRepo(wsFlag, repoFlag, cfg)
	if err != nil {
		return err
	}

	if title == "" {
		fmt.Print("Issue Title: ")
		fmt.Scanln(&title)
		if title == "" {
			return fmt.Errorf("title is required")
		}
	}

	payload := map[string]any{
		"title":    title,
		"kind":     kind,
		"priority": priority,
		"content": map[string]string{
			"raw": body,
		},
	}

	data, err := client.Post(fmt.Sprintf("/repositories/%s/%s/issues", ws, slug), payload)
	if err != nil {
		if apiErr, ok := err.(*api.APIError); ok && apiErr.StatusCode == 404 {
			return issueNotEnabled()
		}
		return err
	}

	var i issue
	if err := json.Unmarshal(data, &i); err != nil {
		return err
	}

	fmt.Printf("%s Created issue %s: %s\n",
		ui.StyleGreen.Render("✓"),
		ui.StyleID.Render(fmt.Sprintf("#%d", i.ID)),
		i.Title,
	)
	fmt.Println(ui.StyleDim.Render(i.Links.HTML.Href))
	return nil
}
