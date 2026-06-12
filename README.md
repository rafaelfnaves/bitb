# bitb — Bitbucket CLI

A command-line interface for Bitbucket Cloud, inspired by GitHub's `gh` CLI. Manage branches, pull requests, issues, and pipelines directly from your terminal.

## Installation

**Requirements:** Go 1.21+ (installed via `brew install go`)

```bash
cd ~/personal/bitb
go install ./...
```

The `bitb` binary is installed to `~/go/bin/bitb`. Make sure `$HOME/go/bin` is in your `$PATH`.

## Authentication

Generate an API token at [id.atlassian.com/manage-profile/security/api-tokens](https://id.atlassian.com/manage-profile/security/api-tokens) (app passwords were deprecated in June 2026).

```bash
bitb auth login     # interactive setup
bitb auth status    # check connectivity
bitb auth logout    # remove credentials
```

Credentials are stored at `~/.config/bitb/config.toml` (mode 0600).

## Commands

### Branches

```bash
bitb branch list                        # list remote branches
bitb branch list --search "feature"     # filter by name
bitb branch list --author "Rafael"      # filter by author
bitb branch delete my-branch            # delete a branch (with confirmation)
bitb branch delete my-branch --yes      # skip confirmation
```

### Pull Requests

```bash
bitb pr list                            # list open PRs
bitb pr list --state MERGED             # list merged PRs
bitb pr view 42                         # view PR details (markdown rendered)
bitb pr view 42 --comments              # include comments
bitb pr view 42 --web                   # open in browser
bitb pr create                          # interactive creation (opens $EDITOR)
bitb pr create --title "Fix bug" --source my-branch --dest main
bitb pr merge 42                        # merge PR
bitb pr merge 42 --strategy squash      # squash merge
bitb pr approve 42                      # approve PR
bitb pr diff 42                         # show colored diff
```

### Repositories

```bash
bitb repo list                          # list all repos in workspace
bitb repo view                          # view current repo info
bitb repo view --web                    # open in browser
```

### Issues

```bash
bitb issue list                         # list open issues
bitb issue list --kind bug              # filter by kind
bitb issue view 15                      # view issue details
bitb issue create --title "Bug report"  # create issue
```

> **Note:** Issues require the repository to have Bitbucket Issues enabled. Repos using Jira will return an error.

### Pipelines

```bash
bitb pipeline list                      # list recent pipelines
bitb pipeline list --branch main        # filter by branch
bitb pipeline view 123                  # view pipeline steps and status
bitb pipeline view 123 --web            # open in browser
```

## Global Flags

Available on most commands:

| Flag | Description |
|---|---|
| `--repo workspace/slug` | Target a specific repo instead of auto-detecting from git remote |
| `--workspace` | Override workspace |
| `--json` | Output raw JSON (useful for scripting) |
| `--web` | Open the resource in the browser |
| `--limit N` | Maximum number of results to return |

## Auto-detection

`bitb` automatically detects the current repository from the git remote. Both SSH and HTTPS remotes are supported:

```
git@bitbucket.org:myworkspace/myrepo.git
https://bitbucket.org/myworkspace/myrepo
```

## Rebuilding after changes

```bash
cd ~/personal/bitb
go build ./...    # check for errors
go install ./...  # install updated binary
```
