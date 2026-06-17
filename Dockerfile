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
ARG LITE_MODE=true
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
      -ldflags="-s -w -X main.AppVersion=${APP_VERSION} -X main.LiteMode=${LITE_MODE}" \
      -o /out/chinesesubfinder \
      ./cmd/chinesesubfinder

FROM ${RUNTIME_IMAGE}
ARG INSTALL_BROWSER=false
ENV TZ=Asia/Shanghai \
    PERMS=true \
    PUID=1026 \
    PGID=100 \
    CSF_DDDDOCR_PYTHON=/opt/csf-ocr/bin/python3 \
    CSF_LLM_SUBTITLE_FALLBACK_PYTHON=/opt/csf-ocr/bin/python3 \
    CSF_LLM_SUBTITLE_FALLBACK_SUBFLOW_ROOT=/opt/subflow \
    UMASK=022 \
    PS1="\u@\h:\w \$ "
COPY third_party/subflow /opt/subflow
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash \
        ca-certificates \
        ffmpeg \
        gosu \
        python3 \
        python3-pip \
        python3-venv \
        tini \
        tzdata \
    && if [ "${INSTALL_BROWSER}" = "true" ]; then apt-get install -y --no-install-recommends \
        chromium \
        fonts-liberation \
        libasound2 \
        libatk-bridge2.0-0 \
        libatk1.0-0 \
        libcups2 \
        libdbus-1-3 \
        libdrm2 \
        libgbm1 \
        libgtk-3-0 \
        libnspr4 \
        libnss3 \
        libu2f-udev \
        libvulkan1 \
        libx11-xcb1 \
        libxcomposite1 \
        libxdamage1 \
        libxfixes3 \
        libxkbcommon0 \
        libxrandr2 \
        xdg-utils; fi \
    && python3 -m venv /opt/csf-ocr \
    && /opt/csf-ocr/bin/pip install --no-cache-dir ddddocr -r /opt/subflow/requirements-translate.txt \
    && ln -snf /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo "${TZ}" > /etc/timezone \
    && rm -rf /var/lib/apt/lists/*
COPY --from=backend-builder /out/chinesesubfinder /usr/bin/chinesesubfinder
COPY docker/lite-entrypoint.sh /usr/bin/entrypoint.sh
RUN sed -i 's/\r$//' /usr/bin/entrypoint.sh \
    && chmod +x /usr/bin/chinesesubfinder /usr/bin/entrypoint.sh
VOLUME ["/config", "/media"]
WORKDIR /config
EXPOSE 19035 19037
ENTRYPOINT ["tini", "--", "/usr/bin/entrypoint.sh"]
