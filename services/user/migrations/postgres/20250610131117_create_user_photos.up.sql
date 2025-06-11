-- Create user_photos table for additional photos (excluding profile photos)
CREATE TABLE user_photos (
  id                    BIGSERIAL      PRIMARY KEY,
  user_id               UUID           NOT NULL,
  photo_url             VARCHAR(500)   NOT NULL,
  photo_key             VARCHAR(500)   NOT NULL,
  display_order         SMALLINT       NOT NULL CHECK (display_order BETWEEN 1 AND 3),
  created_at            TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
  
  CONSTRAINT fk_user_photos_user_id 
    FOREIGN KEY (user_id) REFERENCES user_profiles(user_id) ON DELETE CASCADE,
  CONSTRAINT unique_user_photo_order 
    UNIQUE (user_id, display_order)
);

-- Create indexes
CREATE INDEX idx_user_photos_user_id ON user_photos(user_id);
CREATE INDEX idx_user_photos_display_order ON user_photos(display_order);
CREATE INDEX idx_user_photos_created_at ON user_photos(created_at DESC);