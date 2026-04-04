# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project does

`deploy-doc` is a Go CLI that automates the creation of deployment documents in Confluence. Given a Jira issue key and one or two commit hashes (backend/frontend), it fetches changed files via `git show`, queries Jira for the issue title, and creates/updates a Confluence page with a structured ADF (Atlassian Document Format) table of changed files.

## Commands

```bash
make build          # Compile binary to ./bin/deploy-doc
make install        # Build + install to ~/.local/bin
make run ARGS='...' # Run without compiling (dev mode)
make lint           # go fmt + go vet
make tidy           # go mod tidy
make build-all      # Cross-compile for Linux, Windows, Mac
make release        # lint + build-all + bump patch + git tag + gh release
```

There are no tests in this project currently.

## Architecture

The flow for `deploy-doc generate` is:

1. **`cmd/generate.go`** — parses `--issue`, `--commit-backend`, `--commit-frontend` flags, orchestrates all steps, and handles user interaction (prompts via stdin)
2. **`internal/config/config.go`** — loads credentials from env vars (`ATLASSIAN_EMAIL`, `ATLASSIAN_TOKEN`, `ATLASSIAN_BASE_URL`) or from `~/.config/deploy-doc/config.yaml`
3. **`internal/git/git.go`** — runs `git show --name-only` in the CWD to get changed files, groups them by directory
4. **`internal/atlassian/`** — thin HTTP client with Basic Auth (base64 email:token):
   - `client.go`: shared `Get`/`Post`/`Put` methods
   - `jira.go`: fetches issue summary via Jira REST API v3
   - `confluence.go`: searches for existing docs via CQL, creates/updates pages via Confluence REST API v2
5. **`internal/document/builder.go`** — builds the ADF JSON document from scratch using helper functions. The repo names `operativo-api` (backend) and `echo-logistics` (frontend) are **hardcoded** here in `runGenerate` and in `filesTable`. Bitbucket URLs are constructed as `https://bitbucket.org/devtyt/<repoName>`.

**Self-install behavior**: When the binary is run for the first time from outside its install location, `main.go` calls `installer.Run()` to copy itself to `~/.local/bin` (Linux/Mac) or `%LOCALAPPDATA%\Programs\deploy-doc` (Windows) and add it to `PATH`. Running via `go run` skips this (detects `/go-build/` in path).

## Key constraints

- The CLI uses **no third-party dependencies** — only stdlib. The router in `cmd/root.go` is a simple `map[string]func([]string)error`, not Cobra or similar.
- The config file format is plain `key: value` (not YAML-parsed with a library), so only the three known keys are read.
- `document/builder.go` constructs raw `map[string]any` for ADF — no typed structs. The ADF body is JSON-marshaled to a string and sent as the `body.value` field.
- The `generate` command must be run from inside the target git repository, as it calls `git show` in the CWD.
