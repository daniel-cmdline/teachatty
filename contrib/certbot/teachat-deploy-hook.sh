#!/bin/sh
# certbot deploy hook for teachat-server.
#
# Let's Encrypt's privkey.pem is readable only by root, and teachat-server runs as its
# own user. certbot runs this hook as root after every successful issuance or renewal,
# with RENEWED_LINEAGE set to /etc/letsencrypt/live/<domain>. It copies the certificate
# and key where the teachat user can read them. The server notices the new files and
# reloads them by itself: no restart, nobody gets disconnected.
#
# Use it once when getting the certificate; certbot remembers it for renewals:
#   sudo certbot certonly --standalone -d chat.example.com \
#     --deploy-hook /usr/local/lib/teachat/teachat-deploy-hook.sh
set -eu

user=${TEACHAT_USER:-teachat}
dest=${TEACHAT_TLS_DIR:-/var/lib/teachat/tls}
: "${RENEWED_LINEAGE:?run this as a certbot --deploy-hook (RENEWED_LINEAGE is not set)}"

install -d -m 0750 -o "$user" -g "$user" "$dest"
# the key first, then the certificate: if the server looks in between, the pair doesn't
# match yet, the load fails, and it keeps serving the previous certificate until both
# files are in place
install -m 0600 -o "$user" -g "$user" "$RENEWED_LINEAGE/privkey.pem" "$dest/privkey.pem"
install -m 0644 -o "$user" -g "$user" "$RENEWED_LINEAGE/fullchain.pem" "$dest/fullchain.pem"
echo "teachat: copied the certificate for $(basename "$RENEWED_LINEAGE") to $dest"
