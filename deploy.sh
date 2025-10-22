#!/usr/bin/env bash
set -euo pipefail

APP_DIR="/opt/comp3007/server"
SITE_DIR="/opt/comp3007/site"

cd "$SITE_DIR"
# Ensure we’re on master and clean
git fetch --all --prune
git checkout master
git reset --hard origin/master

cd "$APP_DIR"
# Ensure we’re on master and clean
git fetch --all --prune
git checkout master
git reset --hard origin/master

go build .

# Health check binary exists
test -x server

# Restart service
sudo systemctl restart comp3007
sudo systemctl status --no-pager comp3007 || true
