FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY . .
RUN go work sync && go mod download all
RUN CGO_ENABLED=0 go build -o /build/resource ./cron/cmd/resource/

FROM gcr.io/distroless/static-debian12
COPY --from=builder /build/resource /usr/local/bin/resource
COPY --from=builder /build/resource /opt/resource/check
COPY --from=builder /build/resource /opt/resource/in
COPY --from=builder /build/resource /opt/resource/out
