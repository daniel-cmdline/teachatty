#!/bin/sh
# teachatty-server package: create the system user and let systemd see the unit.
# The service is not started automatically: review /etc/default/teachatty-server first.
set -e
if command -v systemd-sysusers >/dev/null 2>&1; then
    systemd-sysusers /usr/lib/sysusers.d/teachatty-server.conf
elif ! getent passwd teachatty >/dev/null; then
    useradd --system --home-dir /var/lib/teachatty --no-create-home --shell /usr/sbin/nologin teachatty
fi
if [ -d /run/systemd/system ]; then
    systemctl daemon-reload >/dev/null 2>&1 || true
    # an upgrade: restart a running server so it uses the new binary
    if systemctl is-active --quiet teachatty-server; then
        systemctl restart teachatty-server || true
    else
        echo "teachatty-server installed. Start it with: sudo systemctl enable --now teachatty-server"
    fi
fi
