-- 1. ENUM types
CREATE TYPE community_enum AS ENUM (
  'sunni','mujahid','tabligh','jamate_islami','shia','muslim','not_mentioned'
);

CREATE TYPE marital_status_enum AS ENUM (
  'never_married','divorced','nikkah_divorce','widowed','not_mentioned'
);

CREATE TYPE profession_enum AS ENUM (
  'student','doctor','engineer','farmer','teacher','not_mentioned'
);

CREATE TYPE profession_type_enum AS ENUM (
  'full_time','part_time','freelance','self_employed','not_working','not_mentioned'
);

CREATE TYPE education_level_enum AS ENUM (
  'less_than_high_school','high_school','higher_secondary',
  'under_graduation','post_graduation','not_mentioned'
);

CREATE TYPE home_district_enum AS ENUM (
  'thiruvananthapuram','kollam','pathanamthitta','alappuzha',
  'kottayam','ernakulam','thrissur','palakkad',
  'malappuram','kozhikode','wayanad','kannur',
  'kasaragod','idukki','not_mentioned'
);

-- 2. user_profiles table
CREATE TABLE IF NOT EXISTS user_profiles (
  id                      BIGSERIAL      PRIMARY KEY,    -- local PK
  user_id                 UUID           NOT NULL UNIQUE,       -- from Auth service
  is_bride                BOOLEAN        NOT NULL DEFAULT FALSE,
  full_name               VARCHAR(200)   NULL,
  email                   VARCHAR(255)   ,
  phone                   VARCHAR(20)    ,
  date_of_birth           DATE           NULL,
  height_cm               INT            CHECK (height_cm BETWEEN 130 AND 220),
  physically_challenged    BOOLEAN        NOT NULL DEFAULT FALSE,
  community               community_enum NULL,
  marital_status          marital_status_enum NULL,
  profession              profession_enum NULL,
  profession_type         profession_type_enum NULL,
  highest_education_level education_level_enum NULL,
  home_district           home_district_enum NULL,
  profile_picture_url     VARCHAR(255) NULL,
  last_login              TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
  created_at              TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
  updated_at              TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
  is_deleted              BOOLEAN        NOT NULL DEFAULT FALSE,
  deleted_at              TIMESTAMPTZ
);

-- 3. Indexes
CREATE INDEX IF NOT EXISTS idx_user_profiles_user_id ON user_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_profiles_email ON user_profiles(email);

-- 4. partner_preferences table
CREATE TABLE IF NOT EXISTS partner_preferences (
  id                        BIGSERIAL       PRIMARY KEY,
  user_profile_id           BIGINT          NOT NULL REFERENCES user_profiles(id),
  
  -- Age range
  min_age_years             INT             CHECK (min_age_years BETWEEN 18 AND 80),
  max_age_years             INT             CHECK (max_age_years BETWEEN 18 AND 80),
  
  -- Height range
  min_height_cm             INT             CHECK (min_height_cm BETWEEN 130 AND 220),
  max_height_cm             INT             CHECK (max_height_cm BETWEEN 130 AND 220),
  
  -- Accept physically challenged
  accept_physically_challenged BOOLEAN      NOT NULL DEFAULT TRUE,
  
  -- Multiple Enum Preferences (using array types for flexibility)
  preferred_communities     community_enum[],
  preferred_marital_status  marital_status_enum[],
  preferred_professions     profession_enum[],
  preferred_profession_types profession_type_enum[],
  preferred_education_levels education_level_enum[],
  preferred_home_districts  home_district_enum[],
  
  created_at                TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
  updated_at                TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
  is_deleted                BOOLEAN         NOT NULL DEFAULT FALSE,
  deleted_at                TIMESTAMPTZ,
  
  -- Constraints to ensure valid ranges
  CONSTRAINT age_range_valid CHECK (max_age_years >= min_age_years),
  CONSTRAINT height_range_valid CHECK (max_height_cm >= min_height_cm)
);

-- 5. Indexes for partner_preferences
CREATE INDEX IF NOT EXISTS idx_partner_preferences_user_profile_id 
ON partner_preferences(user_profile_id);
