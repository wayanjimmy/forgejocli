package cmd

import (
	"fmt"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	apiclient "github.com/jimboylabs/forgejocli/internal/api"
	"github.com/spf13/cobra"
)

var (
	actionListStatus string
	actionListLimit  int
)

var actionsCmd = &cobra.Command{
	Use:     "actions",
	Short:   "Manage Forgejo Actions (CI/CD)",
	Aliases: []string{"act", "ci"},
}

var actionsListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List action runs for a repository",
	Aliases: []string{"ls", "runs"},
	Example: "  forgejo actions list -r builder\n  forgejo actions list -r builder -s failure\n  forgejo actions list -r builder -s success -n 5",
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(args)
		if repo == "" {
			return fmt.Errorf("repository name is required (use --repo)")
		}

		result, err := apiClient.ListActionRuns(owner, repo, 1, actionListLimit, actionListStatus)
		if err != nil {
			return err
		}

		if len(result.WorkflowRuns) == 0 {
			if isJSON() {
				return outputJSON([]interface{}{})
			}
			fmt.Printf("No action runs found in %s/%s.\n", owner, repo)
			return nil
		}

		if isJSON() {
			return outputJSON(result.WorkflowRuns)
		}

		t := table.NewWriter()
		t.SetStyle(table.StyleLight)
		t.AppendHeader(table.Row{"RUN_ID", "STATUS", "TITLE", "EVENT", "BRANCH", "WORKFLOW", "TRIGGERED_BY", "DURATION", "CREATED"})

		for _, run := range result.WorkflowRuns {
			status := run.Status
			duration := formatDuration(run.Duration)
			triggeredBy := ""
			if run.TriggerUser != nil {
				triggeredBy = run.TriggerUser.Login
			}
			title := run.Title
			if len(title) > 40 {
				title = title[:37] + "..."
			}

			t.AppendRow(table.Row{
				run.IndexInRepo,
				status,
				title,
				run.Event,
				run.PrettyRef,
				run.WorkflowID,
				triggeredBy,
				duration,
				run.Created.Format("2006-01-02 15:04"),
			})
		}

		fmt.Println(t.Render())
		fmt.Printf("\nShowing %d run(s) in %s/%s (total: %d, filter: %s)\n",
			len(result.WorkflowRuns), owner, repo, result.TotalCount, actionListStatus)
		return nil
	},
}

var actionsViewCmd = &cobra.Command{
	Use:   "view <run_id>",
	Short: "View details of an action run",
	Args:  cobra.MaximumNArgs(1),
	Example: "  forgejo actions view -r builder 12\n  forgejo actions view -r builder 12 -O json",
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(nil)
		if repo == "" {
			return fmt.Errorf("repository name is required (use --repo)")
		}

		if len(args) == 0 {
			return fmt.Errorf("run ID is required (pass as argument)")
		}

		runID := parseInt(args[0], 0)
		if runID == 0 {
			return fmt.Errorf("invalid run ID: %s", args[0])
		}

		// The API uses the internal run ID (id field), not index_in_repo.
		// We need to list runs to find the internal ID matching the index.
		// Alternatively, try the index directly — Forgejo may accept it.
		run, err := apiClient.GetActionRun(owner, repo, runID)
		if err != nil {
			// If direct lookup fails, try searching in recent runs
			result, listErr := apiClient.ListActionRuns(owner, repo, 1, 50, "")
			if listErr != nil {
				return fmt.Errorf("run not found and fallback list failed: %w", err)
			}
			var found *apiclient.ActionRun
			for i := range result.WorkflowRuns {
				if result.WorkflowRuns[i].IndexInRepo == runID {
					found = &result.WorkflowRuns[i]
					break
				}
			}
			if found == nil {
				return fmt.Errorf("run #%d not found in %s/%s", runID, owner, repo)
			}
			run = found
		}

		if isJSON() {
			return outputJSON(run)
		}

		triggeredBy := ""
		if run.TriggerUser != nil {
			triggeredBy = run.TriggerUser.Login
		}
		duration := formatDuration(run.Duration)
		commitShort := run.CommitSHA
		if len(commitShort) > 12 {
			commitShort = commitShort[:12]
		}

		fmt.Printf("run_id:        #%d\n", run.IndexInRepo)
		fmt.Printf("internal_id:   %d\n", run.ID)
		fmt.Printf("status:        %s\n", run.Status)
		fmt.Printf("title:         %s\n", run.Title)
		fmt.Printf("event:         %s\n", run.Event)
		fmt.Printf("branch:        %s\n", run.PrettyRef)
		fmt.Printf("workflow:      %s\n", run.WorkflowID)
		fmt.Printf("commit:        %s\n", commitShort)
		fmt.Printf("triggered_by:  %s\n", triggeredBy)
		fmt.Printf("duration:      %s\n", duration)
		fmt.Printf("url:           %s\n", run.HTMLURL)
		fmt.Printf("created:       %s\n", run.Created.Format("2006-01-02 15:04:05"))

		if !run.Started.IsZero() {
			fmt.Printf("started:       %s\n", run.Started.Format("2006-01-02 15:04:05"))
		}
		if !run.Stopped.IsZero() {
			fmt.Printf("stopped:       %s\n", run.Stopped.Format("2006-01-02 15:04:05"))
		}

		return nil
	},
}

func formatDuration(nanos int64) string {
	if nanos <= 0 {
		return "-"
	}
	d := time.Duration(nanos)
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm%.0fs", d.Minutes(), d.Seconds()-float64(int64(d.Minutes())*60))
	}
	return fmt.Sprintf("%.0fh%.0fm", d.Hours(), d.Minutes()-float64(int64(d.Hours())*60))
}

func init() {
	actionsListCmd.Flags().StringVarP(&actionListStatus, "status", "s", "", "filter by status (success/failure/running/waiting/cancelled)")
	actionsListCmd.Flags().IntVarP(&actionListLimit, "limit", "n", 20, "number of runs to list")

	actionsCmd.AddCommand(actionsListCmd)
	actionsCmd.AddCommand(actionsViewCmd)
	rootCmd.AddCommand(actionsCmd)
}
