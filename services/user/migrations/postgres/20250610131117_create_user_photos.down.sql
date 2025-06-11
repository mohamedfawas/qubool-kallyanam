-- Drop indexes
DROP INDEX IF EXISTS idx_user_photos_created_at;
DROP INDEX IF EXISTS idx_user_photos_display_order;
DROP INDEX IF EXISTS idx_user_photos_user_id;

-- Drop table
DROP TABLE IF EXISTS user_photos;