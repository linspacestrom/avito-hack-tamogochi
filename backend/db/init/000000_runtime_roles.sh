#!/bin/sh
set -eu

: "${APP_RUNTIME_PASSWORD:?APP_RUNTIME_PASSWORD must be set}"
: "${APP_PROJECTOR_PASSWORD:?APP_PROJECTOR_PASSWORD must be set}"

psql \
    --set ON_ERROR_STOP=1 \
    --set runtime_password="$APP_RUNTIME_PASSWORD" \
    --set projector_password="$APP_PROJECTOR_PASSWORD" \
    --username "$POSTGRES_USER" \
    --dbname "$POSTGRES_DB" <<-'EOSQL'
CREATE ROLE app_owner
    NOLOGIN
    NOSUPERUSER
    NOCREATEDB
    NOCREATEROLE;

CREATE ROLE app_runtime
    LOGIN
    PASSWORD :'runtime_password'
    NOSUPERUSER
    NOCREATEDB
    NOCREATEROLE
    NOINHERIT
    CONNECTION LIMIT 30;

CREATE ROLE app_projector
    LOGIN
    PASSWORD :'projector_password'
    NOSUPERUSER
    NOCREATEDB
    NOCREATEROLE
    NOINHERIT
    CONNECTION LIMIT 5;
EOSQL
