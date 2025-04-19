-- deploy/compose/postgres/init/01-init-db.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Auth schema
CREATE SCHEMA IF NOT EXISTS auth;

-- User schema
CREATE SCHEMA IF NOT EXISTS users;

-- Admin schema
CREATE SCHEMA IF NOT EXISTS admin;

-- Shared schema
CREATE SCHEMA IF NOT EXISTS shared;

-- Create roles table in the auth schema
CREATE TABLE IF NOT EXISTS auth.roles (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL UNIQUE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert only admin and user roles
INSERT INTO auth.roles (name)
VALUES ('admin'), ('user')
ON CONFLICT (name) DO NOTHING;

-- Create initial admin user (you might want to add a more secure way to do this)
CREATE TABLE IF NOT EXISTS auth.users (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  username VARCHAR(50) NOT NULL UNIQUE,
  email VARCHAR(255) NOT NULL UNIQUE,
  password VARCHAR(255) NOT NULL,
  role_id INT REFERENCES auth.roles(id),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert a default admin user (password should be properly hashed in production)
INSERT INTO auth.users (username, email, password, role_id)
SELECT 'admin', 'admin@example.com', 'hashed_password', r.id
FROM auth.roles r WHERE r.name = 'admin'
ON CONFLICT (username) DO NOTHING;