#!/usr/bin/env bash

#

go build .

# Health check binary exists
test -x server

# Restart service
sudo systemctl restart comp3007
sudo systemctl status --no-pager comp3007 || true
