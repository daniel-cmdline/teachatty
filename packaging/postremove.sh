#!/bin/sh
# teachatty-server package: forget the removed unit. The teachatty user and
# /var/lib/teachatty (certificate, fingerprint) are kept, so reinstalling keeps the same identity.
set -e
if [ -d /run/systemd/system ]; then
    systemctl daemon-reload >/dev/null 2>&1 || true
fi
