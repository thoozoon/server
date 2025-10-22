#!/usr/bin/env bash

# /etc/systemd/system/comp3007.service

sudo tee /etc/systemd/system/comp3007.service >/dev/null <<'UNIT'
[Unit]
Description=comp3007 web server
After=network-online.target
Wants=network-online.target

[Service]
User=comp3007
Group=comp3007
WorkingDirectory=/opt/comp3007/src
EnvironmentFile=-/opt/comp3007/.env
# Build artifact location:
ExecStart=/opt/comp3007/src/bin/comp3007
Restart=on-failure
RestartSec=2
# Hardening (adjust if you need extra capabilities, files, ports, etc.)
NoNewPrivileges=true
ProtectSystem=full
ProtectHome=true
PrivateTmp=true
AmbientCapabilities=
LockPersonality=true

[Install]
WantedBy=multi-user.target
UNIT

sudo systemctl daemon-reload
sudo systemctl enable comp3007
