#!/usr/bin/env bash

/mapbot \
  --db-host="${DB_HOST:-$MAPBOT_DB_PORT_5432_TCP_ADDR}" \
  --db-user="$DB_USER" \
  --db-name="$DB_NAME" \
  --db-pass="$DB_PASS" \
  --db-ssl="${DB_SSL:-true}" \
  --slack-client-id="$CLIENT_ID" \
  --slack-client-secret="$CLIENT_SECRET" \
  --slack-verification-token="$VERIFICATION_TOKEN" \
  --domain="$FQDN" \
  ${TLS:+--tls} \
  --port="$PORT"
