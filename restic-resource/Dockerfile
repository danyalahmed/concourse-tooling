FROM golang:1.26.4-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git
COPY sdk/ /sdk/
COPY restic-resource/ /src/

ARG TARGET_CMD
RUN go build -o /resource/res ./cmd/${TARGET_CMD}

FROM alpine:3.18
RUN apk add --no-cache restic sshfs cifs-utils fuse
RUN sed -i 's/#user_allow_other/user_allow_other/g' /etc/fuse.conf

COPY --from=builder /resource/res /opt/resource/check
RUN ln -s /opt/resource/check /opt/resource/in && \
    ln -s /opt/resource/check /opt/resource/out
