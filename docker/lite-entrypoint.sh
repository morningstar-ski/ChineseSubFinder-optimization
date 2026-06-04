#!/usr/bin/env bash

set -euo pipefail

if command -v gosu >/dev/null 2>&1; then
    run_as_user() {
        gosu "$@"
    }
elif command -v su-exec >/dev/null 2>&1; then
    run_as_user() {
        su-exec "$@"
    }
else
    echo "missing gosu or su-exec, cannot switch to requested PUID/PGID" >&2
    exit 1
fi

skip_recursive_chown=false
if [[ "${PUID}:${PGID}" == "0:0" ]]; then
    skip_recursive_chown=true
fi

if [[ "${skip_recursive_chown}" == true ]]; then
    echo "PUID/PGID=0:0, skip recursive ownership reset for /config and /media"
else
    chown -R "${PUID}:${PGID}" /config
    if [[ ${PERMS} == true ]]; then
        echo "PERMS=true, reset ownership for /media to ${PUID}:${PGID}..."
        chown -R "${PUID}:${PGID}" /media
    fi
fi

if [[ -d /app/cache ]]; then
    echo "Detected /app/cache mount, ensure /config/cache points to it"
    if [[ "${skip_recursive_chown}" != true ]]; then
        chown -R "${PUID}:${PGID}" /app
    fi
    if [[ -L /config/cache && $(readlink -f /config/cache) != /app/cache ]]; then
        rm -rf /config/cache &>/dev/null
    fi
    if [[ ! -e /config/cache ]]; then
        run_as_user "${PUID}:${PGID}" ln -sf /app/cache /config/cache
    fi
else
    if [[ -L /config/cache ]]; then
        echo "Detected stale /config/cache symlink to missing /app/cache, removing it"
        rm -rf /config/cache &>/dev/null
    fi
fi

umask "${UMASK:-022}"
cd /config
run_as_user "${PUID}:${PGID}" chinesesubfinder
