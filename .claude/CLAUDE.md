# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project does

`gtt` is a Go CLI that automates documentation in Atlassian for the deployment
workflow. The repo directory and Go module are still named `deploy-doc` (the
binary was renamed in v1.2.0); user-facing strings, binaries and the config
directory all say `gtt`.

Commands:

| Command | What it does |
|---|---|
| `init` | Interactive wizard: credentials, default space, first project. Validates the token against `/rest/api/3/myself`. |
| `g` / `gen` / `generate` | Creates or updates a deploy document in Confluence from a Jira issue key plus one or more commit hashes. |
| `qa` | Builds the QA consolidated report in Confluence. Two modes: Sprint (`-s` + `-m`) and Kanban (no flags — last 10 business days). |
| `f` / `fetch` | Exports a Confluence page to `.txt`, found by issue key. Opens a native save dialog. |
| `project` | `list` / `ls`, `add`, `default`, `remove`. |
| `update` | Self-update from GitHub Releases, with SHA-256 verification. |
| `backlog scan` | Detects technical debt in a local repo (hotspots by churn, markers, large files, untested classes, `composer audit` with `--deps`). Read-only, offline by default, needs no Atlassian credentials (`config.LoadLocal`). |
| `version`, `help` | — |

## Commands

```bash
make build          # Compile binary to ./bin/gtt
make install        # Build + install to ~/.local/bin
make run ARGS='...' # Run without compiling (dev mode)
make fmt            # go fmt
make vet            # go vet
make lint           # fmt + vet
make test           # go test ./...
make tidy           # go mod tidy
make build-all      # Cross-compile for Linux, Windows, Mac
make release        # lint + build-all + checksums + bump patch + git tag + gh release
make release-minor  # same, bumping minor
make release-major  # same, bumping major
```

`make lint` does NOT run the tests — `make test` is separate. `make release`
runs `lint` only.

Go toolchain: `go.mod` declares `go 1.26.1`. Do not change that directive
without checking what every developer and CI has installed.

## Architecture

```
cmd/            UI, flag parsing, orchestration, all stdin prompts
internal/
  config/       Config + ProjectConfig + QAReportConfig (YAML)
  git/          git show --name-only, grouping, error translation
  atlassian/    HTTP client (Basic Auth) + Jira v3 + Confluence v1/v2
  backlog/      Debt detectors over git history and tracked files (no network)
  document/     ADF construction and section preservation
  installer/    Self-install on first run
  updater/      Version check, cached notice, self-update
  build/        Version var (ldflags)
```

The flow for `gtt generate`:

1. **`cmd/generate.go`** — parses flags, resolves the project, orchestrates the
   four steps, handles every prompt.
2. **`internal/config/config.go`** — env vars take priority over
   `~/.config/gtt/config.yaml`. `MigrateIfNeeded` moves a legacy
   `~/.config/deploy-doc/config.yaml` on first run of v1.2.0+.
3. **`internal/git/git.go`** — `GetChangedFilesMulti` runs `git show` in the
   configured repo path (or the CWD), `GroupByDirectory` groups by parent dir,
   `explainGitError` turns git's stderr into actionable instructions.
4. **`internal/atlassian/`** — `client.go` (Get/Post/Put + status→message),
   `jira.go`, `jira_qa.go`, `confluence.go`, `confluence_fetch.go`,
   `storage_text.go`, `query.go`.
5. **`internal/document/`** — `builder.go` (deploy doc ADF), `qa_builder.go`
   (QA report ADF), `preserve.go` (reads a section back out of an existing doc).

**Self-install behavior**: on first run from outside the install location,
`main.go` asks for confirmation and then calls `installer.Run()` to copy itself
to `~/.local/bin` (Linux/Mac) or `%LOCALAPPDATA%\Programs\gtt` (Windows) and add
it to PATH. Running via `go run` skips this (detects `/go-build/` in the path).

**Update notice**: served from `~/.config/gtt/version_check.json`, refreshed in
the background at most once per 24h, suppressed when stdout is not a TTY or when
`GTT_NO_UPDATE_CHECK` is set.

## Key constraints

- **Dependencies**: `gopkg.in/yaml.v3` and `golang.org/x/sys` (Windows registry)
  only. The command router in `cmd/root.go` is a `map[string]func([]string) error`,
  not Cobra. Keep it that way unless there is a strong reason.
- **ADF is raw `map[string]any`** — no typed structs. The body is JSON-marshaled
  to a string and sent as `body.value` with `representation: atlas_doc_format`.
- **Confluence v2 filters spaces by numeric `space-id`, never `space-key`.**
  An unknown query param is ignored silently, so a wrong name turns the filter
  into a no-op. Resolve keys with `ResolveSpaceID`.
- **Every value interpolated into JQL or CQL must go through
  `atlassian.quoteLiteral`.** Never place a value inside hand-written quotes in
  a format string.
- **`generate` replaces the whole page body on update.** Anything a user edits
  by hand in Confluence is lost unless it is read back and re-emitted — see
  `document.ExtractSection` and `DeployDoc.PreservedConsider`. If the previous
  body cannot be read, do not overwrite: warn and ask.
- **Never render a failed lookup as a failed check** in the QA report. It is
  published as evidence; use the unknown state (`QAIssue.DeployDocUnknown`).
- **No corporate data in the source.** Instance URLs, space keys, repo names,
  VCS org, personal names and the deploy checklist live in config
  (`qa_report`, `deploy_checklist` — the latter also per project), with a CLI
  flag to override where one makes sense. The layering is documented in
  `docs/security/patrones-seguros.md`, which is the authority on this — read it
  before adding any new value. That includes example values in help text: a
  workspace name is topology (P-001), not a scanner false positive (P-007).
- **Report a discarded error whose effect is delayed.** `CreateJiraRemoteLink`
  failing is not fatal, but `qa` verifies that very link — silence there
  resurfaces weeks later as a task wrongly flagged as undocumented. See
  `linkIssueToDoc`.
- The `qa` command's `qa_email` gate is a UX guardrail, not access control: it
  reads from the user's own config. Real authorization is Atlassian permissions.

## Testing

`go test ./...` covers pure functions only — no HTTP mocking of Atlassian.
Covered: `document.ExtractSection` and `Build`, `commitFileURL`, `BuildTitle`,
`atlassian.quoteLiteral`, `parseDevTaskKey`, `BuildReviewMap`, `businessDaysAgo`,
`updater.isNewer`, `git.GroupByDirectory`, `explainGitError`, `cmd.parseFlags`,
`splitHashes`, `sanitizeFilename`, `StorageToText`, `BuildIssueTxt`,
`config.Load` / `LoadLocal` (isolated from the real config.yaml with a temp HOME),
the `backlog` detectors and ranking, plus one end-to-end `backlog.Scan` over a
throwaway git repo (skipped when git is not installed).

**Never run a built binary to try a change** (`go build -o x && ./x`): run from
outside the install dir, `main.go` asks to self-install and an empty stdin
answers yes, overwriting the user's installed `gtt`. Use `go run .`, which the
installer skips (`/go-build/` in the path).

Anything touching the network is verified by hand against a real instance;
`--dry-run` on `generate` and `qa` prints the ADF without publishing.

## Versions and releases

**At the end of every task that changes files in this repo, run the
`version-check` skill** (`bash .claude/skills/version-check/check.sh`) and
include its summary in your report: current published version, next version
and target, blockers and warnings. It is read-only.

- `make release*` does `git add -A`, commit, tag and push **on the current
  branch**: release only from an up-to-date `main`, with a clean tree. On
  2026-10-01 a release from a feature branch produced three tags for one change
  (v1.3.1, v1.4.0, v1.4.1).
- `internal/build/version.go` always says `"dev"`; the real version comes from
  the tag via `-ldflags`. Never read it to know the current version.
- **Never add new content to the bitácora entry of a published version.** Only
  clarifying notes. New work goes into the entry of the next version, which
  must state the real base in `**Versión anterior:**` and be listed in
  `docs/bitacora/README.md`.
- Never delete or move a published tag: document it instead.

## Docs

- `docs/arquitectura.md` — architecture notes
- `docs/guia-de-usuario.md` — end-user guide
- `docs/bitacora/` — one entry per version: Solicitud → Motivación → Diseño
  técnico → Archivos → Cómo verificar. Add an entry for every release.
- `docs/security/patrones-seguros.md` — secure patterns P-001..P-008. Add a new
  pattern whenever a review finds an anti-pattern.
