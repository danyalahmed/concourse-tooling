FROM golang:1.26.3-alpine AS builder
WORKDIR /src
COPY sdk/ /sdk/
COPY restic-resource/ /src/

ARG TARGET_CMD
RUN go build -o /resource/out ./cmd/${TARGET_CMD} && \
    cp /resource/out /resource/in && \
    cp /resource/out /resource/check

FROM alpine:3.18
RUN apk add --no-cache restic sshfs cifs-utils fuse
# Allow non-privileged use of fuse if needed (though privileged mode is used)
RUN sed -i 's/#user_allow_other/user_allow_other/g' /etc/fuse.conf

COPY --from=builder /resource/ /opt/resource/
