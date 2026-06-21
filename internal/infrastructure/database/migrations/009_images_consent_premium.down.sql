DROP TABLE IF EXISTS note_images;
ALTER TABLE users DROP COLUMN IF EXISTS consent_given_at;
ALTER TABLE users ALTER COLUMN is_premium SET DEFAULT false;
