# bb — Bitbucket CLI

A command-line interface for Bitbucket Cloud, inspired by GitHub's `gh` CLI. Manage branches, pull requests, issues, and pipelines directly from your terminal.

## Installation

**Requirements:** Go 1.21+ (installed via `brew install go`)

```bash
cd ~/personal/bb
go install ./...
```

The `bb` binary is installed to `~/go/bin/bb`. Make sure `$HOME/go/bin` is in your `$PATH`.

## Authentication

Generate an API token at [id.atlassian.com/manage-profile/security/api-tokens](https://id.atlassian.com/manage-profile/security/api-tokens) (app passwords were deprecated in June 2026).

```bash
bb auth login     # interactive setup
bb auth status    # check connectivity
bb auth logout    # remove credentials
```

Credentials are stored at `~/.config/bb/config.toml` (mode 0600).

## Commands

### Branches

```bash
bb branch list                        # list remote branches
bb branch list --search "feature"     # filter by name
bb branch list --author "Rafael"      # filter by author
bb branch delete my-branch            # delete a branch (with confirmation)
bb branch delete my-branch --yes      # skip confirmation
```

### Pull Requests

```bash
bb pr list                            # list open PRs
bb pr list --state MERGED             # list merged PRs
bb pr view 42                         # view PR details (markdown rendered)
bb pr view 42 --comments              # include comments
bb pr view 42 --web                   # open in browser
bb pr create                          # interactive creation (opens $EDITOR)
bb pr create --title "Fix bug" --source my-branch --dest main
bb pr merge 42                        # merge PR
bb pr merge 42 --strategy squash      # squash merge
bb pr approve 42                      # approve PR
bb pr diff 42                         # show colored diff
```

### Repositories

```bash
bb repo list                          # list all repos in workspace
bb repo view                          # view current repo info
bb repo view --web                    # open in browser
```

### Issues

```bash
bb issue list                         # list open issues
bb issue list --kind bug              # filter by kind
bb issue view 15                      # view issue details
bb issue create --title "Bug report"  # create issue
```

> **Note:** Issues require the repository to have Bitbucket Issues enabled. Repos using Jira will return an error.

### Pipelines

```bash
bb pipeline list                      # list recent pipelines
bb pipeline list --branch main        # filter by branch
bb pipeline view 123                  # view pipeline steps and status
bb pipeline view 123 --web            # open in browser
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

`bb` automatically detects the current repository from the git remote. Both SSH and HTTPS remotes are supported:

```
git@bitbucket.org:myworkspace/myrepo.git
https://bitbucket.org/myworkspace/myrepo
```

## Rebuilding after changes

```bash
cd ~/personal/bb
go build ./...    # check for errors
go install ./...  # install updated binary
```
