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

lines = content.splitlines(keepends=True)

first_table_index = next(
    (index for index, line in enumerate(lines) if line.lstrip().startswith("[")),
    len(lines),
)
root_lines = lines[:first_table_index]
table_lines = lines[first_table_index:]

updated_root = []
replaced = False
skip_root_notify_block = False

for line in root_lines:
    stripped = line.lstrip()
    if skip_root_notify_block:
        if "]" in line:
            skip_root_notify_block = False
        continue
    if stripped.startswith("notify") and "=" in stripped:
        if not replaced:
            updated_root.append(notify_line)
            replaced = True
        if "[" in line and "]" not in line:
            skip_root_notify_block = True
        continue
    updated_root.append(line)

if not replaced:
    if updated_root and not updated_root[-1].endswith("\n"):
        updated_root[-1] += "\n"
    updated_root.append(notify_line)

updated_table = []
skip_table_notify_block = False
for line in table_lines:
    stripped = line.lstrip()
    if skip_table_notify_block:
        if "]" in line:
            skip_table_notify_block = False
        continue
    if stripped.strip() == notify_line.strip():
        if "[" in line and "]" not in line:
            skip_table_notify_block = True
        continue
    updated_table.append(line)

config_path.write_text("".join(updated_root + updated_table), encoding="utf-8")
PY
}

run_peon_startup_check() {
    peon_dir="${CLAUDE_PEON_DIR:-/usr/local/share/claude/hooks/peon-ping}"
    peon_sh="${peon_dir}/peon.sh"

    if [ ! -f "$peon_sh" ]; then
        return 0
    fi

    if output="$(PEON_TEST=1 PLATFORM=devcontainer bash "$peon_sh" 2>&1 <<'JSON'
{"hook_event_name":"SessionStart","cwd":"/workspace","session_id":"debug-s1","source":"codex"}
JSON
    )"; then
        return 0
    fi

    printf '%s\n' "codexbox-launch: warning: peon-ping startup check failed" >&2
    printf '%s\n' "$output" >&2
    return 0
}

ensure_peon_codex_notify
run_peon_startup_check
exec "$@"
