FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY . .
RUN go work sync && go mod download all
RUN CGO_ENABLED=0 go build -o /build/resource ./cron/cmd/resource/
RUN mkdir -p /build/opt/resource && \
    ln -s /usr/local/bin/resource /build/opt/resource/check && \
    ln -s /usr/local/bin/resource /build/opt/resource/in && \
    ln -s /usr/local/bin/resource /build/opt/resource/out

FROM gcr.io/distroless/static-debian12
COPY --from=builder /build/resource /usr/local/bin/resource
COPY --from=builder /build/opt/resource /opt/resource
