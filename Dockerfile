FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

# System deps
RUN apt-get update && apt-get install -y \
    bash git curl wget ca-certificates gnupg jq pciutils \
    && rm -rf /var/lib/apt/lists/*

# Node.js 22 (LTS)
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# gh CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | \
    gpg --dearmor -o /usr/share/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    > /etc/apt/sources.list.d/github-cli.list && \
    apt-get update && apt-get install -y gh && \
    rm -rf /var/lib/apt/lists/*

# Claude Code (official installer)
RUN curl -fsSL https://claude.ai/install.sh | bash

# Gemini CLI
RUN npm install -g @google/gemini-cli

# OpenAI Codex CLI
RUN npm install -g @openai/codex

# OpenCode
RUN npm install -g opencode-ai || true

# Ollama binary — models are mounted from host ~/.ollama at runtime (no pull needed)
# Install binary directly; skip systemd service (not available in containers).
RUN ARCH=$(dpkg --print-architecture) && \
    curl -fsSL "https://github.com/ollama/ollama/releases/latest/download/ollama-linux-${ARCH}" \
    -o /usr/local/bin/ollama && chmod +x /usr/local/bin/ollama

# Trust all mounted directories for git operations
RUN git config --global --add safe.directory '*'

# Entrypoint: start ollama in background so models are ready, then exec requested command
RUN printf '#!/bin/bash\nset -e\nif command -v ollama &>/dev/null; then\n  OLLAMA_HOST=0.0.0.0 ollama serve &>/tmp/ollama.log &\n  sleep 1\nfi\nexec "$@"\n' \
    > /entrypoint.sh && chmod +x /entrypoint.sh

WORKDIR /workspace

ENTRYPOINT ["/entrypoint.sh"]
CMD ["/bin/bash"]
