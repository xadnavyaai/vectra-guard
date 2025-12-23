# Vectra Guard: Secure Container for AI Agents
# Provides isolated execution environment with built-in monitoring

FROM golang:1.21-alpine AS builder

# Build vectra-guard
WORKDIR /build
COPY . .
RUN go build -o vectra-guard main.go

# Runtime image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    bash \
    curl \
    git \
    nodejs \
    npm \
    python3 \
    py3-pip \
    sudo \
    ca-certificates

# Create non-root user for agent
RUN adduser -D -u 1000 -s /bin/bash agent

# Install vectra-guard
COPY --from=builder /build/vectra-guard /usr/local/bin/
RUN chmod +x /usr/local/bin/vectra-guard

# Create wrapper script
COPY scripts/container-entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Setup directory structure
RUN mkdir -p /workspace /workspace-rw && \
    chown agent:agent /workspace /workspace-rw

# Configure sudo with restrictions
RUN echo 'agent ALL=(root) NOPASSWD: /usr/bin/apt-get, /usr/bin/apk' > /etc/sudoers.d/agent && \
    chmod 0440 /etc/sudoers.d/agent

# Security: Drop all capabilities by default
# Runtime will add specific ones as needed
RUN mkdir -p /.vectra-guard && chown agent:agent /.vectra-guard

# Switch to non-root user
USER agent
WORKDIR /workspace

# Environment
ENV VECTRAGUARD_CONTAINER=true
ENV PATH="/usr/local/bin:$PATH"

# Health check
HEALTHCHECK --interval=30s --timeout=3s \
    CMD vectra-guard session list || exit 1

# Default: Start daemon and drop to shell
ENTRYPOINT ["/entrypoint.sh"]
CMD ["/bin/bash"]

