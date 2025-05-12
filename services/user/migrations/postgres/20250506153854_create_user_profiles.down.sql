-- 1. Drop indexes on partner_preferences
DROP INDEX IF EXISTS idx_partner_preferences_user_profile_id;

-- 2. Drop partner_preferences table
DROP TABLE IF EXISTS partner_preferences;

-- 3. Drop index on user_profiles
DROP INDEX IF EXISTS idx_user_profiles_user_id;

-- 4. Drop user_profiles table
DROP TABLE IF EXISTS user_profiles;

-- 5. Drop ENUM types
DROP TYPE IF EXISTS community_enum;
DROP TYPE IF EXISTS marital_status_enum;
DROP TYPE IF EXISTS profession_enum;
DROP TYPE IF EXISTS profession_type_enum;
DROP TYPE IF EXISTS education_level_enum;
DROP TYPE IF EXISTS home_district_enum;