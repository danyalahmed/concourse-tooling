FROM golang:1.26.4-alpine AS builder
WORKDIR /build
COPY . .
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
