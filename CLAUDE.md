# bb — Bitbucket CLI

## Project Overview

A Go CLI tool for Bitbucket Cloud (REST API v2), built with Cobra + Charm.sh. Binary installs to `~/go/bin/bitb` via `go install ./...`.

## Architecture

```
main.go                  # entry point — calls cmd.Execute()
cmd/
  root.go                # Cobra root, PersistentPreRunE loads config+client into context
  auth.go                # bitb auth login/status/logout
  branch.go              # bitb branch list/delete
  pr.go                  # bitb pr list/view/create/merge/approve/diff
  repo.go                # bitb repo list/view
  issue.go               # bitb issue list/view/create
  pipeline.go            # bitb pipeline list/view
internal/
  api/client.go          # BitbucketClient, Paginate[T], APIError
  config/config.go       # Viper-based config, DetectRepo(), ResolveRepo()
  ui/ui.go               # lipgloss styles, Table renderer, FormatDate, etc.
```

## Key Patterns

**Config + client in context:** `PersistentPreRunE` in `root.go` loads config and builds the API client, storing both in `cmd.Context()`. Commands retrieve them via `clientFromCtx(cmd)` and `configFromCtx(cmd)`.

**Repo resolution:** `config.ResolveRepo(wsFlag, repoFlag, cfg)` cascades: explicit flag > `git remote get-url origin` > config default. All commands use this — never hardcode workspace/slug.

**Pagination:** Use `api.Paginate[T](client, path, params, max)` — follows Bitbucket's `next` links automatically. Returns `[]T`.

**Auth commands skip PersistentPreRunE:** Auth subcommands check `cmd.Parent().Use == "auth"` to skip config loading (config doesn't exist yet during login).

## Dependencies

- `github.com/spf13/cobra` — CLI framework
- `github.com/spf13/viper` — config (TOML at `~/.config/bb/config.toml`)
- `github.com/charmbracelet/lipgloss` — terminal styling and tables
- `github.com/charmbracelet/glamour` — markdown rendering (PR/issue descriptions)
- `github.com/charmbracelet/huh` — interactive forms (auth login)
- `net/http` + `encoding/json` — HTTP client (no external HTTP library)

## Bitbucket API

Base URL: `https://api.bitbucket.org/2.0`

Auth: HTTP Basic with Atlassian email + API token. **App passwords were deprecated June 9, 2026.**

PR diff endpoint returns `text/x-diff` (not JSON) — use `client.GetRaw()`.

Pipeline UUIDs have curly braces `{uuid}` — use `url.PathEscape()` when building paths.

Bitbucket Issues return 404 when disabled for a repo (many repos use Jira instead).

## Build & Install

```bash
go build ./...    # verify compilation
go install ./...  # update ~/go/bin/bb
```

Go is installed via Homebrew (`brew install go`). `$HOME/go/bin` is in PATH via `.zshrc`.
