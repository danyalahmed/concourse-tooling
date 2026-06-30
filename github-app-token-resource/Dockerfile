FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY go.work ./
COPY sdk/go.mod ./sdk/
COPY github-app-token-resource/go.mod github-app-token-resource/go.sum ./github-app-token-resource/
RUN go mod download
COPY sdk/ ./sdk/
COPY github-app-token-resource/ ./github-app-token-resource/
RUN CGO_ENABLED=0 go build -o /build/resource ./github-app-token-resource/cmd/resource/
RUN mkdir -p /build/opt/resource && \
    ln -s /usr/local/bin/resource /build/opt/resource/check && \
    ln -s /usr/local/bin/resource /build/opt/resource/in && \
    ln -s /usr/local/bin/resource /build/opt/resource/out

FROM gcr.io/distroless/static-debian12
COPY --from=builder /build/resource /usr/local/bin/resource
COPY --from=builder /build/opt/resource /opt/resource
