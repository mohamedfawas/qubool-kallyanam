-- Create all the necessary databases
CREATE DATABASE auth;
CREATE DATABASE admin;
CREATE DATABASE users;

-- Create a shared role for all services if needed
CREATE ROLE service_user WITH LOGIN PASSWORD 'service_password';

-- Set privileges for each database
GRANT ALL PRIVILEGES ON DATABASE auth TO service_user;
GRANT ALL PRIVILEGES ON DATABASE admin TO service_user;
GRANT ALL PRIVILEGES ON DATABASE users TO service_user;

-- For auth database-specific setup
\c auth
CREATE SCHEMA IF NOT EXISTS auth;
-- Create any auth-specific tables, functions, etc.

-- For admin database-specific setup
\c admin
CREATE SCHEMA IF NOT EXISTS admin;
-- Create any admin-specific tables, functions, etc.

-- For users database-specific setup
\c users
CREATE SCHEMA IF NOT EXISTS users;
-- Create any users-specific tables, functions, etc.