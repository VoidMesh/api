CREATE TABLE
  users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    username text NOT NULL UNIQUE,
    display_name text NOT NULL,
    email text NOT NULL UNIQUE,
    email_verified boolean DEFAULT false,
    password_hash text NOT NULL,
    reset_password_token text,
    reset_password_expires timestamp,
    created_at timestamp NOT NULL DEFAULT NOW (),
    last_login_at timestamp,
    account_locked boolean DEFAULT false,
    failed_login_attempts integer DEFAULT 0
  );

CREATE TABLE
  meows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    user_id UUID REFERENCES users (id),
    content text NOT NULL,
    created_at timestamp NOT NULL DEFAULT NOW ()
  );

-- Game world tables
CREATE TABLE
  characters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    user_id UUID REFERENCES users (id) ON DELETE CASCADE,
    name text NOT NULL,
    x integer NOT NULL DEFAULT 0,
    y integer NOT NULL DEFAULT 0,
    chunk_x integer NOT NULL DEFAULT 0,
    chunk_y integer NOT NULL DEFAULT 0,
    created_at timestamp NOT NULL DEFAULT NOW (),
    UNIQUE (user_id, name)
  );

CREATE TABLE
  chunks (
    chunk_x integer NOT NULL,
    chunk_y integer NOT NULL,
    seed bigint NOT NULL,
    chunk_data bytea NOT NULL,
    generated_at timestamp NOT NULL DEFAULT NOW (),
    PRIMARY KEY (chunk_x, chunk_y)
  );

CREATE TABLE
  world_settings (
    key text PRIMARY KEY,
    value text NOT NULL
  );

-- Create indexes for performance
CREATE INDEX idx_characters_user_id ON characters (user_id);
CREATE INDEX idx_characters_position ON characters (chunk_x, chunk_y);
CREATE INDEX idx_chunks_position ON chunks (chunk_x, chunk_y);

-- Sessions table for web application (fiber storage format)
CREATE TABLE
  sessions (
    k VARCHAR(64) NOT NULL DEFAULT '',
    v BYTEA NOT NULL,
    e BIGINT NOT NULL DEFAULT '0',
    PRIMARY KEY (k)
  );

-- Create index for session cleanup
CREATE INDEX e ON sessions (e);

-- Insert default world settings
INSERT INTO
  world_settings (key, value)
VALUES
  ('seed', '12345'),
  ('chunk_size', '32'),
  ('world_name', 'VoidMesh World');
