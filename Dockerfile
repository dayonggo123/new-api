# 阶段1：构建前端
FROM node:22-slim AS webbuilder
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm config set registry https://registry.npmmirror.com && npm install --legacy-peer-deps
COPY web/ ./
ENV NODE_OPTIONS="--max-old-space-size=4096"
RUN npm run build

FROM golang:1.26.1-alpine@sha256:2389ebfa5b7f43eeafbd6be0c3700cc46690ef842ad962f6c5bd6be49ed82039 AS builder2
ENV GO111MODULE=on CGO_ENABLED=0

ARG TARGETOS
ARG TARGETARCH
ENV GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64}
ENV GOEXPERIMENT=greenteagc

WORKDIR /build

ADD go.mod go.sum ./
RUN go mod download

ARG CACHEBUST=1
COPY . .
# Explicitly copy files that are being incorrectly cached by Docker
COPY main.go ./main.go
COPY controller/misc.go ./controller/misc.go
COPY router/web-router.go ./router/web-router.go
COPY --from=webbuilder /web/dist ./web/dist
RUN go build -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'" -o new-api

FROM debian:bookworm-slim@sha256:f06537653ac770703bc45b4b113475bd402f451e85223f0f2837acbf89ab020a

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata libasan8 wget \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

COPY --from=builder2 /build/new-api /usr/local/bin/new-api
RUN mkdir -p /new-api/uploads && chmod 755 /usr/local/bin/new-api
EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/usr/local/bin/new-api"]
