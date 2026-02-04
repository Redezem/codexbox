# Repository Guidelines

## Project Structure & Module Organization
- `cmd/codexbox/` holds the CLI entry point (`main.go`).
- `internal/` contains implementation packages (e.g., `internal/docker/`, `internal/project/`, `internal/registry/`).
- Tests live alongside code as `*_test.go` (for example, `internal/project/project_test.go`).
- `Taskfile.yml` defines common dev tasks; `Dockerfile` defines the base image used by the CLI.

## Build, Test, and Development Commands
Use `go-task` if available:
- `task build` builds the CLI binary as `./codexbox`.
- `task test` runs unit tests (`go test ./...`).
- `task fmt` runs `gofmt -w cmd internal`.
- `task vet` runs `go vet ./...`.
- `task check` runs `fmt`, `vet`, and `test`.
- `task image-build` and `task image-update` build or update the base container image via `./codexbox image ...`.

## Coding Style & Naming Conventions
- Follow standard Go formatting; run `gofmt` before committing.
- Package names are lowercase, short, and descriptive (e.g., `project`, `registry`).
- Exported identifiers use `CamelCase`; unexported use `camelCase`.
- Keep file and directory names lowercase and Go-idiomatic.

## Testing Guidelines
- Prefer table-driven tests where multiple cases are needed.
- Name tests `TestXxx` and benchmark `BenchmarkXxx` in `*_test.go` files.
- Run `task test` (or `go test ./...`) before opening a PR.

## Commit & Pull Request Guidelines
- Current history only shows a single commit message: `Initial commit`. No convention is established yet.
- Until a convention is adopted, use short, imperative subjects (e.g., `Add project registry cache`).
- PRs should include: a short summary, key commands run (e.g., `task check`), and any behavior changes. If CLI output changes, include a brief before/after snippet.

## Security & Configuration Tips
- Secrets are passed via `OPENAI_API_KEY` and optional `OPENAI_BASE_URL`; avoid writing them to files.
- If `.codex/` appears in your workspace, run `codexbox init` to add it to `.gitignore`.
- Docker or Podman is required to run the CLI and build images.
