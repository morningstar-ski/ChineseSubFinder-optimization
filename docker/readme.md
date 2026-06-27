# Docker Deployment

## Canonical end-user deployment

End users should use exactly one deployment entry:

- `docker-compose.yaml`

`compose.yaml` is kept only as a compatibility alias for tools that auto-discover the default compose filename. Treat `compose.source.yaml`, `compose.browser.yaml`, and `compose.fnos.yaml` as development or validation helpers, not end-user deployment files.

## Published image

Default image:

```text
ghcr.io/morningstar-ski/chinesesubfinder-optimization:latest
```

## Deployment model

The verified deployment model is:

1. Copy `.env.example` to `.env`
2. Set host paths in `.env`
3. Validate the rendered compose
4. Pull the published image
5. Start `docker-compose.yaml`
6. Verify `/system-status`

The same variable model is intended to work on Windows, Linux, and NAS hosts:

- `CSF_CONTAINER_NAME`
- `CSF_HOSTNAME`
- `CSF_CONFIG_DIR` -> `/config`
- `CSF_MEDIA_DIR` -> `/media`
- `CSF_BROWSER_DIR` -> `/root/.cache/rod/browser`
- `CSF_WEB_PORT` -> `19035`
- `CSF_STATIC_PORT` -> `19037`
- `PUID`, `PGID`, `PERMS`, `TZ`, `UMASK`

## Quick start

From the repository root:

```bash
cp .env.example .env
docker compose -f docker-compose.yaml config
docker compose -f docker-compose.yaml pull
docker compose -f docker-compose.yaml up -d
```

Then verify:

```bash
curl http://127.0.0.1:19035/system-status
```

Expected healthy response:

```json
{"is_setup":true,"is_running_in_docker":true}
```

## Path rules

- `CSF_CONFIG_DIR` should point to a persistent host directory that stores `ChineseSubFinderSettings.json`.
- `CSF_MEDIA_DIR` should point to the media library root that contains the movie and series folders referenced by your runtime settings.
- `CSF_BROWSER_DIR` should point to a persistent browser cache directory.
- `CSF_CONTAINER_NAME` should stay unique on the same Docker host.
- Keep media mounted under `/media` inside the container.
- `PERMS=true` will recursively change ownership for `/media`; use it only when that matches the host setup.

## Example host values

Windows example:

```dotenv
CSF_CONFIG_DIR=./config
CSF_MEDIA_DIR=./media
CSF_BROWSER_DIR=./browser
CSF_CONTAINER_NAME=chinesesubfinder
CSF_HOSTNAME=chinesesubfinder
```

FnOS / Linux example:

```dotenv
CSF_CONFIG_DIR=./config
CSF_MEDIA_DIR=/vol2/1000/video/link
CSF_BROWSER_DIR=./browser
CSF_CONTAINER_NAME=chinesesubfinder
CSF_HOSTNAME=chinesesubfinder
PUID=999
PGID=901
PERMS=false
```

## Development-only compose files

- `compose.source.yaml`: build the current checkout locally before release
- `compose.browser.yaml`: local browser-specific validation overlay
- `compose.fnos.yaml`: local FnOS bridge overlay for developer verification

These files are not the supported end-user entrypoint.

## Source-build verification

When validating the current checkout before release:

```bash
docker compose -f compose.source.yaml up -d --build
```

## Notes

- The full release image intentionally includes browser, OCR, Python, and subtitle-processing dependencies needed by this fork.
- `config/`, `browser/`, `media/`, logs, and caches are runtime state and should not be treated as release artifacts.
