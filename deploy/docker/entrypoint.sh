#!/bin/sh
set -e

echo "Migrating database for service: $SERVICE_NAME"

# Create database if it doesn't exist
echo "Creating database $DB_NAME if it doesn't exist..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "CREATE DATABASE $DB_NAME;" || echo "Database already exists"

# Create service-specific user if it doesn't exist
echo "Creating user $SERVICE_USER if it doesn't exist..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "CREATE USER $SERVICE_USER WITH ENCRYPTED PASSWORD '$SERVICE_PASSWORD';" || echo "User already exists"

# Grant privileges
echo "Granting privileges on $DB_NAME to $SERVICE_USER..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $SERVICE_USER;"

# Connect to the specific database to set schema permissions
# Grant permissions on the public schema
echo "Granting permissions on public schema to $SERVICE_USER..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "GRANT ALL ON SCHEMA public TO $SERVICE_USER;"
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $SERVICE_USER;"
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $SERVICE_USER;"

# Set default privileges
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "ALTER DEFAULT PRIVILEGES GRANT ALL ON TABLES TO $SERVICE_USER;"
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "ALTER DEFAULT PRIVILEGES GRANT ALL ON SEQUENCES TO $SERVICE_USER;"

# Run migrations
echo "Running migrations..."
migrate -path /migrations -database "postgres://$SERVICE_USER:$SERVICE_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable" up

echo "Migrations completed successfully!"