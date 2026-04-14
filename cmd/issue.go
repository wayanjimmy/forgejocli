package cmd

import (
	"fmt"
	"os"
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
	issueAttach    []string
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
	Short: "Create a new issue with optional image attachments",
	Example: `  forgejo issue create -r myproject -t "Bug fix" -b "Description"
  forgejo issue create -r myproject -t "Screenshot bug" --attach ./screenshot.png
  forgejo issue create -r myproject -t "Multi image" --attach ./img1.png --attach ./img2.png`,
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(nil)
		if repo == "" {
			return fmt.Errorf("repository name is required (use --repo)")
		}
		if issueTitle == "" {
			return fmt.Errorf("title is required (use --title)")
		}

		// Validate attachment files exist before creating the issue
		for _, f := range issueAttach {
			if _, err := os.Stat(f); os.IsNotExist(err) {
				return fmt.Errorf("attachment file not found: %s", f)
			}
		}

		// Build body with image references (placeholders, replaced after upload)
		body := issueBody
		var imagePlaceholders []string
		for _, f := range issueAttach {
			placeholder := fmt.Sprintf("![%s](upload-placeholder:%s)", f, f)
			imagePlaceholders = append(imagePlaceholders, placeholder)
		}
		if len(imagePlaceholders) > 0 {
			if body != "" {
				body += "\n\n"
			}
			body += strings.Join(imagePlaceholders, "\n")
		}

		issue, err := apiClient.CreateIssue(owner, repo, apiclient.CreateIssueOption{
			Title: issueTitle,
			Body:  body,
		})
		if err != nil {
			return err
		}

		// Upload attachments and update body with real URLs
		if len(issueAttach) > 0 {
			var imageRefs []string
			for _, f := range issueAttach {
				att, err := apiClient.UploadIssueAsset(owner, repo, issue.Number, f)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to upload %s: %v\n", f, err)
					imageRefs = append(imageRefs, fmt.Sprintf("![%s](upload-failed)", f))
					continue
				}
				imageRefs = append(imageRefs, fmt.Sprintf("![%s](%s)", f, att.BrowserURL))
			}

			// Replace placeholders with real URLs
			newBody := issue.Body
			for i, placeholder := range imagePlaceholders {
				if i < len(imageRefs) {
					newBody = strings.Replace(newBody, placeholder, imageRefs[i], 1)
				}
			}

			// Update issue with final body
			updated, err := apiClient.EditIssue(owner, repo, issue.Number, apiclient.EditIssueOption{
				Body: &newBody,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to update issue body: %v\n", err)
			} else {
				issue = updated
			}
		}

		if isJSON() {
			return outputJSON(issue)
		}

		fmt.Printf("created: issue #%d\n", issue.Number)
		fmt.Printf("title:   %s\n", issue.Title)
		if len(issueAttach) > 0 {
			fmt.Printf("attachments: %d file(s) uploaded\n", len(issueAttach))
		}
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
	issueCreateCmd.Flags().StringArrayVar(&issueAttach, "attach", nil, "attach image file (can be used multiple times)")

	issueCmd.AddCommand(issueListCmd)
	issueCmd.AddCommand(issueViewCmd)
	issueCmd.AddCommand(issueCreateCmd)
	issueCmd.AddCommand(issueCloseCmd)
	issueCmd.AddCommand(issueReopenCmd)
	rootCmd.AddCommand(issueCmd)
}
