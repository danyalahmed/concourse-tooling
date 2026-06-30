FROM golang:1.26.3-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git
COPY sdk/ /sdk/
COPY cpanel-db-backup-resource/ /src/
RUN go build -o /resource/check ./cmd/resource

FROM alpine:3.18
RUN apk add --no-cache openssh-client mariadb-client cifs-utils
COPY --from=builder /resource/check /opt/resource/check
RUN ln -s /opt/resource/check /opt/resource/in && \
    ln -s /opt/resource/check /opt/resource/out
