#!/bin/sh
# Creates the ARI database on the same Postgres instance.
# The primary "payment" database is created automatically via POSTGRES_DB.
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    CREATE DATABASE ari;
EOSQL
