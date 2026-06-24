# Docker Deployment

## Current release surface

This repository has one current deployment lane and one current source-build lane:

- `compose.yaml`: pull and run the published image
- `compose.source.yaml`: build from the current checkout for local verification
- `Dockerfile.release`: release image build definition
- `.github/workflows/release-image.yml`: release workflow that runs frontend lint, frontend build, `go test ./...`, then builds and pushes GHCR images

Do not follow older `latest-lite`, upstream `allanpk716/ChineseSubFinder`, or legacy release-Dockerfile guides for this fork.

## Published image

The default published image is:

```text
ghcr.io/morningstar-ski/chinesesubfinder-optimization:latest
```

`compose.yaml` starts that image directly.

## Run the published image

From the repository root:

```bash
docker compose pull
docker compose up -d
```

Exposed ports:

- `19035`: Web UI and backend API
- `19037`: local static poster/file endpoint used by the UI

Mounted paths:

- `./config:/config`
- `./media:/media`
- `./browser:/root/.cache/rod/browser`

Runtime environment variables in `compose.yaml`:

- `PUID`
- `PGID`
- `PERMS`
- `TZ`
- `UMASK`

## Build from source locally

Use `compose.source.yaml` when validating the current checkout before release:

```bash
docker compose -f compose.source.yaml up -d --build
```

Optional build arguments can be passed through the shell:

```bash
APP_VERSION=dev \
GOPROXY=https://goproxy.cn,direct \
HTTP_PROXY=http://127.0.0.1:7890 \
HTTPS_PROXY=http://127.0.0.1:7890 \
docker compose -f compose.source.yaml up -d --build
```

`compose.source.yaml` uses:

- `Dockerfile` for the local source build
- `CSF_MOVIES_SOURCE` -> `/media/movies`
- `CSF_SERIES_SOURCE` -> `/media/series`
- `./browser:/root/.cache/rod/browser`

## Media and config rules

- Keep runtime settings in `./config/ChineseSubFinderSettings.json`.
- Do not keep a second `ChineseSubFinderSettings.json` in the workspace root.
- Keep media mounts under `/media` inside the container.
- `PERMS=true` will recursively change ownership for `/media`; use it only when that matches the host setup.

## Health and verification

After startup, verify:

```bash
curl http://127.0.0.1:19035/system-status
```

For local pre-delivery verification inside this repository, use the canonical audit entry:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/local_delivery_audit.ps1
```

## Notes

- The full release image intentionally includes browser, OCR, Python, and subtitle-processing dependencies needed by this fork.
- `config/`, `browser/`, `media/`, logs, and caches are runtime state and should not be treated as release artifacts.
