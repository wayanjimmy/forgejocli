package cmd

import (
	"fmt"
	"strconv"

	"github.com/jedib0t/go-pretty/v6/table"
	apiclient "github.com/jimboylabs/forgejocli/internal/api"
	"github.com/spf13/cobra"
)

var repoListLimit int

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repositories",
}

var repoListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List repositories",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		owner := cfg.Owner
		if flagOwner != "" {
			owner = flagOwner
		}

		repos, err := apiClient.ListRepos(owner, 1, repoListLimit)
		if err != nil {
			return err
		}

		if len(repos) == 0 {
			if isJSON() {
				return outputJSON([]interface{}{})
			}
			fmt.Println("No repositories found.")
			return nil
		}

		if isJSON() {
			return outputJSON(repos)
		}

		t := table.NewWriter()
		t.SetStyle(table.StyleLight)
		t.AppendHeader(table.Row{"FULL_NAME", "VISIBILITY", "STARS", "ISSUES", "DEFAULT_BRANCH", "UPDATED"})

		for _, r := range repos {
			vis := "public"
			if r.Private {
				vis = "private"
			}
			t.AppendRow(table.Row{
				r.FullName,
				vis,
				r.Stars,
				r.OpenIssues,
				r.DefaultBranch,
				r.UpdatedAt.Format("2006-01-02"),
			})
		}

		fmt.Println(t.Render())
		return nil
	},
}

var repoViewCmd = &cobra.Command{
	Use:   "view <repo>",
	Short: "View repository details",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(args)
		if repo == "" {
			return fmt.Errorf("repository name is required (use --repo or pass as argument)")
		}

		r, err := apiClient.GetRepo(owner, repo)
		if err != nil {
			return err
		}

		if isJSON() {
			return outputJSON(r)
		}

		fmt.Printf("name:           %s\n", r.FullName)
		fmt.Printf("description:    %s\n", r.Description)
		fmt.Printf("url:            %s\n", r.HTMLURL)
		fmt.Printf("ssh_url:        %s\n", r.SSHURL)
		fmt.Printf("clone_url:      %s\n", r.CloneURL)
		fmt.Printf("default_branch: %s\n", r.DefaultBranch)
		fmt.Printf("stars:          %d\n", r.Stars)
		fmt.Printf("forks:          %d\n", r.Forks)
		fmt.Printf("open_issues:    %d\n", r.OpenIssues)
		fmt.Printf("private:        %v\n", r.Private)
		fmt.Printf("updated_at:     %s\n", r.UpdatedAt.Format("2006-01-02 15:04:05"))

		return nil
	},
}

var repoCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		desc, _ := cmd.Flags().GetString("description")
		private, _ := cmd.Flags().GetBool("private")

		r, err := apiClient.CreateRepo(apiclient.CreateRepoOption{
			Name:        name,
			Description: desc,
			Private:     private,
			AutoInit:    true,
		})
		if err != nil {
			return err
		}

		if isJSON() {
			return outputJSON(r)
		}

		fmt.Printf("created: %s\n", r.FullName)
		fmt.Printf("url:     %s\n", r.HTMLURL)
		return nil
	},
}

var repoDeleteCmd = &cobra.Command{
	Use:   "delete <repo>",
	Short: "Delete a repository",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := resolveOwnerRepo(args)
		if repo == "" {
			return fmt.Errorf("repository name is required")
		}

		confirm, _ := cmd.Flags().GetBool("yes")
		if !confirm {
			fmt.Printf("Delete %s/%s? [y/N] ", owner, repo)
			var input string
			fmt.Scanln(&input)
			if input != "y" && input != "Y" {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		if err := apiClient.DeleteRepo(owner, repo); err != nil {
			return err
		}

		fmt.Printf("deleted: %s/%s\n", owner, repo)
		return nil
	},
}

func init() {
	repoListCmd.Flags().IntVarP(&repoListLimit, "limit", "n", 20, "number of repos to list")
	repoCreateCmd.Flags().StringP("description", "d", "", "repository description")
	repoCreateCmd.Flags().BoolP("private", "p", false, "make repository private")
	repoDeleteCmd.Flags().Bool("yes", false, "skip confirmation")

	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoViewCmd)
	repoCmd.AddCommand(repoCreateCmd)
	repoCmd.AddCommand(repoDeleteCmd)
	rootCmd.AddCommand(repoCmd)
}

// resolveOwnerRepo gets owner and repo from flags, args, or config
func resolveOwnerRepo(args []string) (string, string) {
	owner := cfg.Owner
	repo := flagRepo

	if len(args) > 0 {
		// Check if it's owner/repo format
		for i, c := range args[0] {
			if c == '/' {
				return args[0][:i], args[0][i+1:]
			}
		}
		repo = args[0]
	}

	if flagOwner != "" {
		owner = flagOwner
	}

	return owner, repo
}

// parseInt helper
func parseInt(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
