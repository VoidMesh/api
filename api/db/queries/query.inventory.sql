-- Character inventory operations

-- name: GetCharacterInventory :many
SELECT
  ci.id,
  ci.character_id,
  ci.resource_node_type_id,
  ci.quantity,
  ci.created_at,
  ci.updated_at
FROM character_inventories ci
WHERE ci.character_id = $1
ORDER BY ci.created_at DESC;

-- name: GetInventoryItem :one
SELECT
  ci.id,
  ci.character_id,
  ci.resource_node_type_id,
  ci.quantity,
  ci.created_at,
  ci.updated_at
FROM character_inventories ci
WHERE ci.character_id = $1 AND ci.resource_node_type_id = $2;

-- name: CreateInventoryItem :one
INSERT INTO character_inventories (
  character_id,
  resource_node_type_id,
  quantity
)
VALUES ($1, $2, $3)
RETURNING id, character_id, resource_node_type_id, quantity, created_at, updated_at;

-- name: UpdateInventoryItemQuantity :one
UPDATE character_inventories
SET 
  quantity = $3,
  updated_at = NOW()
WHERE character_id = $1 AND resource_node_type_id = $2
RETURNING id, character_id, resource_node_type_id, quantity, created_at, updated_at;

-- name: AddInventoryItemQuantity :one
UPDATE character_inventories
SET 
  quantity = quantity + $3,
  updated_at = NOW()
WHERE character_id = $1 AND resource_node_type_id = $2
RETURNING id, character_id, resource_node_type_id, quantity, created_at, updated_at;

-- name: RemoveInventoryItemQuantity :one
UPDATE character_inventories
SET 
  quantity = quantity - $3,
  updated_at = NOW()
WHERE character_id = $1 AND resource_node_type_id = $2 AND quantity >= $3
RETURNING id, character_id, resource_node_type_id, quantity, created_at, updated_at;

-- name: DeleteInventoryItem :exec
DELETE FROM character_inventories
WHERE character_id = $1 AND resource_node_type_id = $2;

-- name: DeleteEmptyInventoryItems :exec
DELETE FROM character_inventories
WHERE character_id = $1 AND quantity <= 0;

-- name: InventoryItemExists :one
SELECT EXISTS(
  SELECT 1 FROM character_inventories
  WHERE character_id = $1 AND resource_node_type_id = $2
);

-- name: GetInventoryItemCount :one
SELECT COUNT(*) FROM character_inventories
WHERE character_id = $1;

-- name: GetInventoryItemTotalQuantity :one
SELECT COALESCE(SUM(quantity), 0) FROM character_inventories
WHERE character_id = $1;