# GHCR deployment

Default deployment is now pull-only:

```bash
docker compose pull
docker compose up -d
```

Default image:

```text
ghcr.io/morningstar-ski/chinesesubfinder-optimization:latest
```

To pin a release tag:

```bash
CSF_IMAGE=ghcr.io/morningstar-ski/chinesesubfinder-optimization:v0.55.4-provider.10 docker compose up -d
```

Local source build remains available with the override file:

```bash
docker compose -f compose.yaml -f compose.build.yaml up -d --build
```

Notes:

- `compose.yaml` is for remote hosts that should only pull images.
- `compose.build.yaml` is for local development or one-off image builds.
- The GitHub Actions workflow `.github/workflows/docker-publish.yml` publishes `latest`, tag, and `sha-*` images to GHCR on tag push or manual dispatch.
