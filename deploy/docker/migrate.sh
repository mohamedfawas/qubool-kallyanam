#!/bin/sh
set -e

echo "Waiting for PostgreSQL to be ready..."
until PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -d "postgres" -c '\q'; do
  echo "PostgreSQL is unavailable - sleeping"
  sleep 1
done

echo "PostgreSQL is up - checking if database exists"
if ! PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -lqt | cut -d \| -f 1 | grep -qw "$DB_NAME"; then
  echo "Creating database $DB_NAME"
  PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -c "CREATE DATABASE $DB_NAME"
  
  echo "Creating database user $SERVICE_USER"
  PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -c "CREATE USER $SERVICE_USER WITH PASSWORD '$SERVICE_PASSWORD'"
  
  echo "Granting privileges to $SERVICE_USER on $DB_NAME"
  PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $SERVICE_USER"
fi

echo "Running migrations..."
cd /migrations

for f in *.sql; do
  if [ -f "$f" ]; then
    echo "Executing $f"
    PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" -f "$f"
  fi
done

echo "Migrations completed successfully!"