FROM docker.io/golang:1.26-alpine

# Install necessary packages
RUN apk update && apk add --no-cache \
    bash \
    git \
    curl \
    openssh-client \
    procps \
    && rm -rf /var/lib/apk/lists/*

RUN adduser -D -s /bin/bash developer
USER developer
WORKDIR /workspaces