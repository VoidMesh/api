-- Items table operations

-- name: GetItem :one
SELECT
  id,
  name,
  description,
  item_type,
  rarity,
  stack_size,
  visual_data,
  created_at
FROM items
WHERE id = $1;

-- name: GetItemByName :one
SELECT
  id,
  name,
  description,
  item_type,
  rarity,
  stack_size,
  visual_data,
  created_at
FROM items
WHERE name = $1;

-- name: GetAllItems :many
SELECT
  id,
  name,
  description,
  item_type,
  rarity,
  stack_size,
  visual_data,
  created_at
FROM items
ORDER BY name;

-- name: GetItemsByType :many
SELECT
  id,
  name,
  description,
  item_type,
  rarity,
  stack_size,
  visual_data,
  created_at
FROM items
WHERE item_type = $1
ORDER BY name;

-- name: CreateItem :one
INSERT INTO items (
  name,
  description,
  item_type,
  rarity,
  stack_size,
  visual_data
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, description, item_type, rarity, stack_size, visual_data, created_at;

-- name: UpdateItem :one
UPDATE items
SET
  name = $2,
  description = $3,
  item_type = $4,
  rarity = $5,
  stack_size = $6,
  visual_data = $7
WHERE id = $1
RETURNING id, name, description, item_type, rarity, stack_size, visual_data, created_at;

-- name: DeleteItem :exec
DELETE FROM items
WHERE id = $1;