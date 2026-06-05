FROM ubuntu
ARG DEBIAN_FRONTEND=noninteractive
ARG TARGETARCH
ARG S6_OVERLAY_VERSION=3.2.3.0
RUN apt-get update \
    && apt-get install --no-install-recommends -y \
       ca-certificates \
       dbus-x11 \
       dumb-init \
       ffmpeg \
       fonts-liberation \
       fonts-noto-cjk \
       fonts-noto-color-emoji \
       gtk2-engines-pixbuf \
       imagemagick \
       libasound2 \
       tesseract-ocr \
       tesseract-ocr-chi-sim \
       tesseract-ocr-eng \
       libgbm1 \
       libgcc-9-dev \
       libgtk-3-0 \
       libnss3 \
       libstdc++6 \
       libxss1 \
       libxtst6 \
       tzdata \
       wget \
       x11-apps \
       xfonts-100dpi \
       xfonts-75dpi \
       xfonts-base \
       xfonts-cyrillic \
       xfonts-scalable \
       xorg \
       xz-utils \
       xvfb \
       yasm \
    && case "${TARGETARCH}" in \
         amd64) s6_arch=x86_64 ;; \
         arm64) s6_arch=aarch64 ;; \
         *) echo "unsupported TARGETARCH: ${TARGETARCH}" >&2; exit 1 ;; \
       esac \
    && wget -q -O /tmp/s6-overlay-noarch.tar.xz "https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-noarch.tar.xz" \
    && wget -q -O /tmp/s6-overlay-${s6_arch}.tar.xz "https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-${s6_arch}.tar.xz" \
    && wget -q -O /tmp/s6-overlay-symlinks-noarch.tar.xz "https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-symlinks-noarch.tar.xz" \
    && wget -q -O /tmp/s6-overlay-symlinks-arch.tar.xz "https://github.com/just-containers/s6-overlay/releases/download/v${S6_OVERLAY_VERSION}/s6-overlay-symlinks-arch.tar.xz" \
    && tar -C / -Jxpf /tmp/s6-overlay-noarch.tar.xz \
    && tar -C / -Jxpf /tmp/s6-overlay-${s6_arch}.tar.xz \
    && tar -C / -Jxpf /tmp/s6-overlay-symlinks-noarch.tar.xz \
    && tar -C / -Jxpf /tmp/s6-overlay-symlinks-arch.tar.xz \
    && apt-get clean \
    && rm -rf \
       /tmp/* \
       /var/lib/apt/lists/* \
       /var/tmp/*

