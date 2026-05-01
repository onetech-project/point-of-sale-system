FROM mcr.microsoft.com/devcontainers/go:1.23-bookworm

# Install Node.js LTS
RUN curl -fsSL https://deb.nodesource.com/setup_lts.x | bash - && \
    apt-get install -y nodejs

# Install additional development tools
RUN apt-get update && apt-get install -y \
    postgresql-client \
    redis-tools \
    netcat-traditional \
    curl \
    wget \
    jq \
    git-lfs \
    make \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install Go tools
RUN go install github.com/cosmtrek/air@latest && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Install Node global tools
RUN npm install -g \
    pnpm \
    yarn \
    tsx \
    @types/node

# Create non-root user for vscode
RUN useradd -m -s /bin/bash vscode && \
    usermod -aG docker vscode

# Set working directory
WORKDIR /workspace

USER vscode
