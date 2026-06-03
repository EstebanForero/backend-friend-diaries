ALTER TABLE reactions
  ALTER COLUMN reaction_type TYPE TEXT USING reaction_type::TEXT;

DROP TYPE IF EXISTS reaction_type;
