-- Character inventory operations

-- name: GetCharacterInventory :many
SELECT
  ci.id,
  ci.character_id,
  ci.item_id,
  ci.quantity,
  ci.created_at,
  ci.updated_at,
  i.name as item_name,
  i.description as item_description,
  i.item_type,
  i.rarity,
  i.stack_size,
  i.visual_data
FROM character_inventories ci
JOIN items i ON ci.item_id = i.id
WHERE ci.character_id = $1
ORDER BY i.name;

-- name: GetInventoryItem :one
SELECT
  ci.id,
  ci.character_id,
  ci.item_id,
  ci.quantity,
  ci.created_at,
  ci.updated_at,
  i.name as item_name,
  i.description as item_description,
  i.item_type,
  i.rarity,
  i.stack_size,
  i.visual_data
FROM character_inventories ci
JOIN items i ON ci.item_id = i.id
WHERE ci.character_id = $1 AND ci.item_id = $2;

-- name: CreateInventoryItem :one
INSERT INTO character_inventories (
  character_id,
  item_id,
  quantity
)
VALUES ($1, $2, $3)
RETURNING id, character_id, item_id, quantity, created_at, updated_at;

-- name: UpdateInventoryItemQuantity :one
UPDATE character_inventories
SET 
  quantity = $3,
  updated_at = NOW()
WHERE character_id = $1 AND item_id = $2
RETURNING id, character_id, item_id, quantity, created_at, updated_at;

-- name: AddInventoryItemQuantity :one
UPDATE character_inventories
SET 
  quantity = quantity + $3,
  updated_at = NOW()
WHERE character_id = $1 AND item_id = $2
RETURNING id, character_id, item_id, quantity, created_at, updated_at;

-- name: RemoveInventoryItemQuantity :one
UPDATE character_inventories
SET 
  quantity = quantity - $3,
  updated_at = NOW()
WHERE character_id = $1 AND item_id = $2 AND quantity >= $3
RETURNING id, character_id, item_id, quantity, created_at, updated_at;

-- name: DeleteInventoryItem :exec
DELETE FROM character_inventories
WHERE character_id = $1 AND item_id = $2;

-- name: DeleteEmptyInventoryItems :exec
DELETE FROM character_inventories
WHERE character_id = $1 AND quantity <= 0;

-- name: InventoryItemExists :one
SELECT EXISTS(
  SELECT 1 FROM character_inventories
  WHERE character_id = $1 AND item_id = $2
);

-- name: GetInventoryItemCount :one
SELECT COUNT(*) FROM character_inventories
WHERE character_id = $1;

-- name: GetInventoryItemTotalQuantity :one
SELECT COALESCE(SUM(quantity), 0) FROM character_inventories
WHERE character_id = $1;