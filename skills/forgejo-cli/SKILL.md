---
name: forgejo-cli
description: Interact with Forgejo self-hosted Git server to manage repositories, issues, and pull requests.
---
# Forgejo CLI

## Overview

`forgejo-cli` is a command-line tool to interact with a self-hosted Forgejo server. It manages repositories, issues, and pull requests from the terminal.

The CLI configuration is stored at `~/.config/forgejo-cli/config.yaml` with server URL, owner, proxy, and API token.

## Prerequisites

Before using the CLI, ensure the SOCKS5 proxy is active (if configured):
```bash
# Test proxy connectivity if using SOCKS5 tunnel
nc -zv 127.0.0.1 1095
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--config <path>` | Use custom config file |
| `-o, --owner <name>` | Override repository owner |
| `-r, --repo <name>` | Specify repository name |
| `-O, --output <format>` | Output format: `text` (default) or `json` |

## Commands

### Repository Management

```bash
# List repositories (default limit: 20)
# Aliases: ls
forgejo-cli repo list
forgejo-cli repo list -n 50

# View repository details
# If OWNER/repo-name is not used, it defaults to the owner in config
forgejo-cli repo view OWNER/repo-name
forgejo-cli repo view repo-name

# Create a new repository (requires write:user scope)
forgejo-cli repo create my-new-repo --description "Project description"
forgejo-cli repo create private-repo -p --description "Private project"

# Delete a repository
forgejo-cli repo delete my-repo --yes  # skip confirmation
forgejo-cli repo delete my-org/my-repo  # with explicit owner
```

### Issue Management

```bash
# List issues in a repository
# Aliases: ls
forgejo-cli issue list -r repo-name
forgejo-cli issue list -r repo-name -s open     # filter by state: open/closed/all
forgejo-cli issue list -r repo-name -n 10       # limit to 10 issues
# Also supports positional repo and owner/repo format:
forgejo-cli issue ls my-repo
forgejo-cli issue list my-org/my-repo

# View specific issue
forgejo-cli issue view 1 -r repo-name

# Create an issue
# Required: -t/--title
forgejo-cli issue create -r repo-name -t "Bug report" -b "Description of the bug"

# Create issue with image attachments
forgejo-cli issue create -r repo-name -t "Screenshot bug" --attach ./screenshot.png
forgejo-cli issue create -r repo-name -t "Multiple images" --attach ./img1.png --attach ./img2.png

# Close an issue
forgejo-cli issue close 1 -r repo-name

# Reopen an issue
forgejo-cli issue reopen 1 -r repo-name
```

### Pull Request Management

```bash
# List pull requests
# Aliases: 'ls' for list, 'pull' for pr
forgejo-cli pr list -r repo-name
forgejo-cli pr list -r repo-name -s all         # show all PRs (open/closed/merged)
forgejo-cli pr list -r repo-name -n 10          # limit results
# Also supports positional repo and owner/repo format:
forgejo-cli pr ls my-repo
forgejo-cli pull list my-org/my-repo

# View specific PR
forgejo-cli pr view 1 -r repo-name

# Create a pull request
# Required: -t/--title, --head. --base defaults to 'main'
forgejo-cli pr create -r repo-name -t "New feature" -b "Description" --head feature-branch --base main

# Create PR with image attachments
forgejo-cli pr create -r repo-name -t "UI changes" --attach ./design.png --head feature/ui

# Merge a pull request
forgejo-cli pr merge 1 -r repo-name
forgejo-cli pr merge 1 -r repo-name -m "Custom merge message"

# Close a pull request
forgejo-cli pr close 1 -r repo-name

# Reopen a pull request
forgejo-cli pr reopen 1 -r repo-name
```

## Output Formats

### Text (Default)
Human-readable tables and formatted output.

### JSON
Useful for scripting and automation:
```bash
forgejo-cli repo list -O json
forgejo-cli issue view 1 -r repo-name -O json
forgejo-cli repo view repo-name -O json | jq '.stars_count'
```

## Configuration

Initialize the CLI with your Forgejo server details:
```bash
forgejo-cli init --server URL --token TOKEN --owner OWNER [--proxy PROXY_URL]
```

See `forgejo-cli init --help` for all options. Environment variables (`FORGEJO_CLI_*`) are also supported.

## Common Workflows

### Daily Development Workflow

```bash
# 1. Check open issues
forgejo-cli issue list -r myproject -s open

# 2. Create a new issue for a bug
forgejo-cli issue create -r myproject -t "Fix: login timeout" -b "Users report 500 error on login"

# 3. View issue details
forgejo-cli issue view 42 -r myproject

# 4. After fixing, close the issue
forgejo-cli issue close 42 -r myproject
```

### Pull Request Workflow

```bash
# 1. Create a PR after pushing a branch
forgejo-cli pr create -r myproject \
  -t "Add user authentication" \
  -b "Implements JWT-based auth" \
  --head feature/auth \
  --base main

# 2. View the PR
forgejo-cli pr view 5 -r myproject

# 3. After review, merge it
forgejo-cli pr merge 5 -r myproject
```

### Repository Discovery

```bash
# List all repos and filter with jq
forgejo-cli repo list -O json -n 100 | jq -r '.[].full_name' | grep "api"

# Find repos with open issues
forgejo-cli repo list -O json | jq '.[] | select(.open_issues_count > 0) | .full_name'
```

## Required Token Scopes

Based on Forgejo token scope documentation:

| Scope | Required For |
|-------|--------------|
| `read:repository` | List/view repositories |
| `write:repository` | Create PRs, merge PRs |
| `read:issue` | List/view issues |
| `write:issue` | Create/close/reopen issues, upload attachments |
| `write:user` | Create repositories |

## Constraints

- **Repository creation** requires `write:user` token scope
- **Proxy**: If configured, the SOCKS5 proxy must be active for connectivity
- **Owner override**: Use `-o` flag when working with repos outside the default organization

## Troubleshooting

### 403 Forbidden Errors
The token lacks required scope. Check token scopes in Forgejo user settings.

### Connection Errors
If using a proxy, ensure it's active:
```bash
nc -zv 127.0.0.1 1095  # test proxy connectivity
```

### 404 Not Found
The repository or issue may not exist, or the owner is incorrect:
```bash
# Check available repos
forgejo-cli repo list

# Override owner if needed
forgejo-cli -o DifferentOrg repo list
```

## Reference

- **Forgejo Token Scopes**: https://forgejo.org/docs/latest/user/token-scope/
- **Forgejo API Docs**: Available at your instance's `/api/swagger` endpoint
