FROM fedora:41

RUN dnf -y update && dnf -y install \
    git curl wget ca-certificates \
    gcc gcc-c++ clang lld make cmake ninja-build pkgconf-pkg-config gdb \
    python3 python3-pip python3-virtualenv \
    nodejs npm \
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

RUN curl https://sh.rustup.rs -sSf | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"

RUN curl -sSfL https://taskfile.dev/install.sh | sh -s -- -d -b /usr/local/bin

RUN npm install -g @openai/codex

WORKDIR /workspace
CMD ["sleep", "infinity"]
