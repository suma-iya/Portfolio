#!/usr/bin/env bash
# Runs ON the EC2 instance, as root. deploy/deploy.sh uploads a release
# directory and calls this script; you shouldn't need to run it by hand.
#
# Safe to run on every deploy: packages, the service user and .env are only
# created the first time; the binary, site files and configs are refreshed.
#
# Usage: sudo DOMAINS="example.com" LETSENCRYPT_EMAIL="me@x.com" \
#          bash server-setup.sh /home/ubuntu/portfolio-release
set -euo pipefail

RELEASE_DIR="${1:?usage: server-setup.sh <release-dir>}"
DOMAINS="${DOMAINS:-}"
LETSENCRYPT_EMAIL="${LETSENCRYPT_EMAIL:-}"
PRIMARY_DOMAIN="${DOMAINS%% *}"

APP_DIR=/opt/portfolio-api
WEB_DIR=/var/www/portfolio
ACME_DIR=/var/www/letsencrypt
SITE_CONF=/etc/nginx/sites-available/portfolio
CERT="/etc/letsencrypt/live/${PRIMARY_DOMAIN}/fullchain.pem"

log()  { printf '\033[1;34m==> %s\033[0m\n' "$*"; }
warn() { printf '\033[1;33m!!  %s\033[0m\n' "$*"; }

[[ $EUID -eq 0 ]] || { echo "server-setup.sh must run as root" >&2; exit 1; }

# --- 1. Packages (first run only) -----------------------------------------
if ! command -v nginx >/dev/null || ! command -v certbot >/dev/null || ! command -v rsync >/dev/null; then
  log "Installing nginx, certbot, rsync"
  apt-get update -y
  DEBIAN_FRONTEND=noninteractive apt-get install -y nginx certbot rsync
fi
systemctl enable --now nginx >/dev/null

# --- 2. Service user + directories ----------------------------------------
if ! id portfolio &>/dev/null; then
  log "Creating system user 'portfolio'"
  useradd --system --no-create-home --shell /usr/sbin/nologin portfolio
fi
install -d -m 755 "$APP_DIR" "$WEB_DIR" "$ACME_DIR"

# --- 3. Backend config (first run only; never overwritten) -----------------
if [[ ! -f "$APP_DIR/.env" ]]; then
  log "Creating $APP_DIR/.env from template"
  install -m 600 -o root -g root "$RELEASE_DIR/env.production.example" "$APP_DIR/.env"
fi

# --- 4. Backend binary + systemd service ----------------------------------
log "Installing backend binary and service"
install -m 755 -o root -g root "$RELEASE_DIR/portfolio-backend" "$APP_DIR/portfolio-backend"
install -m 644 "$RELEASE_DIR/portfolio-api.service" /etc/systemd/system/portfolio-api.service
systemctl daemon-reload
systemctl enable portfolio-api >/dev/null

BACKEND_READY=1
if grep -q 'your-16-char-app-password' "$APP_DIR/.env"; then
  BACKEND_READY=0
  systemctl stop portfolio-api 2>/dev/null || true
  warn "Backend NOT started: $APP_DIR/.env still has placeholder SMTP credentials."
  warn "Fix it with:  sudo nano $APP_DIR/.env && sudo systemctl restart portfolio-api"
else
  systemctl restart portfolio-api
fi

# --- 5. Static site --------------------------------------------------------
log "Publishing frontend to $WEB_DIR"
rsync -a --delete --chown=root:root --chmod=D755,F644 "$RELEASE_DIR/frontend/" "$WEB_DIR/"

# --- 6. Nginx --------------------------------------------------------------
render_site() {
  sed -e "s|__SERVER_NAME__|${DOMAINS:-_}|g" \
      -e "s|__PRIMARY_DOMAIN__|${PRIMARY_DOMAIN}|g" \
      "$RELEASE_DIR/$1" > "$SITE_CONF"
  nginx -t -q
  systemctl reload nginx
}

log "Configuring nginx"
install -m 644 "$RELEASE_DIR/portfolio-common.conf" /etc/nginx/snippets/portfolio-common.conf
ln -sf "$SITE_CONF" /etc/nginx/sites-enabled/portfolio
rm -f /etc/nginx/sites-enabled/default

if [[ -n "$PRIMARY_DOMAIN" && -f "$CERT" ]]; then
  render_site portfolio-https.conf
else
  render_site portfolio-http.conf
fi

# --- 7. HTTPS certificate (first time a domain is configured) --------------
if [[ -n "$PRIMARY_DOMAIN" && ! -f "$CERT" ]]; then
  if [[ -z "$LETSENCRYPT_EMAIL" ]]; then
    warn "DOMAINS is set but LETSENCRYPT_EMAIL is empty; skipping HTTPS."
  else
    log "Requesting Let's Encrypt certificate for: $DOMAINS"
    domain_args=()
    for d in $DOMAINS; do domain_args+=(-d "$d"); done
    # --webroot keeps certbot from editing our nginx files; renewals run from
    # certbot's systemd timer and reload nginx via the deploy hook.
    if certbot certonly --webroot -w "$ACME_DIR" "${domain_args[@]}" \
         --email "$LETSENCRYPT_EMAIL" --agree-tos --non-interactive \
         --deploy-hook "systemctl reload nginx"; then
      render_site portfolio-https.conf
      log "HTTPS enabled"
    else
      warn "Certificate request failed. Check that DNS for $DOMAINS points at this"
      warn "server and that port 80 is open in the EC2 security group. Site stays on HTTP."
    fi
  fi
fi

# --- 8. Smoke test ---------------------------------------------------------
if [[ $BACKEND_READY -eq 1 ]]; then
  sleep 1
  if curl -fsS http://127.0.0.1:8080/health >/dev/null 2>&1; then
    log "Backend healthy"
  else
    warn "Backend health check failed. See: sudo journalctl -u portfolio-api -n 50"
  fi
fi

log "Done"
