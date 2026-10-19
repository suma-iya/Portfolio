#!/usr/bin/env bash
# Build the portfolio on your machine and ship it to the EC2 instance.
# Use the same command for the first deploy and every update:
#
#   ./deploy/deploy.sh
#
# Settings come from deploy/deploy.env (copy deploy/deploy.env.example).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
if [[ -f "$ROOT/deploy/deploy.env" ]]; then
  # shellcheck disable=SC1091
  source "$ROOT/deploy/deploy.env"
fi

: "${EC2_HOST:?Set EC2_HOST in deploy/deploy.env (see deploy/deploy.env.example)}"
EC2_USER="${EC2_USER:-ubuntu}"
SSH_KEY="${SSH_KEY:-}"
SSH_PORT="${SSH_PORT:-22}"
DOMAINS="${DOMAINS:-}"
LETSENCRYPT_EMAIL="${LETSENCRYPT_EMAIL:-}"
REMOTE="$EC2_USER@$EC2_HOST"
REMOTE_DIR=portfolio-release   # relative to the remote user's home

SSH_OPTS=(-p "$SSH_PORT" -o StrictHostKeyChecking=accept-new -o ConnectTimeout=10)
[[ -n "$SSH_KEY" ]] && SSH_OPTS+=(-i "$SSH_KEY")

log() { printf '\033[1;32m==> %s\033[0m\n' "$*"; }

command -v go >/dev/null || { echo "Go is not installed (https://go.dev/dl/)" >&2; exit 1; }

log "Connecting to $REMOTE"
case "$(ssh "${SSH_OPTS[@]}" "$REMOTE" uname -m)" in
  x86_64)  GOARCH=amd64 ;;   # t2.micro / t3.micro
  aarch64) GOARCH=arm64 ;;   # t4g.micro
  *) echo "Unsupported server architecture" >&2; exit 1 ;;
esac

BUILD_DIR="$(mktemp -d)"
trap 'rm -rf "$BUILD_DIR"' EXIT

log "Building Go backend for linux/$GOARCH"
(cd "$ROOT/backend" && CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" \
  go build -trimpath -ldflags="-s -w" -o "$BUILD_DIR/portfolio-backend" ./cmd/server)

cp "$ROOT"/nginx/*.conf \
   "$ROOT/deploy/portfolio-api.service" \
   "$ROOT/deploy/server-setup.sh" \
   "$ROOT/deploy/env.production.example" \
   "$BUILD_DIR/"
cp -R "$ROOT/frontend" "$BUILD_DIR/frontend"

log "Uploading release"
# tar over ssh: works with macOS' built-in tools, nothing extra to install.
COPYFILE_DISABLE=1 tar -C "$BUILD_DIR" --exclude .DS_Store -czf - . \
  | ssh "${SSH_OPTS[@]}" "$REMOTE" \
      "rm -rf ~/$REMOTE_DIR && mkdir -p ~/$REMOTE_DIR && tar -xzf - --warning=no-unknown-keyword -C ~/$REMOTE_DIR"

log "Installing on server"
ssh -t "${SSH_OPTS[@]}" "$REMOTE" \
  "sudo env DOMAINS=$(printf %q "$DOMAINS") LETSENCRYPT_EMAIL=$(printf %q "$LETSENCRYPT_EMAIL") \
   bash ~/$REMOTE_DIR/server-setup.sh ~/$REMOTE_DIR"

if [[ -n "$DOMAINS" ]]; then
  URL="https://${DOMAINS%% *}"
else
  URL="http://$EC2_HOST"
fi
log "Checking $URL/health"
# Retries cover the moment right after an nginx reload, when old workers may
# still answer with the previous config.
if curl -fsS --max-time 10 --retry 4 --retry-delay 2 --retry-all-errors "$URL/health"; then
  echo
  log "Live at $URL"
else
  echo
  echo "Health check failed from here. If the server output above shows a warning, fix that first." >&2
  exit 1
fi
