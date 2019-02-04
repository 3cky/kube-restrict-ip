### SETUP STAGE

ARG PKG="github.com/3cky/kube-restrict-ip"


### BUILD STAGE

FROM golang:1.11-alpine

ARG PKG
ENV PKG=$PKG

RUN set -x \
  && apk --no-cache --update add \
    make \
    git

COPY . /$PKG

RUN set -x \
  && cd /$PKG \
  && make build-static


### PACKAGE STAGE

FROM gcr.io/google-containers/debian-iptables-amd64:v11.0.1

ARG PKG
ENV PKG=$PKG

ENTRYPOINT ["/kube-restrict-ip"]

COPY --from=0 /$PKG/bin/kube-restrict-ip /kube-restrict-ip
