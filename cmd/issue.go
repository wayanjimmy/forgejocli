package cmd

import (
	"fmt"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	apiclient "github.com/jimboylabs/forgejocli/internal/api"
	"github.com/spf13/cobra"
)

var (
	issueListState string
	issueListLimit int
	issueTitle     string
	issueBody      string
)

var issueCmd = &cobra.Command{
	Use:     "issue",
	Short:   "Manage issues",
	Aliases: []string{"i"},
}

var issueListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List issues",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(args)
		if repo == "" {
			return fmt.Errorf("repository name is required (use --repo)")
		}

		issues, err := apiClient.ListIssues(owner, repo, 1, issueListLimit, issueListState)
		if err != nil {
			return err
		}

		if len(issues) == 0 {
			if isJSON() {
				return outputJSON([]interface{}{})
			}
			fmt.Println("No issues found.")
			return nil
		}

		if isJSON() {
			return outputJSON(issues)
		}

		t := table.NewWriter()
		t.SetStyle(table.StyleLight)
		t.AppendHeader(table.Row{"NUMBER", "STATE", "TITLE", "AUTHOR", "CREATED"})

		for _, issue := range issues {
			title := issue.Title
			if len(title) > 60 {
				title = title[:57] + "..."
			}
			author := ""
			if issue.User != nil {
				author = issue.User.Login
			}
			t.AppendRow(table.Row{
				issue.Number,
				issue.State,
				title,
				author,
				issue.CreatedAt.Format("2006-01-02"),
			})
		}

		fmt.Println(t.Render())
		fmt.Printf("\nShowing %d issue(s) in %s/%s (state: %s)\n", len(issues), owner, repo, issueListState)
		return nil
	},
}

var issueViewCmd = &cobra.Command{
	Use:   "view <number>",
	Short: "View an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(nil)
		if repo == "" {
			return fmt.Errorf("repository name is required (use --repo)")
		}

		index := parseInt(args[0], 0)
		if index == 0 {
			return fmt.Errorf("invalid issue number: %s", args[0])
		}

		issue, err := apiClient.GetIssue(owner, repo, index)
		if err != nil {
			return err
		}

		if isJSON() {
			return outputJSON(issue)
		}

		author := ""
		if issue.User != nil {
			author = issue.User.Login
		}

		fmt.Printf("number:     #%d\n", issue.Number)
		fmt.Printf("state:      %s\n", issue.State)
		fmt.Printf("title:      %s\n", issue.Title)
		fmt.Printf("author:     %s\n", author)
		fmt.Printf("url:        %s\n", issue.HTMLURL)
		fmt.Printf("created_at: %s\n", issue.CreatedAt.Format("2006-01-02 15:04:05"))

		if len(issue.Labels) > 0 {
			var labels []string
			for _, l := range issue.Labels {
				labels = append(labels, l.Name)
			}
			fmt.Printf("labels:     %s\n", strings.Join(labels, ", "))
		}

		if issue.Body != "" {
			fmt.Println()
			fmt.Println(issue.Body)
		}

		return nil
	},
}

var issueCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue",
	Example: `  forgejo issue create --repo myproject --title "Bug fix" --body "Description"
  forgejo issue create -r myproject -t "Quick bug" -b "Something broke"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(nil)
		if repo == "" {
			return fmt.Errorf("repository name is required (use --repo)")
		}
		if issueTitle == "" {
			return fmt.Errorf("title is required (use --title)")
		}

		issue, err := apiClient.CreateIssue(owner, repo, apiclient.CreateIssueOption{
			Title: issueTitle,
			Body:  issueBody,
		})
		if err != nil {
			return err
		}

		if isJSON() {
			return outputJSON(issue)
		}

		fmt.Printf("created: issue #%d\n", issue.Number)
		fmt.Printf("title:   %s\n", issue.Title)
		fmt.Printf("url:     %s\n", issue.HTMLURL)
		return nil
	},
}

var issueCloseCmd = &cobra.Command{
	Use:   "close <number>",
	Short: "Close an issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(nil)
		if repo == "" {
			return fmt.Errorf("repository name is required (use --repo)")
		}

		index := parseInt(args[0], 0)
		if index == 0 {
			return fmt.Errorf("invalid issue number: %s", args[0])
		}

		issue, err := apiClient.CloseIssue(owner, repo, index)
		if err != nil {
			return err
		}

		if isJSON() {
			return outputJSON(issue)
		}

		fmt.Printf("closed: issue #%d\n", issue.Number)
		fmt.Printf("title:  %s\n", issue.Title)
		return nil
	},
}

var issueReopenCmd = &cobra.Command{
	Use:   "reopen <number>",
	Short: "Reopen a closed issue",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(nil)
		if repo == "" {
			return fmt.Errorf("repository name is required (use --repo)")
		}

		index := parseInt(args[0], 0)
		if index == 0 {
			return fmt.Errorf("invalid issue number: %s", args[0])
		}

		issue, err := apiClient.ReopenIssue(owner, repo, index)
		if err != nil {
			return err
		}

		if isJSON() {
			return outputJSON(issue)
		}

		fmt.Printf("reopened: issue #%d\n", issue.Number)
		fmt.Printf("title:    %s\n", issue.Title)
		return nil
	},
}

func init() {
	issueListCmd.Flags().StringVarP(&issueListState, "state", "s", "open", "filter by state (open/closed/all)")
	issueListCmd.Flags().IntVarP(&issueListLimit, "limit", "n", 20, "number of issues to list")

	issueCreateCmd.Flags().StringVarP(&issueTitle, "title", "t", "", "issue title")
	issueCreateCmd.Flags().StringVarP(&issueBody, "body", "b", "", "issue body")

	issueCmd.AddCommand(issueListCmd)
	issueCmd.AddCommand(issueViewCmd)
	issueCmd.AddCommand(issueCreateCmd)
	issueCmd.AddCommand(issueCloseCmd)
	issueCmd.AddCommand(issueReopenCmd)
	rootCmd.AddCommand(issueCmd)
}
