package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"

	"github.com/rafaelfnaves/bitb/internal/api"
	"github.com/rafaelfnaves/bitb/internal/config"
	"github.com/rafaelfnaves/bitb/internal/ui"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage pull requests",
	Long: `Manage pull requests in a Bitbucket repository.

EXAMPLES
  # List open pull requests
  bitb pr list

  # View details of a specific PR
  bitb pr view 42

  # Create a new pull request interactively
  bitb pr create

  # Merge a pull request
  bitb pr merge 42

  # Approve a pull request
  bitb pr approve 42

  # Show the diff of a pull request
  bitb pr diff 42`,
}

func init() {
	rootCmd.AddCommand(prCmd)
	prCmd.AddCommand(prListCmd, prViewCmd, prCreateCmd, prMergeCmd, prApproveCmd, prDiffCmd)

	prListCmd.Flags().StringP("workspace", "w", "", "Workspace slug")
	prListCmd.Flags().StringP("repo", "r", "", "Repository (workspace/slug or slug)")
	prListCmd.Flags().StringP("state", "s", "OPEN", "PR state: OPEN, MERGED, DECLINED")
	prListCmd.Flags().StringP("author", "a", "", "Filter by author display name")
	prListCmd.Flags().IntP("limit", "n", 30, "Maximum number of PRs to show")
	prListCmd.Flags().Bool("json", false, "Output raw JSON")
	prListCmd.Flags().Bool("web", false, "Open in browser")

	prViewCmd.Flags().StringP("workspace", "w", "", "Workspace slug")
	prViewCmd.Flags().StringP("repo", "r", "", "Repository (workspace/slug or slug)")
	prViewCmd.Flags().BoolP("comments", "c", false, "Show comments")
	prViewCmd.Flags().Bool("json", false, "Output raw JSON")
	prViewCmd.Flags().Bool("web", false, "Open in browser")

	prCreateCmd.Flags().StringP("workspace", "w", "", "Workspace slug")
	prCreateCmd.Flags().StringP("repo", "r", "", "Repository (workspace/slug or slug)")
	prCreateCmd.Flags().StringP("title", "t", "", "PR title")
	prCreateCmd.Flags().StringP("body", "b", "", "PR description")
	prCreateCmd.Flags().StringP("source", "s", "", "Source branch (default: current branch)")
	prCreateCmd.Flags().StringP("dest", "d", "", "Destination branch (default: main or master)")
	prCreateCmd.Flags().Bool("draft", false, "Mark as work in progress")
	prCreateCmd.Flags().Bool("web", false, "Open in browser after creation")

	prMergeCmd.Flags().StringP("workspace", "w", "", "Workspace slug")
	prMergeCmd.Flags().StringP("repo", "r", "", "Repository (workspace/slug or slug)")
	prMergeCmd.Flags().StringP("message", "m", "", "Merge commit message")
	prMergeCmd.Flags().String("strategy", "merge_commit", "Merge strategy: merge_commit, squash, fast_forward")
	prMergeCmd.Flags().Bool("close-source", true, "Close source branch after merge")

	prApproveCmd.Flags().StringP("workspace", "w", "", "Workspace slug")
	prApproveCmd.Flags().StringP("repo", "r", "", "Repository (workspace/slug or slug)")

	prDiffCmd.Flags().StringP("workspace", "w", "", "Workspace slug")
	prDiffCmd.Flags().StringP("repo", "r", "", "Repository (workspace/slug or slug)")
}

type prBranch struct {
	Branch struct {
		Name string `json:"name"`
	} `json:"branch"`
}

type prParticipant struct {
	User struct {
		DisplayName string `json:"display_name"`
	} `json:"user"`
	Role     string `json:"role"`
	Approved bool   `json:"approved"`
}

type pullRequest struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	State       string `json:"state"`
	Description string `json:"description"`
	UpdatedOn   string `json:"updated_on"`
	CreatedOn   string `json:"created_on"`
	Source      prBranch `json:"source"`
	Destination prBranch `json:"destination"`
	Author      struct {
		DisplayName string `json:"display_name"`
	} `json:"author"`
	Reviewers    []struct{ DisplayName string `json:"display_name"` } `json:"reviewers"`
	Participants []prParticipant `json:"participants"`
	Links        struct {
		HTML struct{ Href string `json:"href"` } `json:"html"`
	} `json:"links"`
	TaskCount int `json:"task_count"`
}

type prComment struct {
	ID      int    `json:"id"`
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
	CreatedOn string `json:"created_on"`
	Author    struct {
		DisplayName string `json:"display_name"`
	} `json:"author"`
	Inline *struct {
		Path string `json:"path"`
		From *int   `json:"from"`
		To   *int   `json:"to"`
	} `json:"inline"`
}

var prListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests",
	Long: `List pull requests for the current or specified repository.

Defaults to listing OPEN pull requests. Use --state to filter by status.

EXAMPLES
  # List open pull requests (default)
  bitb pr list

  # List merged pull requests
  bitb pr list --state MERGED

  # Filter by author
  bitb pr list --author "Rafael"

  # Output as JSON
  bitb pr list --json

  # Open pull requests list in browser
  bitb pr list --web

  # Target a specific repository
  bitb pr list --repo myworkspace/myrepo`,
	RunE: runPRList,
}

var prViewCmd = &cobra.Command{
	Use:   "view <id>",
	Short: "View a pull request",
	Long: `Display detailed information about a pull request.

Shows title, state, branches, author, reviewers, approvals, and description
(rendered as markdown). Use --comments to also show all comments.

EXAMPLES
  # View PR details
  bitb pr view 42

  # View PR with comments
  bitb pr view 42 --comments

  # Open PR in browser
  bitb pr view 42 --web

  # Output raw JSON
  bitb pr view 42 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runPRView,
}

var prCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a pull request",
	Long: `Create a new pull request for the current branch.

The source branch defaults to the current git branch. The destination
branch is auto-detected from the repository's main branch setting.

If --title is not provided, you will be prompted for it.
If --body is not provided, your $EDITOR will open for the description.

EXAMPLES
  # Create interactively (prompts for title, opens editor for body)
  bitb pr create

  # Create with flags
  bitb pr create --title "Fix login bug" --source feature/fix --dest main

  # Create as draft (work in progress)
  bitb pr create --title "WIP: new feature" --draft

  # Open in browser after creation
  bitb pr create --title "My PR" --web`,
	RunE: runPRCreate,
}

var prMergeCmd = &cobra.Command{
	Use:   "merge <id>",
	Short: "Merge a pull request",
	Long: `Merge a pull request into its destination branch.

The default merge strategy is merge_commit. Use --strategy to change it.
The source branch is closed after merge by default.

MERGE STRATEGIES
  merge_commit   Standard merge commit (default)
  squash         Squash all commits into one
  fast_forward   Fast-forward merge (requires linear history)

EXAMPLES
  # Merge with default strategy
  bitb pr merge 42

  # Squash merge
  bitb pr merge 42 --strategy squash

  # Custom merge commit message
  bitb pr merge 42 --message "Merge feature X into main"

  # Keep source branch after merge
  bitb pr merge 42 --close-source=false`,
	Args: cobra.ExactArgs(1),
	RunE: runPRMerge,
}

var prApproveCmd = &cobra.Command{
	Use:   "approve <id>",
	Short: "Approve a pull request",
	Long: `Approve a pull request as the authenticated user.

EXAMPLE
  bitb pr approve 42`,
	Args: cobra.ExactArgs(1),
	RunE: runPRApprove,
}

var prDiffCmd = &cobra.Command{
	Use:   "diff <id>",
	Short: "Show the diff of a pull request",
	Long: `Display the unified diff of a pull request with syntax highlighting.

Added lines are shown in green, removed lines in red, and file headers in bold.

EXAMPLE
  bitb pr diff 42`,
	Args: cobra.ExactArgs(1),
	RunE: runPRDiff,
}

func runPRList(cmd *cobra.Command, _ []string) error {
	cfg := configFromCtx(cmd)
	client := clientFromCtx(cmd)

	wsFlag, _ := cmd.Flags().GetString("workspace")
	repoFlag, _ := cmd.Flags().GetString("repo")
	state, _ := cmd.Flags().GetString("state")
	author, _ := cmd.Flags().GetString("author")
	limit, _ := cmd.Flags().GetInt("limit")
	jsonOutput, _ := cmd.Flags().GetBool("json")
	web, _ := cmd.Flags().GetBool("web")

	ws, slug, err := config.ResolveRepo(wsFlag, repoFlag, cfg)
	if err != nil {
		return err
	}

	params := url.Values{"state": {strings.ToUpper(state)}}
	if author != "" {
		params.Set("q", fmt.Sprintf(`author.display_name="%s"`, author))
	}

	prs, err := api.Paginate[pullRequest](client, fmt.Sprintf("/repositories/%s/%s/pullrequests", ws, slug), params, limit)
	if err != nil {
		return err
	}

	if web && len(prs) > 0 {
		openURL(fmt.Sprintf("https://bitbucket.org/%s/%s/pull-requests", ws, slug))
		return nil
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(prs, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(prs) == 0 {
		fmt.Printf("No %s pull requests found.\n", strings.ToLower(state))
		return nil
	}

	fmt.Println(ui.StyleTitle.Render(fmt.Sprintf("Pull Requests — %s/%s [%s]", ws, slug, state)))
	fmt.Println()

	t := ui.NewTable("ID", "Title", "Author", "Branches", "Updated")
	for _, pr := range prs {
		branches := fmt.Sprintf("%s → %s", pr.Source.Branch.Name, pr.Destination.Branch.Name)
		t.AddRow(
			ui.StyleID.Render(fmt.Sprintf("#%d", pr.ID)),
			ui.Truncate(pr.Title, 50),
			pr.Author.DisplayName,
			ui.Truncate(branches, 40),
			ui.FormatDate(pr.UpdatedOn),
		)
	}
	fmt.Print(t.Render())
	fmt.Printf("\n%s\n", ui.StyleDim.Render(fmt.Sprintf("%d pull request(s)", len(prs))))
	return nil
}

func runPRView(cmd *cobra.Command, args []string) error {
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

	data, err := client.Get(fmt.Sprintf("/repositories/%s/%s/pullrequests/%s", ws, slug, args[0]), nil)
	if err != nil {
		return err
	}

	var pr pullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return err
	}

	if web {
		openURL(pr.Links.HTML.Href)
		return nil
	}

	if jsonOutput {
		fmt.Println(string(data))
		return nil
	}

	// Header
	fmt.Printf("%s %s\n", ui.StyleID.Render(fmt.Sprintf("#%d", pr.ID)), ui.StyleBold.Render(pr.Title))
	fmt.Printf("%s  %s → %s  %s  %s\n",
		ui.PRState(pr.State),
		ui.StyleCyan.Render(pr.Source.Branch.Name),
		ui.StyleCyan.Render(pr.Destination.Branch.Name),
		ui.StyleDim.Render("by "+pr.Author.DisplayName),
		ui.StyleDim.Render(ui.FormatDate(pr.CreatedOn)),
	)
	fmt.Println()

	// Reviewers
	if len(pr.Reviewers) > 0 {
		names := make([]string, len(pr.Reviewers))
		for i, r := range pr.Reviewers {
			names[i] = r.DisplayName
		}
		fmt.Printf("%s %s\n", ui.StyleDim.Render("Reviewers:"), strings.Join(names, ", "))
	}

	// Approvals
	var approved []string
	for _, p := range pr.Participants {
		if p.Approved {
			approved = append(approved, p.User.DisplayName)
		}
	}
	if len(approved) > 0 {
		fmt.Printf("%s %s\n", ui.StyleGreen.Render("Approved by:"), strings.Join(approved, ", "))
	}

	fmt.Println()

	// Description
	if pr.Description != "" {
		renderer, err := glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(100))
		if err == nil {
			rendered, err := renderer.Render(pr.Description)
			if err == nil {
				fmt.Print(rendered)
			} else {
				fmt.Println(pr.Description)
			}
		} else {
			fmt.Println(pr.Description)
		}
	} else {
		fmt.Println(ui.StyleDim.Render("No description provided."))
	}

	// Link
	fmt.Println()
	fmt.Println(ui.StyleDim.Render(pr.Links.HTML.Href))

	// Comments
	if showComments {
		commentsData, err := client.Get(fmt.Sprintf("/repositories/%s/%s/pullrequests/%s/comments", ws, slug, args[0]), nil)
		if err == nil {
			var page api.Page[prComment]
			if json.Unmarshal(commentsData, &page) == nil && len(page.Values) > 0 {
				fmt.Println()
				fmt.Println(ui.StyleTitle.Render(fmt.Sprintf("Comments (%d)", len(page.Values))))
				fmt.Println()
				for _, c := range page.Values {
					if c.Inline != nil {
						fmt.Printf("%s %s\n", ui.StyleDim.Render("→"), ui.StyleCyan.Render(c.Inline.Path))
					}
					fmt.Printf("%s %s\n", ui.StyleBold.Render(c.Author.DisplayName), ui.StyleDim.Render(ui.FormatDate(c.CreatedOn)))
					fmt.Println(c.Content.Raw)
					fmt.Println(ui.StyleDim.Render(strings.Repeat("─", 60)))
				}
			}
		}
	}

	return nil
}

func runPRCreate(cmd *cobra.Command, _ []string) error {
	cfg := configFromCtx(cmd)
	client := clientFromCtx(cmd)

	wsFlag, _ := cmd.Flags().GetString("workspace")
	repoFlag, _ := cmd.Flags().GetString("repo")
	title, _ := cmd.Flags().GetString("title")
	body, _ := cmd.Flags().GetString("body")
	source, _ := cmd.Flags().GetString("source")
	dest, _ := cmd.Flags().GetString("dest")
	draft, _ := cmd.Flags().GetBool("draft")
	web, _ := cmd.Flags().GetBool("web")

	ws, slug, err := config.ResolveRepo(wsFlag, repoFlag, cfg)
	if err != nil {
		return err
	}

	// Auto-detect source branch
	if source == "" {
		source = config.CurrentBranch()
		if source == "" {
			return fmt.Errorf("could not detect current branch — use --source")
		}
	}

	// Auto-detect destination branch
	if dest == "" {
		dest = detectDefaultBranch(client, ws, slug)
	}

	// Prompt for title if not provided
	if title == "" {
		fmt.Print("PR Title: ")
		fmt.Scanln(&title)
		if title == "" {
			return fmt.Errorf("title is required")
		}
	}

	// Open editor for body if not provided
	if body == "" {
		body = openEditorForBody(title)
	}

	payload := map[string]any{
		"title": title,
		"description": body,
		"source": map[string]any{
			"branch": map[string]string{"name": source},
		},
		"destination": map[string]any{
			"branch": map[string]string{"name": dest},
		},
		"close_source_branch": true,
	}
	if draft {
		payload["draft"] = true
	}

	data, err := client.Post(fmt.Sprintf("/repositories/%s/%s/pullrequests", ws, slug), payload)
	if err != nil {
		return err
	}

	var pr pullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return err
	}

	fmt.Printf("%s Created PR %s: %s\n",
		ui.StyleGreen.Render("✓"),
		ui.StyleID.Render(fmt.Sprintf("#%d", pr.ID)),
		ui.StyleBold.Render(pr.Title),
	)
	fmt.Println(ui.StyleDim.Render(pr.Links.HTML.Href))

	if web {
		openURL(pr.Links.HTML.Href)
	}
	return nil
}

func runPRMerge(cmd *cobra.Command, args []string) error {
	cfg := configFromCtx(cmd)
	client := clientFromCtx(cmd)

	wsFlag, _ := cmd.Flags().GetString("workspace")
	repoFlag, _ := cmd.Flags().GetString("repo")
	message, _ := cmd.Flags().GetString("message")
	strategy, _ := cmd.Flags().GetString("strategy")
	closeSource, _ := cmd.Flags().GetBool("close-source")

	ws, slug, err := config.ResolveRepo(wsFlag, repoFlag, cfg)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"type":                 "pullrequest",
		"merge_strategy":       strategy,
		"close_source_branch":  closeSource,
	}
	if message != "" {
		payload["message"] = message
	}

	data, err := client.Post(fmt.Sprintf("/repositories/%s/%s/pullrequests/%s/merge", ws, slug, args[0]), payload)
	if err != nil {
		return err
	}

	var pr pullRequest
	if err := json.Unmarshal(data, &pr); err != nil {
		return err
	}

	fmt.Printf("%s Merged PR %s: %s\n",
		ui.StyleMagenta.Render("⎇"),
		ui.StyleID.Render(fmt.Sprintf("#%d", pr.ID)),
		pr.Title,
	)
	return nil
}

func runPRApprove(cmd *cobra.Command, args []string) error {
	cfg := configFromCtx(cmd)
	client := clientFromCtx(cmd)

	wsFlag, _ := cmd.Flags().GetString("workspace")
	repoFlag, _ := cmd.Flags().GetString("repo")

	ws, slug, err := config.ResolveRepo(wsFlag, repoFlag, cfg)
	if err != nil {
		return err
	}

	_, err = client.Post(fmt.Sprintf("/repositories/%s/%s/pullrequests/%s/approve", ws, slug, args[0]), map[string]any{})
	if err != nil {
		return err
	}

	fmt.Printf("%s Approved PR #%s\n", ui.StyleGreen.Render("✓"), args[0])
	return nil
}

func runPRDiff(cmd *cobra.Command, args []string) error {
	cfg := configFromCtx(cmd)
	client := clientFromCtx(cmd)

	wsFlag, _ := cmd.Flags().GetString("workspace")
	repoFlag, _ := cmd.Flags().GetString("repo")

	ws, slug, err := config.ResolveRepo(wsFlag, repoFlag, cfg)
	if err != nil {
		return err
	}

	diff, err := client.GetRaw(fmt.Sprintf("/repositories/%s/%s/pullrequests/%s/diff", ws, slug, args[0]))
	if err != nil {
		return err
	}

	// Print diff with basic coloring
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			fmt.Println(ui.StyleBold.Render(line))
		case strings.HasPrefix(line, "+"):
			fmt.Println(ui.StyleGreen.Render(line))
		case strings.HasPrefix(line, "-"):
			fmt.Println(ui.StyleRed.Render(line))
		case strings.HasPrefix(line, "@@"):
			fmt.Println(ui.StyleCyan.Render(line))
		default:
			fmt.Println(line)
		}
	}
	return nil
}

func detectDefaultBranch(client *api.Client, ws, slug string) string {
	data, err := client.Get(fmt.Sprintf("/repositories/%s/%s", ws, slug), nil)
	if err != nil {
		return "main"
	}
	var repo struct {
		Mainbranch struct {
			Name string `json:"name"`
		} `json:"mainbranch"`
	}
	if json.Unmarshal(data, &repo) == nil && repo.Mainbranch.Name != "" {
		return repo.Mainbranch.Name
	}
	return "main"
}

func openEditorForBody(title string) string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		return ""
	}

	tmpFile, err := os.CreateTemp("", "bb-pr-*.md")
	if err != nil {
		return ""
	}
	defer os.Remove(tmpFile.Name())

	template := fmt.Sprintf("# %s\n\n<!-- PR description (lines starting with # will be ignored) -->\n\n## Changes\n\n\n## Testing\n\n", title)
	tmpFile.WriteString(template)
	tmpFile.Close()

	c := exec.Command(editor, tmpFile.Name())
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return ""
	}

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return ""
	}

	// Strip comment lines
	var lines []string
	for _, line := range strings.Split(string(content), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "<!--") {
			lines = append(lines, line)
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func openURL(url string) {
	cmds := []string{"xdg-open", "open", "sensible-browser"}
	for _, c := range cmds {
		if path, err := exec.LookPath(c); err == nil {
			exec.Command(path, url).Start()
			return
		}
	}
	fmt.Println(ui.StyleCyan.Render(url))
}
