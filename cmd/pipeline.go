package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"github.com/spf13/cobra"

	"github.com/rafaelfnaves/bitb/internal/api"
	"github.com/rafaelfnaves/bitb/internal/config"
	"github.com/rafaelfnaves/bitb/internal/ui"
)

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "View pipelines",
}

func init() {
	rootCmd.AddCommand(pipelineCmd)
	pipelineCmd.AddCommand(pipelineListCmd, pipelineViewCmd)

	pipelineListCmd.Flags().StringP("workspace", "w", "", "Workspace slug")
	pipelineListCmd.Flags().StringP("repo", "r", "", "Repository (workspace/slug or slug)")
	pipelineListCmd.Flags().StringP("branch", "b", "", "Filter by branch name")
	pipelineListCmd.Flags().IntP("limit", "n", 20, "Maximum number of pipelines to show")
	pipelineListCmd.Flags().Bool("json", false, "Output raw JSON")

	pipelineViewCmd.Flags().StringP("workspace", "w", "", "Workspace slug")
	pipelineViewCmd.Flags().StringP("repo", "r", "", "Repository (workspace/slug or slug)")
	pipelineViewCmd.Flags().Bool("json", false, "Output raw JSON")
	pipelineViewCmd.Flags().Bool("web", false, "Open in browser")
}

type pipelineTarget struct {
	RefName string `json:"ref_name"`
	RefType string `json:"ref_type"`
	Commit  *struct {
		Hash string `json:"hash"`
	} `json:"commit"`
}

type pipelineDuration struct {
	Seconds int `json:"seconds"`
}

type pipeline struct {
	UUID        string           `json:"uuid"`
	BuildNumber int              `json:"build_number"`
	State       struct {
		Name   string `json:"name"`
		Result *struct {
			Name string `json:"name"`
		} `json:"result"`
		Stage *struct {
			Name string `json:"name"`
		} `json:"stage"`
	} `json:"state"`
	Target      pipelineTarget   `json:"target"`
	CreatedOn   string           `json:"created_on"`
	CompletedOn string           `json:"completed_on"`
	DurationInSeconds int        `json:"duration_in_seconds"`
	Creator     struct {
		DisplayName string `json:"display_name"`
	} `json:"creator"`
	Links struct {
		HTML struct{ Href string `json:"href"` } `json:"html"`
	} `json:"links"`
}

type pipelineStep struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	State     struct {
		Name   string `json:"name"`
		Result *struct {
			Name string `json:"name"`
		} `json:"result"`
	} `json:"state"`
	DurationInSeconds int    `json:"duration_in_seconds"`
	CompletedOn       string `json:"completed_on"`
}

var pipelineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent pipelines",
	RunE:  runPipelineList,
}

var pipelineViewCmd = &cobra.Command{
	Use:   "view <build-number>",
	Short: "View pipeline details and steps",
	Args:  cobra.ExactArgs(1),
	RunE:  runPipelineView,
}

func runPipelineList(cmd *cobra.Command, _ []string) error {
	cfg := configFromCtx(cmd)
	client := clientFromCtx(cmd)

	wsFlag, _ := cmd.Flags().GetString("workspace")
	repoFlag, _ := cmd.Flags().GetString("repo")
	branch, _ := cmd.Flags().GetString("branch")
	limit, _ := cmd.Flags().GetInt("limit")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	ws, slug, err := config.ResolveRepo(wsFlag, repoFlag, cfg)
	if err != nil {
		return err
	}

	params := url.Values{"sort": {"-created_on"}}
	if branch != "" {
		params.Set("target.branch", branch)
	}

	pipelines, err := api.Paginate[pipeline](client, fmt.Sprintf("/repositories/%s/%s/pipelines/", ws, slug), params, limit)
	if err != nil {
		return err
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(pipelines, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(pipelines) == 0 {
		fmt.Println("No pipelines found.")
		return nil
	}

	fmt.Println(ui.StyleTitle.Render(fmt.Sprintf("Pipelines — %s/%s", ws, slug)))
	fmt.Println()

	t := ui.NewTable("Build", "Status", "Branch", "Duration", "Created", "By")
	for _, p := range pipelines {
		stateName := p.State.Name
		resultName := ""
		if p.State.Result != nil {
			resultName = p.State.Result.Name
		}
		status := ui.PipelineState(stateName, resultName)

		branch := p.Target.RefName
		if branch == "" {
			branch = "—"
		}

		duration := "—"
		if p.DurationInSeconds > 0 {
			duration = formatDuration(p.DurationInSeconds)
		}

		t.AddRow(
			ui.StyleID.Render(fmt.Sprintf("#%d", p.BuildNumber)),
			status,
			ui.Truncate(branch, 30),
			duration,
			ui.FormatDate(p.CreatedOn),
			p.Creator.DisplayName,
		)
	}
	fmt.Print(t.Render())
	fmt.Printf("\n%s\n", ui.StyleDim.Render(fmt.Sprintf("%d pipeline(s)", len(pipelines))))
	return nil
}

func runPipelineView(cmd *cobra.Command, args []string) error {
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

	// Find pipeline by build number
	buildNum := args[0]
	params := url.Values{
		"sort":    {"-created_on"},
		"pagelen": {"100"},
	}
	pipelines, err := api.Paginate[pipeline](client, fmt.Sprintf("/repositories/%s/%s/pipelines/", ws, slug), params, 200)
	if err != nil {
		return err
	}

	var found *pipeline
	for i, p := range pipelines {
		if fmt.Sprintf("%d", p.BuildNumber) == buildNum {
			found = &pipelines[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("pipeline #%s not found", buildNum)
	}

	if web {
		openURL(found.Links.HTML.Href)
		return nil
	}

	if jsonOutput {
		data, _ := json.MarshalIndent(found, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Header
	stateName := found.State.Name
	resultName := ""
	if found.State.Result != nil {
		resultName = found.State.Result.Name
	}

	fmt.Printf("%s %s  %s  %s\n\n",
		ui.StyleTitle.Render(fmt.Sprintf("Pipeline #%d", found.BuildNumber)),
		ui.PipelineState(stateName, resultName),
		ui.StyleCyan.Render(found.Target.RefName),
		ui.StyleDim.Render(ui.FormatDate(found.CreatedOn)),
	)

	if found.DurationInSeconds > 0 {
		fmt.Printf("%s %s\n\n", ui.StyleDim.Render("Duration:"), formatDuration(found.DurationInSeconds))
	}

	// Fetch steps
	stepsData, err := client.Get(fmt.Sprintf("/repositories/%s/%s/pipelines/%s/steps/",
		ws, slug, url.PathEscape(found.UUID)), nil)
	if err == nil {
		var stepsPage api.Page[pipelineStep]
		if json.Unmarshal(stepsData, &stepsPage) == nil && len(stepsPage.Values) > 0 {
			fmt.Println(ui.StyleBold.Render("Steps:"))
			fmt.Println()
			t := ui.NewTable("Step", "Status", "Duration")
			for _, step := range stepsPage.Values {
				stepStateName := step.State.Name
				stepResultName := ""
				if step.State.Result != nil {
					stepResultName = step.State.Result.Name
				}
				dur := "—"
				if step.DurationInSeconds > 0 {
					dur = formatDuration(step.DurationInSeconds)
				}
				t.AddRow(
					ui.Truncate(step.Name, 50),
					ui.PipelineState(stepStateName, stepResultName),
					dur,
				)
			}
			fmt.Print(t.Render())
		}
	}

	fmt.Println()
	fmt.Println(ui.StyleDim.Render(found.Links.HTML.Href))
	return nil
}

func formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	mins := seconds / 60
	secs := seconds % 60
	if mins < 60 {
		return fmt.Sprintf("%dm%ds", mins, secs)
	}
	hours := mins / 60
	mins = mins % 60
	return fmt.Sprintf("%dh%dm", hours, mins)
}

