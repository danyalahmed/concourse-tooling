FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY go.work ./
COPY sdk/go.mod ./sdk/
COPY github-app-token-resource/go.mod github-app-token-resource/go.sum ./github-app-token-resource/
RUN go mod download
COPY sdk/ ./sdk/
COPY github-app-token-resource/ ./github-app-token-resource/
RUN CGO_ENABLED=0 go build -o /build/resource ./github-app-token-resource/cmd/resource/

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /build/resource /usr/local/bin/resource
RUN mkdir -p /opt/resource \
	&& ln -s /usr/local/bin/resource /opt/resource/check \
	&& ln -s /usr/local/bin/resource /opt/resource/in \
	&& ln -s /usr/local/bin/resource /opt/resource/out
