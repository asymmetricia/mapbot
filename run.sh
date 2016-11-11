#!/usr/bin/env bash

/mapbot \
  --db-host=$DB_HOST \
  --db-user=$DB_USER \
  --db-name=$DB_NAME \
  --db-pass=$DB_PASS \
  --slack-client-id=$CLIENT_ID \
  --slack-client-secret=$CLIENT_SECRET \
  --slack-domain=$FQDN \
  --slack-oauth-port=$PORT
