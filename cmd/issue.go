package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	apiclient "github.com/jimboylabs/forgejocli/internal/api"
	"github.com/spf13/cobra"
)

var (
	issueListState    string
	issueListLimit    int
	issueTitle        string
	issueBody         string
	issueAttach       []string
	commentBody       string
	issueCommentPage  int
	issueCommentLimit int
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

		// Use local flag retrieval (avoid global variables)
		viewComments, _ := cmd.Flags().GetBool("comments")
		commentLimit, _ := cmd.Flags().GetInt("comment-limit")

		// If comments requested, fetch both in one call with limit
		if viewComments {
			result, err := apiClient.GetIssueWithComments(owner, repo, index, commentLimit)
			if err != nil {
				return err
			}
			return displayIssueWithComments(result, commentLimit, isJSON())
		}

		// Original behavior: issue only
		issue, err := apiClient.GetIssue(owner, repo, index)
		if err != nil {
			return err
		}
		return displayIssue(issue, isJSON())
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

// displayIssue shows issue without comments (original behavior)
func displayIssue(issue *apiclient.Issue, jsonOutput bool) error {
	if jsonOutput {
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
}

// displayIssueWithComments shows issue with comments
// Refactored to reuse displayIssue (DRY principle)
func displayIssueWithComments(result *apiclient.IssueWithComments, commentLimit int, jsonOutput bool) error {
	if jsonOutput {
		return outputJSON(result)
	}

	// Reuse base display function (DRY principle)
	if err := displayIssue(result.Issue, false); err != nil {
		return err
	}

	// Comments section - show count and truncation notice
	commentCount := len(result.Comments)
	fmt.Printf("\n─%s─\n", strings.Repeat("─", 58))
	if commentLimit > 0 && commentCount == commentLimit {
		fmt.Printf("Comments (showing %d of ? - use -n 0 to show all)\n", commentCount)
	} else {
		fmt.Printf("Comments (%d)\n", commentCount)
	}
	fmt.Printf("─%s─\n", strings.Repeat("─", 58))

	if len(result.Comments) == 0 {
		fmt.Println("No comments.")
		return nil
	}

	// Show comments with their IDs (critical for delete workflow)
	for i, comment := range result.Comments {
		author := ""
		if comment.User != nil {
			author = comment.User.Login
		}

		// Show both sequence number AND comment ID (needed for delete)
		fmt.Printf("\n#%d (ID: %d) by %s on %s\n",
			i+1,
			comment.ID,
			author,
			comment.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println(strings.Repeat("─", 60))
		fmt.Println(comment.Body)
	}

	return nil
}

var issueCommentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Manage issue comments",
}

var issueCommentListCmd = &cobra.Command{
	Use:   "list <issue-number>",
	Short: "List comments on an issue",
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

		// Use pagination parameters (0 = use defaults from API: page 1, limit 30)
		page := issueCommentPage
		if page == 0 {
			page = 1
		}
		limit := issueCommentLimit
		if limit == 0 {
			limit = 30 // Default page size
		}

		comments, err := apiClient.GetIssueComments(owner, repo, index, page, limit)
		if err != nil {
			return err
		}

		if isJSON() {
			return outputJSON(comments)
		}

		if len(comments) == 0 {
			fmt.Println("No comments found.")
			return nil
		}

		for _, comment := range comments {
			author := ""
			if comment.User != nil {
				author = comment.User.Login
			}
			// Emphasize Comment ID for delete workflow
			fmt.Printf("[Comment ID: %d] by %s on %s:\n%s\n\n",
				comment.ID,
				author,
				comment.CreatedAt.Format("2006-01-02 15:04"),
				comment.Body)
		}

		// Show pagination hint if results might be truncated
		if len(comments) == limit {
			fmt.Printf("(showing %d comments, use --page %d to see more)\n", limit, page+1)
		}

		return nil
	},
}

var issueCommentCreateCmd = &cobra.Command{
	Use:   "create <issue-number>",
	Short: "Add a comment to an issue",
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

		// Get body from flag or stdin
		body := commentBody
		if body == "" || body == "-" {
			// Read from stdin
			stat, _ := os.Stdin.Stat()
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				// Stdin has data piped
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("reading from stdin: %w", err)
				}
				body = string(data)
			} else {
				return fmt.Errorf("comment body is required (use --body or pipe via stdin)")
			}
		}

		if body == "" {
			return fmt.Errorf("comment body cannot be empty")
		}

		comment, err := apiClient.CreateComment(owner, repo, index, body)
		if err != nil {
			return err
		}

		if isJSON() {
			return outputJSON(comment)
		}

		fmt.Printf("Comment added to issue #%d\n", index)
		fmt.Printf("URL: %s\n", comment.HTMLURL)
		return nil
	},
}

var issueCommentDeleteCmd = &cobra.Command{
	Use:   "delete <comment-id>",
	Short: "Delete a comment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(nil)
		if repo == "" {
			return fmt.Errorf("repository name is required (use --repo)")
		}

		commentID := parseInt(args[0], 0)
		if commentID <= 0 {
			return fmt.Errorf("invalid comment ID: %s (must be a positive integer)", args[0])
		}

		if err := apiClient.DeleteComment(owner, repo, commentID); err != nil {
			return err
		}

		fmt.Printf("Deleted comment #%d\n", commentID)
		return nil
	},
}

func init() {
	issueListCmd.Flags().StringVarP(&issueListState, "state", "s", "open", "filter by state (open/closed/all)")
	issueListCmd.Flags().IntVarP(&issueListLimit, "limit", "n", 20, "number of issues to list")

	issueCreateCmd.Flags().StringVarP(&issueTitle, "title", "t", "", "issue title")
	issueCreateCmd.Flags().StringVarP(&issueBody, "body", "b", "", "issue body")
	issueCreateCmd.Flags().StringArrayVar(&issueAttach, "attach", nil, "attach image file (can be used multiple times)")

	// NEW: Add comments flags to view command
	// Note: Using direct flag registration (not global variables)
	issueViewCmd.Flags().BoolP("comments", "c", false, "include comments")
	issueViewCmd.Flags().IntP("comment-limit", "n", 30, "max comments to show (0 = all)")

	// Comment command flags
	// Note: Supports --body "text", --body - (stdin), or pipe via stdin
	issueCommentCreateCmd.Flags().StringVarP(&commentBody, "body", "b", "", "comment body (use '-' for stdin)")

	// NEW: Pagination flags for comment list
	issueCommentListCmd.Flags().IntVarP(&issueCommentLimit, "limit", "n", 30, "number of comments per page")
	issueCommentListCmd.Flags().IntVarP(&issueCommentPage, "page", "p", 1, "page number")

	// Add subcommands
	issueCommentCmd.AddCommand(issueCommentListCmd)
	issueCommentCmd.AddCommand(issueCommentCreateCmd)
	issueCommentCmd.AddCommand(issueCommentDeleteCmd)
	issueCmd.AddCommand(issueCommentCmd)

	issueCmd.AddCommand(issueListCmd)
	issueCmd.AddCommand(issueViewCmd)
	issueCmd.AddCommand(issueCreateCmd)
	issueCmd.AddCommand(issueCloseCmd)
	issueCmd.AddCommand(issueReopenCmd)
	rootCmd.AddCommand(issueCmd)
}
