-- Drop indexes
DROP INDEX IF EXISTS idx_user_videos_created_at;
DROP INDEX IF EXISTS idx_user_videos_user_id;

-- Drop table
DROP TABLE IF EXISTS user_videos;