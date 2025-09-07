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
  worlds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    seed bigint NOT NULL,
    created_at timestamp NOT NULL DEFAULT NOW()
  );

CREATE TABLE
  chunks (
    world_id UUID NOT NULL REFERENCES worlds(id) ON DELETE CASCADE,
    chunk_x integer NOT NULL,
    chunk_y integer NOT NULL,
    chunk_data bytea NOT NULL,
    generated_at timestamp NOT NULL DEFAULT NOW(),
    PRIMARY KEY (world_id, chunk_x, chunk_y)
  );

-- Resource node system tables
CREATE TABLE
  resource_nodes (
    id SERIAL PRIMARY KEY,
    resource_node_type_id integer NOT NULL, -- Resource node type ID (defined in code as enum)
    world_id UUID NOT NULL,
    chunk_x integer NOT NULL,
    chunk_y integer NOT NULL,
    cluster_id text NOT NULL,
    x integer NOT NULL, -- Global X coordinate
    y integer NOT NULL, -- Global Y coordinate
    size integer NOT NULL DEFAULT 1,
    created_at timestamp NOT NULL DEFAULT NOW(),
    FOREIGN KEY (world_id, chunk_x, chunk_y) REFERENCES chunks (world_id, chunk_x, chunk_y) ON DELETE CASCADE,
    UNIQUE (world_id, x, y)
  );

-- Items system - all harvestable items
CREATE TABLE
  items (
    id SERIAL PRIMARY KEY,
    name text NOT NULL UNIQUE,
    description text NOT NULL,
    item_type text NOT NULL DEFAULT 'material', -- 'material', 'resource_node', 'tool', etc.
    rarity text NOT NULL DEFAULT 'common', -- 'common', 'uncommon', 'rare', 'very_rare'
    stack_size integer NOT NULL DEFAULT 64,
    visual_data jsonb, -- Contains sprite, color, etc.
    created_at timestamp NOT NULL DEFAULT NOW()
  );

-- Resource node drop system
CREATE TABLE
  resource_node_drops (
    id SERIAL PRIMARY KEY,
    resource_node_type_id integer NOT NULL,
    item_id integer NOT NULL REFERENCES items (id) ON DELETE CASCADE,
    chance decimal(4,3) NOT NULL CHECK (chance >= 0.0 AND chance <= 1.0),
    min_quantity integer NOT NULL DEFAULT 1 CHECK (min_quantity > 0),
    max_quantity integer NOT NULL DEFAULT 1 CHECK (max_quantity >= min_quantity),
    created_at timestamp NOT NULL DEFAULT NOW(),
    UNIQUE (resource_node_type_id, item_id)
  );

-- Character inventory system
CREATE TABLE
  character_inventories (
    id SERIAL PRIMARY KEY,
    character_id UUID NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    item_id integer NOT NULL REFERENCES items (id) ON DELETE CASCADE,
    quantity integer NOT NULL DEFAULT 1 CHECK (quantity > 0),
    created_at timestamp NOT NULL DEFAULT NOW(),
    updated_at timestamp NOT NULL DEFAULT NOW(),
    UNIQUE (character_id, item_id)
  );

-- Create indexes for performance
CREATE INDEX idx_characters_user_id ON characters (user_id);
CREATE INDEX idx_characters_position ON characters (chunk_x, chunk_y);
CREATE INDEX idx_chunks_world_id ON chunks (world_id);
CREATE INDEX idx_chunks_position ON chunks (world_id, chunk_x, chunk_y);
CREATE INDEX idx_resource_nodes_world_id ON resource_nodes (world_id);
CREATE INDEX idx_resource_nodes_chunk ON resource_nodes (world_id, chunk_x, chunk_y);
CREATE INDEX idx_resource_nodes_type ON resource_nodes (resource_node_type_id);
CREATE INDEX idx_resource_nodes_cluster ON resource_nodes (cluster_id);
CREATE INDEX idx_resource_nodes_global_position ON resource_nodes (world_id, x, y);
CREATE INDEX idx_items_name ON items (name);
CREATE INDEX idx_items_type ON items (item_type);
CREATE INDEX idx_items_rarity ON items (rarity);
CREATE INDEX idx_resource_node_drops_resource_type ON resource_node_drops (resource_node_type_id);
CREATE INDEX idx_resource_node_drops_item ON resource_node_drops (item_id);
CREATE INDEX idx_character_inventories_character_id ON character_inventories (character_id);
CREATE INDEX idx_character_inventories_item_id ON character_inventories (item_id);


-- Insert default world
INSERT INTO
  worlds (name, seed)
VALUES
  ('VoidMesh World', floor(random() * 1000000000)::bigint);
