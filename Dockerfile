# syntax=docker/dockerfile:1

FROM alpine:latest

RUN <<SCRIPT
apk update
apk add --no-cache \
        git \
        nano \
        openssh \
        ca-certificates
update-ca-certificates
SCRIPT

COPY glab_*.apk /tmp/
RUN apk add --allow-untrusted /tmp/glab_*.apk

CMD ["glab"]
