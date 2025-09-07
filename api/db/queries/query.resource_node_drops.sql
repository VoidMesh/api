-- Resource node drops operations

-- name: GetResourceNodeDrops :many
SELECT
  rnd.id,
  rnd.resource_node_type_id,
  rnd.item_id,
  rnd.chance,
  rnd.min_quantity,
  rnd.max_quantity,
  rnd.created_at,
  i.name as item_name,
  i.description as item_description,
  i.item_type,
  i.rarity,
  i.stack_size,
  i.visual_data
FROM resource_node_drops rnd
JOIN items i ON rnd.item_id = i.id
WHERE rnd.resource_node_type_id = $1
ORDER BY rnd.chance DESC;

-- name: GetAllResourceNodeDrops :many
SELECT
  rnd.id,
  rnd.resource_node_type_id,
  rnd.item_id,
  rnd.chance,
  rnd.min_quantity,
  rnd.max_quantity,
  rnd.created_at,
  i.name as item_name,
  i.description as item_description,
  i.item_type,
  i.rarity,
  i.stack_size,
  i.visual_data
FROM resource_node_drops rnd
JOIN items i ON rnd.item_id = i.id
ORDER BY rnd.resource_node_type_id, rnd.chance DESC;

-- name: CreateResourceNodeDrop :one
INSERT INTO resource_node_drops (
  resource_node_type_id,
  item_id,
  chance,
  min_quantity,
  max_quantity
)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, resource_node_type_id, item_id, chance, min_quantity, max_quantity, created_at;

-- name: UpdateResourceNodeDrop :one
UPDATE resource_node_drops
SET
  chance = $3,
  min_quantity = $4,
  max_quantity = $5
WHERE resource_node_type_id = $1 AND item_id = $2
RETURNING id, resource_node_type_id, item_id, chance, min_quantity, max_quantity, created_at;

-- name: DeleteResourceNodeDrop :exec
DELETE FROM resource_node_drops
WHERE resource_node_type_id = $1 AND item_id = $2;

-- name: DeleteAllResourceNodeDrops :exec
DELETE FROM resource_node_drops
WHERE resource_node_type_id = $1;