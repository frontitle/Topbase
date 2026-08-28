-- User-managed profile details, including a compact client-resized avatar.
ALTER TABLE users ADD COLUMN avatar_url TEXT NOT NULL DEFAULT '';
