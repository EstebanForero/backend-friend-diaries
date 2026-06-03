CREATE TYPE reaction_type AS ENUM ('love', 'funny', 'sad', 'support', 'seen', 'favorite');

UPDATE reactions
SET reaction_type = 'love'
WHERE reaction_type NOT IN ('love', 'funny', 'sad', 'support', 'seen', 'favorite');

ALTER TABLE reactions
  ALTER COLUMN reaction_type TYPE reaction_type USING reaction_type::reaction_type;
