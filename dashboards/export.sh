#!/bin/sh

set -e

cd "$(dirname "$0")"

. ../.env

: ${DASHBOARD_NAME:=indexer}
: ${DASHBOARD_UID:="$(jq -r .uid "${DASHBOARD_NAME}.json")"}


curl -s --fail-with-body "${GRAFANA_URL}/api/dashboards/uid/${DASHBOARD_UID}" | jq --sort-keys -f export.jq > "${DASHBOARD_NAME}.json"
