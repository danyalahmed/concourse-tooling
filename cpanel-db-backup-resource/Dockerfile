FROM golang:1.26.3-alpine AS builder
WORKDIR /src
COPY sdk/ /sdk/
COPY cpanel-db-backup-resource/ /src/
RUN go build -o /resource/check ./cmd/resource && \
    cp /resource/check /resource/in && \
    cp /resource/check /resource/out

FROM alpine:3.18
RUN apk add --no-cache openssh-client mariadb-client cifs-utils
COPY --from=builder /resource/ /opt/resource/
