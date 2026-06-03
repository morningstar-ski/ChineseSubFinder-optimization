# syntax=docker/dockerfile:1.7

ARG NODE_IMAGE=node:20-bookworm-slim
ARG GO_IMAGE=golang:1.22-bookworm
ARG RUNTIME_IMAGE=debian:bookworm-slim

FROM ${NODE_IMAGE} AS frontend-builder
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ARG NPM_CONFIG_REGISTRY
ENV http_proxy=${HTTP_PROXY} \
    https_proxy=${HTTPS_PROXY} \
    no_proxy=${NO_PROXY} \
    HTTP_PROXY=${HTTP_PROXY} \
    HTTPS_PROXY=${HTTPS_PROXY} \
    NO_PROXY=${NO_PROXY} \
    NPM_CONFIG_REGISTRY=${NPM_CONFIG_REGISTRY}
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM ${GO_IMAGE} AS backend-builder
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ARG GOPROXY=https://proxy.golang.org,direct
ARG APP_VERSION=dev
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ENV http_proxy=${HTTP_PROXY} \
    https_proxy=${HTTPS_PROXY} \
    no_proxy=${NO_PROXY} \
    HTTP_PROXY=${HTTP_PROXY} \
    HTTPS_PROXY=${HTTPS_PROXY} \
    NO_PROXY=${NO_PROXY} \
    GOPROXY=${GOPROXY}
WORKDIR /src
RUN apt-get update \
    && apt-get install -y --no-install-recommends build-essential ca-certificates git pkg-config \
    && rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
RUN set -eux; \
    export GOOS="${TARGETOS:-linux}"; \
    export GOARCH="${TARGETARCH:-amd64}"; \
    if [ "${GOARCH}" = "arm" ] && [ -n "${TARGETVARIANT:-}" ]; then export GOARM="${TARGETVARIANT#v}"; fi; \
    CGO_ENABLED=1 go build \
      -ldflags="-s -w -X main.AppVersion=${APP_VERSION} -X main.LiteMode=true" \
      -o /out/chinesesubfinder \
      ./cmd/chinesesubfinder

FROM ${RUNTIME_IMAGE}
ENV TZ=Asia/Shanghai \
    PERMS=true \
    PUID=1026 \
    PGID=100 \
    UMASK=022 \
    PS1="\u@\h:\w \$ "
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash \
        ca-certificates \
        ffmpeg \
        gosu \
        tini \
        tzdata \
    && ln -snf /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo "${TZ}" > /etc/timezone \
    && rm -rf /var/lib/apt/lists/*
COPY --from=backend-builder /out/chinesesubfinder /usr/bin/chinesesubfinder
COPY docker/lite-entrypoint.sh /usr/bin/entrypoint.sh
RUN chmod +x /usr/bin/chinesesubfinder /usr/bin/entrypoint.sh
VOLUME ["/config", "/media"]
WORKDIR /config
EXPOSE 19035 19037
ENTRYPOINT ["tini", "--", "/usr/bin/entrypoint.sh"]
