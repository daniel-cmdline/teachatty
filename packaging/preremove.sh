#!/bin/sh
# teachatty-server package: stop the service on removal, but not on upgrade.
# deb passes "remove" (or "upgrade"); rpm passes the number of versions left (0 = removal).
set -e
case "$1" in
    remove|purge|0)
        if [ -d /run/systemd/system ]; then
            systemctl disable --now teachatty-server >/dev/null 2>&1 || true
        fi
        ;;
esac
