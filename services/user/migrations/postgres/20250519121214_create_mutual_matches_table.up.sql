CREATE TABLE mutual_matches (
  id BIGSERIAL PRIMARY KEY,
  user_id_1 UUID NOT NULL,
  user_id_2 UUID NOT NULL,
  matched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
  deleted_at TIMESTAMPTZ,
  -- Ensure consistent ordering of user IDs and prevent duplicates
  CONSTRAINT unique_mutual_match UNIQUE(user_id_1, user_id_2),
  CONSTRAINT order_user_ids CHECK (user_id_1 < user_id_2)
);

CREATE INDEX idx_mutual_matches_user_id_1 ON mutual_matches(user_id_1);
CREATE INDEX idx_mutual_matches_user_id_2 ON mutual_matches(user_id_2);
CREATE INDEX idx_mutual_matches_matched_at ON mutual_matches(matched_at DESC);
CREATE INDEX idx_mutual_matches_is_active ON mutual_matches(is_active);