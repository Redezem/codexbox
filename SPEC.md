# SPEC.md

This document defines CLI behavior, Go project layout, and the Docker image used by `codexbox`.

---

## CLI Help Output (target)

```
codexbox [flags]

Run or resume a Codex sandbox for the current project.

Commands:
  list               List project containers
  rm <project>      Remove a project container
  rebase <project>  Recreate project container using latest image
  init              Add .codex to .gitignore
  doctor            Check docker + environment
  config            Show configuration paths

  image build       Build base image
  image update      Update and rebuild base image

Flags:
  --engine docker|podman
  --image <tag>
  --project-scope repo|dir
  --shell
  --cmd "<command>"
  --fresh
  --readonly
  --cpus <n>
  --memory <size>
  --env-file <path>
  --no-gpu
```

---

## Go Repository Layout

```
codexbox/
  cmd/codexbox/main.go
  internal/cli/
  internal/docker/
  internal/project/
  internal/registry/
  internal/lock/
  internal/image/
  Dockerfile
  README.md
  SPEC.md
```

### Responsibilities

- `cli` – cobra/urfave CLI wiring
- `docker` – thin wrapper over docker CLI calls
- `project` – project ID calculation and path logic
- `registry` – JSON registry + container label recovery
- `lock` – file locks for concurrency
- `image` – build/update image logic

---

## Container Lifecycle

### First Run

```
docker create \
  --name codexbox-<project_id> \
  -v <project>:/workspace \
  -v codexbox-cache-go:/go/pkg/mod \
  -v codexbox-cache-cargo:/root/.cargo \
  -v codexbox-cache-npm:/root/.npm \
  -v codexbox-cache-pip:/root/.cache/pip \
  -w /workspace \
  --init \
  <image> sleep infinity
```

Then:

```
docker start ...
docker exec -it ... codex
```

### Resume

```
docker start ...
docker exec -it ... codex resume
```

### Exit

On Codex exit:

```
docker stop ...
```

---

## Dockerfile (Fedora)

```Dockerfile
FROM fedora:41

RUN dnf -y update && dnf -y install \
    git curl wget ca-certificates \
    gcc gcc-c++ clang lld make cmake ninja-build pkgconf-pkg-config gdb \
    python3 python3-pip python3-virtualenv \
    nodejs npm \
    go \
    && dnf clean all

# Rust via rustup
RUN curl https://sh.rustup.rs -sSf | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"

# Install Codex CLI (example via npm)
RUN npm install -g @openai/codex

WORKDIR /workspace
CMD ["sleep", "infinity"]
```

---

## Labels on Containers

```
com.codexbox.project_id
com.codexbox.path
com.codexbox.image_tag
com.codexbox.created_at
```

---

## Registry Location

- macOS: `~/Library/Application Support/codexbox/registry.json`
- Linux: `~/.local/share/codexbox/registry.json`
```