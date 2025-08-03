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

-- Resource system tables
CREATE TABLE
  resource_types (
    id SERIAL PRIMARY KEY,
    name text NOT NULL,
    description text,
    terrain_type text NOT NULL,
    rarity text NOT NULL,
    visual_data jsonb,
    properties jsonb,
    created_at timestamp NOT NULL DEFAULT NOW()
  );

CREATE TABLE
  resource_nodes (
    id SERIAL PRIMARY KEY,
    resource_type_id integer NOT NULL REFERENCES resource_types (id),
    chunk_x integer NOT NULL,
    chunk_y integer NOT NULL,
    cluster_id text NOT NULL,
    pos_x integer NOT NULL,
    pos_y integer NOT NULL,
    size integer NOT NULL DEFAULT 1,
    created_at timestamp NOT NULL DEFAULT NOW(),
    FOREIGN KEY (chunk_x, chunk_y) REFERENCES chunks (chunk_x, chunk_y) ON DELETE CASCADE,
    UNIQUE (chunk_x, chunk_y, pos_x, pos_y)
  );

-- Create indexes for performance
CREATE INDEX idx_characters_user_id ON characters (user_id);
CREATE INDEX idx_characters_position ON characters (chunk_x, chunk_y);
CREATE INDEX idx_chunks_position ON chunks (chunk_x, chunk_y);
CREATE INDEX idx_resource_nodes_chunk ON resource_nodes (chunk_x, chunk_y);
CREATE INDEX idx_resource_nodes_type ON resource_nodes (resource_type_id);
CREATE INDEX idx_resource_nodes_cluster ON resource_nodes (cluster_id);


-- Insert default world settings
INSERT INTO
  world_settings (key, value)
VALUES
  ('seed', '12345'),
  ('chunk_size', '32'),
  ('world_name', 'VoidMesh World');

-- Seed data for resource_types

-- Grass Terrain Resources
INSERT INTO resource_types (name, description, terrain_type, rarity, visual_data, properties)
VALUES 
  (
    'Herb Patch',
    'A cluster of medicinal herbs with various healing properties.',
    'TERRAIN_TYPE_GRASS',
    'common',
    '{"sprite": "herb_patch", "color": "#7CFC00"}',
    '{"harvest_time": 2, "respawn_time": 300, "yield_min": 1, "yield_max": 3, "secondary_drops": [{"name": "Common Grass", "chance": 0.7, "min": 1, "max": 2}, {"name": "Seeds", "chance": 0.3, "min": 1, "max": 1}]}'
  ),
  (
    'Berry Bush',
    'A bush full of sweet, edible berries.',
    'TERRAIN_TYPE_GRASS',
    'common',
    '{"sprite": "berry_bush", "color": "#8B0000"}',
    '{"harvest_time": 3, "respawn_time": 400, "yield_min": 2, "yield_max": 5, "secondary_drops": [{"name": "Twigs", "chance": 0.5, "min": 1, "max": 2}, {"name": "Leaves", "chance": 0.6, "min": 1, "max": 3}]}'
  ),
  (
    'Mineral Outcropping',
    'A small deposit of valuable minerals protruding from the ground.',
    'TERRAIN_TYPE_GRASS',
    'uncommon',
    '{"sprite": "mineral_outcrop", "color": "#A9A9A9"}',
    '{"harvest_time": 5, "respawn_time": 600, "yield_min": 1, "yield_max": 3, "secondary_drops": [{"name": "Stone", "chance": 0.8, "min": 1, "max": 3}, {"name": "Dirt", "chance": 0.4, "min": 1, "max": 2}]}'
  );

-- Water Terrain Resources
INSERT INTO resource_types (name, description, terrain_type, rarity, visual_data, properties)
VALUES 
  (
    'Fishing Spot',
    'A location with an abundance of fish.',
    'TERRAIN_TYPE_WATER',
    'common',
    '{"sprite": "fishing_spot", "color": "#1E90FF"}',
    '{"harvest_time": 4, "respawn_time": 240, "yield_min": 1, "yield_max": 3, "secondary_drops": [{"name": "Algae", "chance": 0.4, "min": 1, "max": 2}, {"name": "Shells", "chance": 0.2, "min": 1, "max": 1}]}'
  ),
  (
    'Kelp Bed',
    'A thick growth of underwater plants, useful for crafting and cooking.',
    'TERRAIN_TYPE_WATER',
    'common',
    '{"sprite": "kelp_bed", "color": "#2E8B57"}',
    '{"harvest_time": 3, "respawn_time": 360, "yield_min": 2, "yield_max": 4, "secondary_drops": [{"name": "Salt", "chance": 0.3, "min": 1, "max": 1}, {"name": "Tiny Fish", "chance": 0.25, "min": 1, "max": 1}]}'
  ),
  (
    'Pearl Formation',
    'A cluster of oysters that may contain valuable pearls.',
    'TERRAIN_TYPE_WATER',
    'rare',
    '{"sprite": "pearl_formation", "color": "#FFFFFF"}',
    '{"harvest_time": 6, "respawn_time": 900, "yield_min": 1, "yield_max": 2, "secondary_drops": [{"name": "Shells", "chance": 0.9, "min": 2, "max": 4}, {"name": "Sand", "chance": 0.5, "min": 1, "max": 2}]}'
  );

-- Sand Terrain Resources
INSERT INTO resource_types (name, description, terrain_type, rarity, visual_data, properties)
VALUES 
  (
    'Crystal Formation',
    'Beautiful crystals with magical properties.',
    'TERRAIN_TYPE_SAND',
    'uncommon',
    '{"sprite": "crystal_formation", "color": "#B19CD9"}',
    '{"harvest_time": 5, "respawn_time": 720, "yield_min": 1, "yield_max": 3, "secondary_drops": [{"name": "Sand", "chance": 0.8, "min": 2, "max": 4}, {"name": "Stone Fragments", "chance": 0.4, "min": 1, "max": 2}]}'
  ),
  (
    'Clay Deposit',
    'A rich deposit of clay, perfect for pottery and building.',
    'TERRAIN_TYPE_SAND',
    'common',
    '{"sprite": "clay_deposit", "color": "#CD853F"}',
    '{"harvest_time": 4, "respawn_time": 480, "yield_min": 2, "yield_max": 5, "secondary_drops": [{"name": "Sand", "chance": 0.7, "min": 1, "max": 3}, {"name": "Silt", "chance": 0.4, "min": 1, "max": 2}]}'
  ),
  (
    'Desert Plant',
    'A rare plant adapted to the harsh desert conditions.',
    'TERRAIN_TYPE_SAND',
    'uncommon',
    '{"sprite": "desert_plant", "color": "#EEDD82"}',
    '{"harvest_time": 2, "respawn_time": 540, "yield_min": 1, "yield_max": 2, "secondary_drops": [{"name": "Sand", "chance": 0.6, "min": 1, "max": 2}, {"name": "Seeds", "chance": 0.2, "min": 1, "max": 1}]}'
  );

-- Stone/Wood Terrain Resources
INSERT INTO resource_types (name, description, terrain_type, rarity, visual_data, properties)
VALUES 
  (
    'Harvestable Tree',
    'A mature tree that can be harvested for wood.',
    'TERRAIN_TYPE_DIRT',
    'common',
    '{"sprite": "harvestable_tree", "color": "#8B4513"}',
    '{"harvest_time": 8, "respawn_time": 1200, "yield_min": 3, "yield_max": 8, "secondary_drops": [{"name": "Sticks", "chance": 0.8, "min": 2, "max": 4}, {"name": "Leaves", "chance": 0.9, "min": 3, "max": 6}, {"name": "Bark", "chance": 0.4, "min": 1, "max": 2}]}'
  ),
  (
    'Mushroom Circle',
    'A circle of mushrooms with alchemical properties.',
    'TERRAIN_TYPE_DIRT',
    'uncommon',
    '{"sprite": "mushroom_circle", "color": "#FFE4B5"}',
    '{"harvest_time": 2, "respawn_time": 480, "yield_min": 2, "yield_max": 6, "secondary_drops": [{"name": "Spores", "chance": 0.3, "min": 1, "max": 1}, {"name": "Dirt", "chance": 0.6, "min": 1, "max": 2}]}'
  ),
  (
    'Wild Honey Hive',
    'A beehive containing wild honey.',
    'TERRAIN_TYPE_DIRT',
    'rare',
    '{"sprite": "honey_hive", "color": "#FFD700"}',
    '{"harvest_time": 5, "respawn_time": 900, "yield_min": 1, "yield_max": 4, "secondary_drops": [{"name": "Beeswax", "chance": 0.7, "min": 1, "max": 2}, {"name": "Bark", "chance": 0.3, "min": 1, "max": 1}]}'
  ),
  (
    'Stone Vein',
    'A rich vein of high-quality stone.',
    'TERRAIN_TYPE_STONE',
    'common',
    '{"sprite": "stone_vein", "color": "#708090"}',
    '{"harvest_time": 10, "respawn_time": 720, "yield_min": 3, "yield_max": 8, "secondary_drops": [{"name": "Gravel", "chance": 0.6, "min": 2, "max": 4}, {"name": "Dust", "chance": 0.4, "min": 1, "max": 2}]}'
  ),
  (
    'Gem Deposit',
    'A deposit containing precious gems.',
    'TERRAIN_TYPE_STONE',
    'rare',
    '{"sprite": "gem_deposit", "color": "#E0115F"}',
    '{"harvest_time": 12, "respawn_time": 1800, "yield_min": 1, "yield_max": 3, "secondary_drops": [{"name": "Stone", "chance": 0.9, "min": 2, "max": 4}, {"name": "Crystal Fragments", "chance": 0.4, "min": 1, "max": 2}]}'
  ),
  (
    'Metal Ore',
    'A deposit of useful metal ore.',
    'TERRAIN_TYPE_STONE',
    'uncommon',
    '{"sprite": "metal_ore", "color": "#B87333"}',
    '{"harvest_time": 15, "respawn_time": 1500, "yield_min": 2, "yield_max": 5, "secondary_drops": [{"name": "Stone", "chance": 0.9, "min": 2, "max": 5}, {"name": "Pyrite", "chance": 0.3, "min": 1, "max": 2}, {"name": "Sulfur", "chance": 0.2, "min": 1, "max": 1}]}'
  );
