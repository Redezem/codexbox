#!/bin/sh
set -eu

resolve_codex_home() {
    if [ -n "${CODEX_HOME:-}" ]; then
        printf '%s\n' "$CODEX_HOME"
        return
    fi
    if [ -d /root/.codex ] || [ -f /root/.codex/config.toml ]; then
        printf '%s\n' "/root/.codex"
        return
    fi
    printf '%s\n' "${HOME:-/root}/.codex"
}

ensure_peon_codex_notify() {
    peon_dir="${CLAUDE_PEON_DIR:-/usr/local/share/claude/hooks/peon-ping}"
    peon_sh="${peon_dir}/peon.sh"
    adapter="${peon_dir}/adapters/codex.sh"

    if [ ! -f "$peon_sh" ]; then
        printf '%s\n' "codexbox-launch: warning: peon-ping is enabled but ${peon_sh} is missing" >&2
        return 0
    fi
    if [ ! -f "$adapter" ]; then
        printf '%s\n' "codexbox-launch: warning: peon-ping Codex adapter is missing at ${adapter}" >&2
        return 0
    fi

    codex_home="$(resolve_codex_home)"
    export CODEX_HOME="$codex_home"
    mkdir -p "$codex_home"
    config_path="${codex_home}/config.toml"

    python3 - "$config_path" "$adapter" <<'PY'
import pathlib
import sys

config_path = pathlib.Path(sys.argv[1])
adapter_path = sys.argv[2]
notify_line = f'notify = ["bash", "{adapter_path}"]\n'

if config_path.exists():
    content = config_path.read_text(encoding="utf-8")
else:
    content = ""

if "adapters/codex.sh" in content or "adapters/codex.ps1" in content:
    raise SystemExit(0)

lines = content.splitlines(keepends=True)
updated = []
replaced = False
skip_notify_block = False

for line in lines:
    stripped = line.lstrip()
    if skip_notify_block:
        if "]" in line:
            skip_notify_block = False
        continue
    if not replaced and stripped.startswith("notify") and "=" in stripped:
        replaced = True
        updated.append(notify_line)
        if "[" in line and "]" not in line:
            skip_notify_block = True
        continue
    updated.append(line)

if not replaced:
    if updated and not updated[-1].endswith("\n"):
        updated[-1] += "\n"
    updated.append(notify_line)

config_path.write_text("".join(updated), encoding="utf-8")
PY
}

ensure_peon_codex_notify
exec "$@"
