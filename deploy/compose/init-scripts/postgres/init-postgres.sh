#!/bin/bash
# deploy/compose/init-scripts/postgres/init-postgres.sh
set -e

# Create databases for each service
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE auth;
    CREATE DATABASE users;
    CREATE DATABASE admin;
EOSQL

# Grant privileges
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "auth" <<-EOSQL
    CREATE SCHEMA IF NOT EXISTS auth;
EOSQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "users" <<-EOSQL
    CREATE SCHEMA IF NOT EXISTS user_schema;
EOSQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "admin" <<-EOSQL
    CREATE SCHEMA IF NOT EXISTS admin;
EOSQL