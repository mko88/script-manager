#!/usr/bin/env bash
# Deploys the app.
#
# Usage: deploy.sh [options] <target> [region]
#
# Options:
#   -f, --force          Skip the confirmation
#   -e, --env ENV        Environment {dev|staging|prod} (required)
#   -t, --tag TAG...     Image tags to push
#   -n NAME              Release name, default: latest
#   --mode=MODE          Rollout mode (fast|safe)
#   -q  Quiet
#   --timeout seconds    How long to wait,
#                        in seconds
set -euo pipefail
while getopts "xy:" opt; do :; done
