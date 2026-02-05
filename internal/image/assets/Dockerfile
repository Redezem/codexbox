FROM fedora:41

RUN dnf -y update && dnf -y install \
    git curl wget ca-certificates \
    gcc gcc-c++ clang lld make cmake ninja-build pkgconf-pkg-config gdb \
    python3 python3-pip python3-virtualenv \
    nodejs npm \
    go \
    && dnf clean all

RUN curl https://sh.rustup.rs -sSf | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"

RUN npm install -g @openai/codex

WORKDIR /workspace
CMD ["sleep", "infinity"]
