package cmd

import (
	"fmt"

	"github.com/jedib0t/go-pretty/v6/table"
	apiclient "github.com/jimboylabs/forgejocli/internal/api"
	"github.com/spf13/cobra"
)

var (
	prListState string
	prListLimit int
	prTitle     string
	prBody      string
	prHead      string
	prBase      string
	prMergeMsg  string
)

var prCmd = &cobra.Command{
	Use:     "pr",
	Short:   "Manage pull requests",
	Aliases: []string{"pull"},
}

var prListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List pull requests",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(args)
		if repo == "" {
			return fmt.Errorf("repository name is required (use --repo)")
		}

		prs, err := apiClient.ListPRs(owner, repo, 1, prListLimit, prListState)
		if err != nil {
			return err
		}

		if len(prs) == 0 {
			if isJSON() {
				return outputJSON([]interface{}{})
			}
			fmt.Println("No pull requests found.")
			return nil
		}

		if isJSON() {
			return outputJSON(prs)
		}

		t := table.NewWriter()
		t.SetStyle(table.StyleLight)
		t.AppendHeader(table.Row{"NUMBER", "STATE", "TITLE", "BRANCH", "AUTHOR", "CREATED"})

		for _, pr := range prs {
			state := pr.State
			if pr.Merged {
				state = "merged"
			}
			title := pr.Title
			if len(title) > 50 {
				title = title[:47] + "..."
			}
			author := ""
			if pr.User != nil {
				author = pr.User.Login
			}
			branch := pr.Head.Ref + " -> " + pr.Base.Ref
			t.AppendRow(table.Row{
				pr.Number,
				state,
				title,
				branch,
				author,
				pr.CreatedAt.Format("2006-01-02"),
			})
		}

		fmt.Println(t.Render())
		fmt.Printf("\nShowing %d PR(s) in %s/%s (state: %s)\n", len(prs), owner, repo, prListState)
		return nil
	},
}

var prViewCmd = &cobra.Command{
	Use:   "view <number>",
	Short: "View a pull request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(nil)
		if repo == "" {
			return fmt.Errorf("repository name is required (use --repo)")
		}

		index := parseInt(args[0], 0)
		if index == 0 {
			return fmt.Errorf("invalid PR number: %s", args[0])
		}

		pr, err := apiClient.GetPR(owner, repo, index)
		if err != nil {
			return err
		}

		if isJSON() {
			return outputJSON(pr)
		}

		state := pr.State
		if pr.Merged {
			state = "merged"
		}

		author := ""
		if pr.User != nil {
			author = pr.User.Login
		}

		fmt.Printf("number:     #%d\n", pr.Number)
		fmt.Printf("state:      %s\n", state)
		fmt.Printf("title:      %s\n", pr.Title)
		fmt.Printf("author:     %s\n", author)
		fmt.Printf("head:       %s\n", pr.Head.Ref)
		fmt.Printf("base:       %s\n", pr.Base.Ref)
		fmt.Printf("mergeable:  %v\n", pr.Mergeable)
		fmt.Printf("url:        %s\n", pr.HTMLURL)
		fmt.Printf("created_at: %s\n", pr.CreatedAt.Format("2006-01-02 15:04:05"))

		if pr.Body != "" {
			fmt.Println()
			fmt.Println(pr.Body)
		}

		return nil
	},
}

var prCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a pull request",
	Example: `  forgejo pr create --repo myproject --title "New feature" --body "Adds X" --head feature-branch --base main`,
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(nil)
		if repo == "" {
			return fmt.Errorf("repository name is required (use --repo)")
		}
		if prTitle == "" {
			return fmt.Errorf("title is required (use --title)")
		}
		if prHead == "" {
			return fmt.Errorf("head branch is required (use --head)")
		}
		if prBase == "" {
			prBase = "main"
		}

		pr, err := apiClient.CreatePR(owner, repo, apiclient.CreatePROption{
			Title: prTitle,
			Body:  prBody,
			Head:  prHead,
			Base:  prBase,
		})
		if err != nil {
			return err
		}

		if isJSON() {
			return outputJSON(pr)
		}

		fmt.Printf("created: PR #%d\n", pr.Number)
		fmt.Printf("title:   %s\n", pr.Title)
		fmt.Printf("head:    %s\n", prHead)
		fmt.Printf("base:    %s\n", prBase)
		fmt.Printf("url:     %s\n", pr.HTMLURL)
		return nil
	},
}

var prMergeCmd = &cobra.Command{
	Use:   "merge <number>",
	Short: "Merge a pull request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(nil)
		if repo == "" {
			return fmt.Errorf("repository name is required (use --repo)")
		}

		index := parseInt(args[0], 0)
		if index == 0 {
			return fmt.Errorf("invalid PR number: %s", args[0])
		}

		if err := apiClient.MergePR(owner, repo, index, prMergeMsg); err != nil {
			return err
		}

		fmt.Printf("merged: PR #%d\n", index)
		return nil
	},
}

var prCloseCmd = &cobra.Command{
	Use:   "close <number>",
	Short: "Close a pull request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(nil)
		if repo == "" {
			return fmt.Errorf("repository name is required (use --repo)")
		}

		index := parseInt(args[0], 0)
		if index == 0 {
			return fmt.Errorf("invalid PR number: %s", args[0])
		}

		pr, err := apiClient.ClosePR(owner, repo, index)
		if err != nil {
			return err
		}

		if isJSON() {
			return outputJSON(pr)
		}

		fmt.Printf("closed: PR #%d\n", pr.Number)
		fmt.Printf("title:  %s\n", pr.Title)
		return nil
	},
}

var prReopenCmd = &cobra.Command{
	Use:   "reopen <number>",
	Short: "Reopen a closed pull request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(nil)
		if repo == "" {
			return fmt.Errorf("repository name is required (use --repo)")
		}

		index := parseInt(args[0], 0)
		if index == 0 {
			return fmt.Errorf("invalid PR number: %s", args[0])
		}

		pr, err := apiClient.ReopenPR(owner, repo, index)
		if err != nil {
			return err
		}

		if isJSON() {
			return outputJSON(pr)
		}

		fmt.Printf("reopened: PR #%d\n", pr.Number)
		fmt.Printf("title:    %s\n", pr.Title)
		return nil
	},
}

func init() {
	prListCmd.Flags().StringVarP(&prListState, "state", "s", "open", "filter by state (open/closed/all)")
	prListCmd.Flags().IntVarP(&prListLimit, "limit", "n", 20, "number of PRs to list")

	prCreateCmd.Flags().StringVarP(&prTitle, "title", "t", "", "PR title")
	prCreateCmd.Flags().StringVarP(&prBody, "body", "b", "", "PR body")
	prCreateCmd.Flags().StringVar(&prHead, "head", "", "head branch (source)")
	prCreateCmd.Flags().StringVar(&prBase, "base", "main", "base branch (target)")

	prMergeCmd.Flags().StringVarP(&prMergeMsg, "message", "m", "", "merge commit message")

	prCmd.AddCommand(prListCmd)
	prCmd.AddCommand(prViewCmd)
	prCmd.AddCommand(prCreateCmd)
	prCmd.AddCommand(prMergeCmd)
	prCmd.AddCommand(prCloseCmd)
	prCmd.AddCommand(prReopenCmd)
	rootCmd.AddCommand(prCmd)
}
