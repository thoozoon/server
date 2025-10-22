#!/usr/bin/env bash

# Create a dedicated user (optional but recommended)
sudo useradd -m -s /bin/bash comp3007 || true
sudo usermod -aG sudo comp3007

# Create app dir owned by that user
sudo mkdir -p /opt/comp3007
sudo chown -R comp3007:comp3007 /opt/comp3007

# As the comp3007 user: clone your repo
sudo -u comp3007 -H bash -lc '
  cd /opt/comp3007
  git clone --branch master --depth 1 https://github.com/thoozoon/comp3007.git src || true
  cd src
  git fetch --all --prune
'

# OPTIONAL: if your app needs env vars, put them here:
sudo -u comp3007 tee /opt/comp3007/.env >/dev/null <<'ENV'
# EXAMPLE:
PORT=8080
# Add other secrets via systemd Environment= or files readable only by root if sensitive.
ENV
