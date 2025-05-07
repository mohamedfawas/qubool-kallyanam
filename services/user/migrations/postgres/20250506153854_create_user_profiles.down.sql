DROP INDEX IF EXISTS idx_user_profiles_user_id;

-- 2. Drop the table
DROP TABLE IF EXISTS user_profiles;

-- 3. Drop ENUM types
DROP TYPE IF EXISTS community_enum;
DROP TYPE IF EXISTS marital_status_enum;
DROP TYPE IF EXISTS profession_enum;
DROP TYPE IF EXISTS profession_type_enum;
DROP TYPE IF EXISTS education_level_enum;
DROP TYPE IF EXISTS home_district_enum;