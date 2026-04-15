package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jimboylabs/forgejocli/internal/api"
	"github.com/spf13/cobra"
)

var (
	logsOpenBrowser bool
	logsJobID       int
	logsAttempt     int
)

var actionsLogsCmd = &cobra.Command{
	Use:     "logs <run_id>",
	Short:   "Get logs URL for an action run (opens browser or prints URL)",
	Aliases: []string{"log"},
	Long: `View action run logs.

NOTE: Forgejo does not expose action logs via the API. This command provides
the web UI URL to view logs. You can either:
  - Use --open to automatically open the URL in your browser
  - Copy the URL and paste it into your browser manually

The logs URL pattern is:
  {server}/{owner}/{repo}/actions/runs/{run_index}/jobs/{job_index}/attempt/{attempt}

If the run has multiple jobs or attempts, you'll need to navigate to the
specific job in the web UI to see its logs.`,
	Args: cobra.ExactArgs(1),
	Example: `  forgejo actions logs 11 -r builder
  forgejo actions logs 11 -r builder --open
  forgejo actions logs 11 -r builder --job 0 --attempt 1`,
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(nil)
		if repo == "" {
			return fmt.Errorf("repository name is required (use --repo)")
		}

		runIndex := parseInt(args[0], 0)
		if runIndex == 0 {
			return fmt.Errorf("invalid run ID: %s", args[0])
		}

		// List runs to find the matching run index
		result, err := apiClient.ListActionRuns(owner, repo, 1, 50, "")
		if err != nil {
			return fmt.Errorf("listing action runs: %w", err)
		}

		var targetRun *api.ActionRun
		for i := range result.WorkflowRuns {
			if result.WorkflowRuns[i].IndexInRepo == runIndex {
				targetRun = &result.WorkflowRuns[i]
				break
			}
		}

		if targetRun == nil {
			return fmt.Errorf("run #%d not found in %s/%s", runIndex, owner, repo)
		}

		// Build the logs URL with job and attempt
		logsURL := fmt.Sprintf("%s/jobs/%d/attempt/%d",
			targetRun.HTMLURL, logsJobID, logsAttempt)

		if isJSON() {
			output := map[string]interface{}{
				"run_id":      targetRun.IndexInRepo,
				"internal_id": targetRun.ID,
				"status":      targetRun.Status,
				"title":       targetRun.Title,
				"workflow":    targetRun.WorkflowID,
				"commit":      targetRun.CommitSHA,
				"logs_url":    logsURL,
				"note":        "Forgejo does not expose logs via API. Open the URL in browser to view logs.",
			}
			return outputJSON(output)
		}

		// Print info
		t := table.NewWriter()
		t.SetStyle(table.StyleLight)
		t.AppendRow(table.Row{"Run ID", fmt.Sprintf("#%d", targetRun.IndexInRepo)})
		t.AppendRow(table.Row{"Internal ID", targetRun.ID})
		t.AppendRow(table.Row{"Status", targetRun.Status})
		t.AppendRow(table.Row{"Title", targetRun.Title})
		t.AppendRow(table.Row{"Workflow", targetRun.WorkflowID})
		t.AppendRow(table.Row{"Commit", targetRun.CommitSHA})
		t.AppendRow(table.Row{"Job Index", logsJobID})
		t.AppendRow(table.Row{"Attempt", logsAttempt})
		t.AppendRow(table.Row{"Logs URL", logsURL})
		fmt.Println(t.Render())

		fmt.Println()
		fmt.Println("⚠️  Note: Forgejo does not expose action logs via the REST API.")
		fmt.Println("   To view logs, open the URL in your browser.")

		if logsOpenBrowser {
			fmt.Println()
			fmt.Println("🌐 Opening browser...")
			if err := openBrowser(logsURL); err != nil {
				return fmt.Errorf("failed to open browser: %w", err)
			}
		}

		return nil
	},
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default: // linux and others
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

func init() {
	actionsLogsCmd.Flags().BoolVarP(&logsOpenBrowser, "open", "b", false, "open logs URL in browser")
	actionsLogsCmd.Flags().IntVarP(&logsJobID, "job", "j", 0, "job index (default 0 for single-job workflows)")
	actionsLogsCmd.Flags().IntVarP(&logsAttempt, "attempt", "a", 1, "attempt number (default 1)")

	actionsCmd.AddCommand(actionsLogsCmd)
}
