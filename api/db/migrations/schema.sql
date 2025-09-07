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

-- Insert harvestable items
INSERT INTO items (name, description, item_type, rarity, stack_size, visual_data) VALUES
  -- Primary harvest materials
  ('Herbs', 'Medicinal herbs with healing properties', 'material', 'common', 64, '{"sprite": "herbs", "color": "#7CFC00"}'),
  ('Berries', 'Sweet, edible berries', 'material', 'common', 64, '{"sprite": "berries", "color": "#8B0000"}'),
  ('Minerals', 'Valuable minerals and crystals', 'material', 'uncommon', 64, '{"sprite": "minerals", "color": "#A9A9A9"}'),
  ('Fish', 'Fresh caught fish', 'material', 'common', 64, '{"sprite": "fish", "color": "#1E90FF"}'),
  
  -- Secondary drop materials
  ('Common Grass', 'Common grass found everywhere', 'material', 'common', 64, '{"sprite": "grass", "color": "#228B22"}'),
  ('Seeds', 'Various plant seeds', 'material', 'common', 64, '{"sprite": "seeds", "color": "#8B4513"}'),
  ('Twigs', 'Small branches and twigs', 'material', 'common', 64, '{"sprite": "twigs", "color": "#654321"}'),
  ('Leaves', 'Fallen leaves', 'material', 'common', 64, '{"sprite": "leaves", "color": "#32CD32"}'),
  ('Stone', 'Common stone pieces', 'material', 'common', 64, '{"sprite": "stone", "color": "#696969"}'),
  ('Dirt', 'Rich soil and dirt', 'material', 'common', 64, '{"sprite": "dirt", "color": "#8B4513"}'),
  ('Algae', 'Underwater plant matter', 'material', 'common', 64, '{"sprite": "algae", "color": "#006400"}'),
  ('Shells', 'Decorative seashells', 'material', 'uncommon', 64, '{"sprite": "shells", "color": "#F5DEB3"}');

-- Insert resource node drop configurations
INSERT INTO resource_node_drops (resource_node_type_id, item_id, chance, min_quantity, max_quantity) VALUES
  -- Herb Patch (ID: 1) drops
  (1, (SELECT id FROM items WHERE name = 'Herbs'), 1.0, 1, 3),                -- Primary drop: Herbs (100% chance, 1-3 yield)
  (1, (SELECT id FROM items WHERE name = 'Common Grass'), 0.7, 1, 2),         -- Secondary: Common Grass (70% chance, 1-2 amount)
  (1, (SELECT id FROM items WHERE name = 'Seeds'), 0.3, 1, 1),                -- Secondary: Seeds (30% chance, 1 amount)
  
  -- Berry Bush (ID: 2) drops
  (2, (SELECT id FROM items WHERE name = 'Berries'), 1.0, 2, 5),              -- Primary drop: Berries (100% chance, 2-5 yield)
  (2, (SELECT id FROM items WHERE name = 'Twigs'), 0.5, 1, 2),                -- Secondary: Twigs (50% chance, 1-2 amount)
  (2, (SELECT id FROM items WHERE name = 'Leaves'), 0.6, 1, 3),               -- Secondary: Leaves (60% chance, 1-3 amount)
  
  -- Mineral Outcropping (ID: 3) drops
  (3, (SELECT id FROM items WHERE name = 'Minerals'), 1.0, 1, 3),             -- Primary drop: Minerals (100% chance, 1-3 yield)
  (3, (SELECT id FROM items WHERE name = 'Stone'), 0.8, 1, 3),                -- Secondary: Stone (80% chance, 1-3 amount)
  (3, (SELECT id FROM items WHERE name = 'Dirt'), 0.4, 1, 2),                 -- Secondary: Dirt (40% chance, 1-2 amount)
  
  -- Fishing Spot (ID: 4) drops
  (4, (SELECT id FROM items WHERE name = 'Fish'), 1.0, 1, 3),                 -- Primary drop: Fish (100% chance, 1-3 yield)
  (4, (SELECT id FROM items WHERE name = 'Algae'), 0.4, 1, 2),                -- Secondary: Algae (40% chance, 1-2 amount)
  (4, (SELECT id FROM items WHERE name = 'Shells'), 0.2, 1, 1);               -- Secondary: Shells (20% chance, 1 amount)
