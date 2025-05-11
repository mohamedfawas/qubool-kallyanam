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
  user_id                 UUID           NOT NULL,       -- from Auth service
  is_bride                BOOLEAN        NOT NULL DEFAULT FALSE,
  full_name               VARCHAR(200)   ,
  phone VARCHAR(20),
  date_of_birth           DATE           ,
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
  deleted_at              TIMESTAMPTZ
);

-- 3. Indexes
CREATE INDEX IF NOT EXISTS idx_user_profiles_user_id ON user_profiles(user_id);
