FROM fedora:rawhide

ENV REMOTE_CONTAINERS=true \
    CODEXBOX=true \
    CLAUDE_CONFIG_DIR=/usr/local/share/claude \
    CLAUDE_PEON_DIR=/usr/local/share/claude/hooks/peon-ping

RUN dnf -y update && dnf -y install \
    git curl wget ca-certificates \
    gcc gcc-c++ clang lld make cmake ninja-build pkgconf-pkg-config gdb \
    python3 python3-pip python3-virtualenv \
    nodejs npm zsh \
    && dnf clean all

RUN python3 - <<'PY' >/tmp/go-version
import json,urllib.request
with urllib.request.urlopen('https://go.dev/dl/?mode=json') as resp:
    data = json.load(resp)
for rel in data:
    if rel.get('stable'):
        print(rel['version'])
        break
else:
    raise SystemExit('no stable Go release found')
PY

RUN set -eu; \
    GO_VERSION="$(cat /tmp/go-version)"; \
    rm -f /tmp/go-version; \
    ARCH="$(uname -m)"; \
    case "${ARCH}" in \
        x86_64) GO_ARCH="amd64" ;; \
        aarch64) GO_ARCH="arm64" ;; \
        *) echo "unsupported arch: ${ARCH}" >&2; exit 1 ;; \
    esac; \
    curl -sSfL "https://go.dev/dl/${GO_VERSION}.linux-${GO_ARCH}.tar.gz" -o /tmp/go.tgz; \
    rm -rf /usr/local/go; \
    tar -C /usr/local -xzf /tmp/go.tgz; \
    rm -f /tmp/go.tgz
ENV PATH="/usr/local/go/bin:${PATH}"

RUN python3 - <<'PY' >/tmp/dotnet-version
import json,urllib.request
with urllib.request.urlopen('https://dotnetcli.blob.core.windows.net/dotnet/release-metadata/releases-index.json') as resp:
    channels = json.load(resp)['releases-index']
stable_channels = [
    channel for channel in channels
    if channel.get('support-phase') not in {'eol', 'preview'} and channel.get('latest-sdk')
]
if not stable_channels:
    raise SystemExit('no supported stable .NET channel found')
def channel_key(channel):
    return tuple(int(part) for part in channel['channel-version'].split('.'))
latest = max(stable_channels, key=channel_key)
print(latest['latest-sdk'])
PY

RUN set -eu; \
    DOTNET_VERSION="$(cat /tmp/dotnet-version)"; \
    rm -f /tmp/dotnet-version; \
    curl -sSfL https://dot.net/v1/dotnet-install.sh -o /tmp/dotnet-install.sh; \
    chmod +x /tmp/dotnet-install.sh; \
    /tmp/dotnet-install.sh --version "${DOTNET_VERSION}" --install-dir /usr/local/dotnet --no-path; \
    ln -sf /usr/local/dotnet/dotnet /usr/local/bin/dotnet; \
    rm -f /tmp/dotnet-install.sh
ENV DOTNET_ROOT="/usr/local/dotnet"
ENV PATH="/usr/local/dotnet:${PATH}"

RUN curl https://sh.rustup.rs -sSf | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"

RUN curl -sSfL https://taskfile.dev/install.sh | sh -s -- -d -b /usr/local/bin

RUN curl -fsSL https://mise.jdx.dev/install.sh | env MISE_INSTALL_PATH=/usr/local/bin/mise sh

RUN git clone --depth=1 https://github.com/PeonPing/peon-ping.git /tmp/peon-ping \
    && CLAUDE_CONFIG_DIR=/usr/local/share/claude REMOTE_CONTAINERS=true bash /tmp/peon-ping/install.sh --no-rc \
    && python3 -c 'import json; from pathlib import Path; config_path = Path("/usr/local/share/claude/hooks/peon-ping/config.json"); config = json.loads(config_path.read_text(encoding="utf-8")); config["default_pack"] = "peasant"; "active_pack" in config and config.__setitem__("active_pack", "peasant"); config_path.write_text(json.dumps(config, indent=2) + "\n", encoding="utf-8")' \
    && ln -sf /usr/local/share/claude/hooks/peon-ping/peon.sh /usr/local/bin/peon \
    && ln -sf /usr/local/share/claude/hooks/peon-ping/relay.sh /usr/local/bin/peon-relay \
    && rm -rf /tmp/peon-ping

RUN npm install -g @openai/codex

COPY codexbox-launch.sh /usr/local/bin/codexbox-launch
RUN chmod +x /usr/local/bin/codexbox-launch

WORKDIR /workspace
CMD ["sleep", "infinity"]
