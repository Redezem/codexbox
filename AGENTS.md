# Repository Guidelines

## Living Documentation Rule
- `AGENTS.md` and `README.md` are required maintenance surfaces. If behavior, commands, defaults, image contents, startup flow, flags, or contributor workflow change, update both files in the same change unless there is a strong reason not to.
- Treat the code as the source of truth. If docs and implementation disagree, fix the docs immediately.
- When touching container startup, image contents, CLI flags, registry behavior, or project detection, explicitly review both `AGENTS.md` and `README.md` before finishing.

## Project Overview
- `codexbox` is a Go CLI that gives each project a persistent Docker or Podman container and runs OpenAI Codex inside it.
- Default invocation runs `/usr/local/bin/codexbox-launch`, not `codex` directly. The launch wrapper bootstraps Codex notify integration for peon-ping and runs a peon-ping startup self-check before executing Codex.
- The base image is Fedora-based and installs Go, .NET, Rust, Node.js, Python, zsh, `task`, `mise`, `@openai/codex`, and peon-ping.
- peon-ping is installed in the image under `/usr/local/share/claude/hooks/peon-ping`, and the default configured voice pack is `peasant`.
- Project containers are long-lived and per-project. `codexbox` starts the container, runs the session, then stops the container without deleting it.

## Project Structure & Module Organization
- `cmd/codexbox/main.go` is the CLI entry point.
- `internal/cli/` contains Cobra command wiring, runtime defaults, container creation, and top-level command behavior.
- `internal/docker/` wraps Docker or Podman CLI calls.
- `internal/project/` handles project detection, project IDs, container names, and cache volume names.
- `internal/registry/` stores project metadata on disk.
- `internal/lock/` provides the registry file lock used around list, remove, and rebase operations.
- `internal/image/` builds the base image from embedded assets.
- `internal/image/assets/Dockerfile` is the embedded image definition used by `codexbox image build`.
- `internal/image/assets/codexbox-launch.sh` is the embedded startup wrapper copied into the image.
- Root `Dockerfile` mirrors the embedded Dockerfile for direct/manual image work and must stay in sync with `internal/image/assets/Dockerfile`.
- Tests live beside code in `*_test.go`.

## Build, Test, and Development Commands
Use `go-task` if available:
- `task build` builds `./codexbox`.
- `task fmt` runs `gofmt -w cmd internal`.
- `task tidy` runs `go mod tidy`.
- `task test` runs `go test ./...`.
- `task vet` runs `go vet ./...`.
- `task check` runs `fmt`, `vet`, and `test`.
- `task image-build` runs `./codexbox image build`.
- `task image-update` runs `./codexbox image update`.
- `task clean` removes `./codexbox`.

Direct Go equivalents:
- `go build -o codexbox ./cmd/codexbox`
- `go test ./...`
- `go vet ./...`
- `gofmt -w cmd internal`

Important workflow note:
- `task image-build` and `task image-update` require `./codexbox` to exist first. Build the binary before using those tasks.

## Runtime Behavior
- Default command path: `codexbox` -> container start -> `codexbox-launch` -> peon-ping config ensure -> peon-ping startup test -> `codex --dangerously-bypass-approvals-and-sandbox`.
- `--shell` bypasses the launch wrapper and starts `bash`.
- `--cmd` bypasses the launch wrapper and runs `sh -lc <command>`.
- Containers mount the project at `/workspace`.
- Containers also mount host `~/.codex` to `/root/.codex`, even when the project workspace is read-only.
- Shared per-project cache volumes are mounted for Go, Cargo, npm, and pip.
- On Linux, the container is created with the host UID:GID to avoid root-owned files in the workspace.
- Runtime environment forwards `OPENAI_API_KEY` and optional `OPENAI_BASE_URL` into the container when present.
- `REMOTE_CONTAINERS=true`, `CODEXBOX=true`, `CLAUDE_PEON_DIR`, and `CODEX_HOME` are injected into the container runtime.

## Image Build Sources
- `codexbox image build` and `codexbox image update` do not build from the root `Dockerfile` directly.
- The CLI embeds `internal/image/assets/Dockerfile` and `internal/image/assets/codexbox-launch.sh` with `go:embed`, writes them to a temp directory, and builds from that generated context.
- If you update image behavior, you almost always need to touch:
  - `internal/image/assets/Dockerfile`
  - `internal/image/assets/codexbox-launch.sh` when startup behavior changes
  - root `Dockerfile` to keep manual builds consistent
  - `internal/image/image_test.go` if wrapper behavior changes

## Coding Style & Naming Conventions
- Follow standard Go formatting. Run `gofmt` before finishing.
- Package names should be lowercase, short, and descriptive.
- Exported identifiers use `CamelCase`; unexported identifiers use `camelCase`.
- Keep file and directory names lowercase and Go-idiomatic.
- Prefer small focused helpers over wide command functions when behavior becomes testable in isolation.
- Preserve the current style of using the standard library plus Cobra; do not introduce new heavy dependencies casually.

## Testing Guidelines
- Prefer table-driven tests when multiple cases exist.
- Name tests `TestXxx` and benchmarks `BenchmarkXxx`.
- Run `task test` or `go test ./...` before wrapping up changes.
- When changing CLI execution selection, update `internal/cli/root_test.go`.
- When changing launch script behavior, update `internal/image/image_test.go`.
- When changing registry serialization or defaults, update `internal/registry/registry_test.go`.
- When changing project ID logic, update `internal/project/project_test.go`.

## Commit & Pull Request Guidelines
- Use short, imperative commit subjects, for example `Add peon startup verification`.
- PRs should include:
  - a short summary
  - commands run for validation, for example `task check`
  - any behavior changes
  - before/after CLI output when user-facing output changes
  - documentation updates when behavior changed

## Security & Configuration Tips
- Never bake secrets into images or files.
- Secrets belong in `OPENAI_API_KEY`, optional `OPENAI_BASE_URL`, or an env file passed with `--env-file`.
- If `.codex/` appears in a workspace, `codexbox init` adds it to `.gitignore`.
- Docker or Podman is required to run the CLI and build images.

## Contributor Guide

### What To Read First
- Read `README.md` for the user-facing contract.
- Read `internal/cli/root.go` for actual runtime behavior.
- Read `internal/image/assets/Dockerfile` and `internal/image/assets/codexbox-launch.sh` for image and startup logic.
- Read the nearest `*_test.go` file before changing a package.

### Safe Change Process
- Start by locating the true source file. Do not assume the root `Dockerfile` is enough for image changes.
- Change implementation first.
- Update tests for the affected package.
- Update `AGENTS.md` and `README.md` if the behavior or workflow changed.
- Run the narrowest useful validation first, then broader checks.
- Keep diffs tight. This repo is small enough that broad refactors are rarely justified for a one-feature change.

### Common Pitfalls
- `internal/image/assets/Dockerfile` is the image source for CLI builds. Editing only the root `Dockerfile` will not change `codexbox image build`.
- The inverse is also a pitfall: editing only the embedded asset Dockerfile leaves the root `Dockerfile` stale for manual builds and review.
- Default `codexbox` execution goes through `codexbox-launch`; `--shell` and `--cmd` do not. If a feature depends on the wrapper, document that scope clearly.
- Launch script changes need shell-level care. The script runs under `/bin/sh`, so avoid Bash-only syntax there unless you intentionally invoke `bash` for a subprocess.
- Registry operations are lock-sensitive. Reuse the existing lock helpers rather than adding unlocked reads or writes.
- `task image-build` and `task image-update` call `./codexbox`, so they fail if the binary was not built first.
- `list` iterates a Go map, so output order is not stable. Tests or docs should not imply ordering.
- `rm` expects a project ID, not a filesystem path.
- `rebase [project]` requires the project to exist in the registry first.
- Container creation mounts the host `~/.codex`; if you change `CODEX_HOME` behavior, review both mount logic and launch script config resolution.
- Linux containers run as the host UID:GID. Any change that assumes root in the workspace can break real usage.
- peon-ping config has evolved. Upstream now uses `default_pack`, with `active_pack` as a legacy compatibility concern. Do not reintroduce stale assumptions.
- peon-ping startup validation in `codexbox-launch` is intentionally non-fatal. Keep that property unless there is a strong product reason to block startup.

### High-Value Validation Patterns
- For normal Go changes: `task check`
- For CLI behavior only: `go test ./internal/cli ./internal/project ./internal/registry`
- For launch wrapper changes: `go test ./internal/image`
- For image changes: `task build` then `./codexbox image build`
- For docs-only changes: verify examples and file paths against current code before finishing

### Documentation Expectations
- User-facing behavior belongs in `README.md`.
- Agent/contributor workflow, pitfalls, and internal maintenance rules belong in `AGENTS.md`.
- If you add a new flag, command, environment variable, mount, cache, or startup step, update both docs.
- If you remove behavior, remove stale documentation in the same change. Do not leave “future cleanup” documentation debt behind.

### Review Checklist For Future Agents
- Is the actual source file being edited, or only a mirror/generated copy?
- Are tests updated where behavior changed?
- Are `AGENTS.md` and `README.md` updated if the change affects users or maintainers?
- Does the change preserve container lifecycle expectations?
- Does the change preserve registry locking where needed?
- Does the change preserve Linux UID:GID friendliness?
- Does the change preserve peon-ping startup behavior and wrapper semantics where expected?

## Current Command Surface
- `codexbox`
- `codexbox list`
- `codexbox rm <project>`
- `codexbox rebase [project]`
- `codexbox init`
- `codexbox doctor`
- `codexbox config`
- `codexbox status`
- `codexbox image build`
- `codexbox image update`

Supported persistent flags:
- `--engine docker|podman`
- `--image <tag>`
- `--project-scope repo|dir`
- `--shell`
- `--cmd "<command>"`
- `--fresh`
- `--readonly`
- `--cpus <n>`
- `--memory <size>`
- `--env-file <path>`
- `--no-gpu`

## Summarized Changelog
- Initial project: introduced the `codexbox` CLI, project detection, container lifecycle, and registry persistence.
- Project identity and caching: added stable project IDs, shared per-project cache volumes, and image embedding support.
- Image improvements: switched to dynamic Go installation, added `task`, added .NET SDK and `zsh`, and updated the Fedora base images.
- Runtime defaults: default execution now uses Codex bypass mode instead of the earlier created-flag flow.
- Registry safety: added a lock helper and applied it to list, remove, and rebase operations.
- peon-ping integration: added peon-ping to the image, mounted Codex config into the container, added the `codexbox-launch` wrapper, and wired notify integration.
- peon-ping launch hardening: launch script now inserts the top-level `notify` setting before the first TOML table, removes duplicate broken nested notify entries, sets the default voice pack to `peasant`, and performs a startup `SessionStart` self-check in test mode.
