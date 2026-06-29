FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY . .
RUN go work sync && go mod download all
RUN CGO_ENABLED=0 go build -o /build/resource ./cron/cmd/resource/

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /build/resource /usr/local/bin/resource
RUN mkdir -p /opt/resource \
	&& ln -s /usr/local/bin/resource /opt/resource/check \
	&& ln -s /usr/local/bin/resource /opt/resource/in \
	&& ln -s /usr/local/bin/resource /opt/resource/out
