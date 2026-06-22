#!/bin/bash

MAPCTF_USERNAME="${MAPCTF_USERNAME:=admin}"
MAPCTF_PASSWORD="${MAPCTF_PASSWORD:=admin}"
MAP_UUID="${MAP_UUID:=local-dev}"
WAIT="${WAIT:=5}"

# Wait until all services operational
until /opt/mapctf/bin/mapctf-api config-check
do
  echo "Backend is not ready"
  sleep "$WAIT"
done

# Reset password for admin user
/opt/mapctf/bin/mapctf-api create-admin-user -u "${MAPCTF_USERNAME}" -p "${MAPCTF_PASSWORD}" -U "${MAP_UUID}" || \
  echo "Failed to create admin user; keeping container running for debugging"

exec sleep infinity
