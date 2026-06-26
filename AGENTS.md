# AGENTS.md

Guidance for AI coding agents working in this repository. Read this before exploring further.

## Project snapshot

- **What it is:** `go-ripgrep` — a pure-Go port of [ripgrep](https://github.com/BurntSushi/ripgrep). It ships both a CLI tool (`rg`) that mirrors ripgrep's interface and an embeddable Go SDK.
- **Language:** Go (module requires Go **1.26+**, see `go.mod`).
- **Dependencies:** None — standard library only (`go.mod` has no `require` block). Keep it that way unless there's a strong reason.
- **Distribution:** Single static binary; cross-compiled for Linux (amd64/arm64/loong64), macOS (amd64/arm64), Windows (amd64/arm64); also published as npm packages.
- **License:** MIT.

## Important directories

- `cmd/rg/` — CLI entry point. `main.go` parses args into `CliArgs`, converts to SDK `Options`, and drives output.
- `sdk.go` (repo root) — Public SDK. `Search(ctx, paths, opts)` orchestrates matcher, glob set, directory walker, and worker pool; returns a streaming `<-chan printer.FileResult`.
- `pkg/matcher/` — Pattern matching. `RegexMatcher` (Go `regexp`/RE2) and `FixedMatcher` (`bytes.Contains`); `BuildMatcher()` selects one.
- `pkg/searcher/` — Per-file line-by-line reading, binary detection (NUL byte in first 1024 bytes), and before/after context handling.
- `pkg/printer/` — Output formatting: text (grouped/non-grouped), NDJSON (`--json`), and ANSI color.
- `pkg/globset/` — Compiles `-g`/`--glob` patterns to regex; implements include/exclude with negation (`!`).
- `pkg/ignore/` — Stack-based `.gitignore`/`.ignore`/`.rgignore` handling for nested directories; hidden-file rules.
- `tests/` — `integration_test.go` (package `tests`), end-to-end tests.
- `docs/` — Detailed docs: `architecture.md`, `cli-reference.md`, `sdk-reference.md`, `getting-started.md`, `npm-integration.md`. Consult `architecture.md` for the full data-flow/concurrency model.
- `scripts/` — npm packaging helpers (`build-npm-packages.sh`, `sync-npm-version.sh`, `npm-installer-wrapper.js`).
- `npm/` — npm package manifest and scaffolding.
- `bin/` — build output (generated, do not commit artifacts).

## Architecture notes

- **Pipeline:** Input paths → directory walker → file discovery → pattern matching → result formatting → output.
- **Concurrency:** One walker goroutine feeds `filesChan` (cap `threads*4`); N worker goroutines search files and feed `outChan` (cap `threads*2`); a closer goroutine `wg.Wait()`s then closes `outChan`. Default worker count is `runtime.NumCPU()` (overridable via `Options.Threads`).
- **Cancellation:** Every goroutine checks `ctx.Done()` at directory, file, and channel-op boundaries. Preserve these checks when editing the walker or workers.
- **No shared mutable state:** data flows through channels; avoid introducing mutexes unless unavoidable.
- **Module path quirk:** the module is `go-ripgrep` and the root package is named `goriggrep` (note the spelling). Internal imports use paths like `go-ripgrep/pkg/matcher`. Don't "fix" the package name casually — it would break imports across the codebase.

## Build / test / run commands

Use the `Makefile` (run `make help` for the full list):

- `make build` — build current-platform binary to `bin/rg`.
- `make test` — `go test -v -race ./...` (always run before finishing changes).
- `make fmt` — `gofmt -w .` then `goimports`/`go fmt`.
- `make run` — build and run `./bin/rg`.
- `make build-all` — cross-compile all platforms into `bin/`.
- `make install` — `go install ./cmd/rg`.
- `make clean` — remove `bin/` and npm artifacts.

Direct Go commands also work: `go build ./cmd/rg`, `go test ./...`, `go vet ./...`.

npm packaging: `make npm-version`, `make npm-packages`, `make npm-pack`, `make npm-publish-all`.

## Coding conventions and working rules

- Format with `gofmt`/`goimports` before committing (`make fmt`). Code must pass `go vet ./...` cleanly.
- Keep the standard-library-only constraint; do not add third-party dependencies without strong justification.
- Match existing style: small focused packages under `pkg/`, table-driven tests in `*_test.go` next to the code.
- Add/extend tests for behavior changes — unit tests in the relevant `pkg/` and end-to-end coverage in `tests/integration_test.go`.
- Maintain ripgrep CLI compatibility: when touching `cmd/rg/main.go`, keep flag names/semantics aligned with ripgrep and update `docs/cli-reference.md`.
- When changing the SDK surface (`sdk.go` `Options`/`Search`), update `docs/sdk-reference.md` accordingly.
- Regex matching uses Go's RE2 — no PCRE2/backreferences. Don't promise features RE2 can't support.

## Things an agent must NOT do

- Don't add external/third-party dependencies or CGO; the project is pure Go, std-lib only, single-binary by design.
- Don't rename the root package (`goriggrep`) or the module path (`go-ripgrep`) without updating every import — avoid unless explicitly requested.
- Don't remove `ctx.Done()` cancellation checks or convert the channel-based design to shared-state/mutex patterns.
- Don't commit build artifacts (`bin/`, `npm/packages/`, `*.tgz`).
- Don't publish npm packages (`make npm-publish-all`) or bump versions unless explicitly asked.
- Don't break ripgrep CLI flag compatibility or the NDJSON output format.
