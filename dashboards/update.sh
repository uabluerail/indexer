#!/bin/sh

set -e

cd "$(dirname "$0")"

. ../.env

: ${DASHBOARD_NAME:=indexer}
: ${DASHBOARD_UID:="$(jq -r .uid "${DASHBOARD_NAME}.json")"}

if ! curl -X HEAD -s --fail-with-body "${GRAFANA_URL}/api/dashboards/uid/${DASHBOARD_UID}"; then
  echo "Dashboard with UID ${DASHBOARD_UID} is not found. Please import $(dirname "$0")/${DASHBOARD_NAME}.json once, and later use this command again to update it." >&2
  exit 1
fi

CUR_DASHBOARD="$(mktemp -t "${DASHBOARD_NAME}.json.XXXXXXX")"
curl -s --fail-with-body "${GRAFANA_URL}/api/dashboards/uid/${DASHBOARD_UID}" > "${CUR_DASHBOARD}"

jq --slurpfile current "${CUR_DASHBOARD}" \
  -f update.jq "${DASHBOARD_NAME}.json" \
  | curl --json @- -s --fail-with-body "${GRAFANA_URL}/api/dashboards/db"

rm "${CUR_DASHBOARD}"
