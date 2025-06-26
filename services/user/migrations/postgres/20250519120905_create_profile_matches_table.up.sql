CREATE TYPE match_status_enum AS ENUM (
  'liked', 'passed'
);

CREATE TABLE profile_matches (
  id BIGSERIAL PRIMARY KEY,
  user_id UUID NOT NULL,
  target_id UUID NOT NULL,
  status match_status_enum NOT NULL,
  is_disliked BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
  deleted_at TIMESTAMPTZ,
  CONSTRAINT unique_profile_match UNIQUE(user_id, target_id)
);

CREATE INDEX idx_profile_matches_user_id ON profile_matches(user_id);
CREATE INDEX idx_profile_matches_target_id ON profile_matches(target_id);
CREATE INDEX idx_profile_matches_status ON profile_matches(status);
CREATE INDEX idx_profile_matches_created_at ON profile_matches(created_at);