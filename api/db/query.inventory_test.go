package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateInventoryItem(t *testing.T) {
	tests := []struct {
		name        string
		params      CreateInventoryItemParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, item CharacterInventory)
	}{
		{
			name: "successful inventory item creation",
			params: CreateInventoryItemParams{
				CharacterID: mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID:      1, // Wood item
				Quantity:    10,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "character_id", "item_id", "quantity", "created_at", "updated_at",
				}).AddRow(
					int32(1), "750e8400-e29b-41d4-a716-446655440000", int32(1), int32(10), 
					pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO character_inventories").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(1), int32(10)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, item CharacterInventory) {
				assert.Equal(t, mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), item.CharacterID)
				assert.Equal(t, int32(1), item.ItemID)
				assert.Equal(t, int32(10), item.Quantity)
				assert.True(t, item.CreatedAt.Valid)
				assert.True(t, item.UpdatedAt.Valid)
			},
		},
		{
			name: "create inventory item with minimum quantity",
			params: CreateInventoryItemParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 2, // Stone
				Quantity:           1,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "character_id", "item_id", "quantity", "created_at", "updated_at",
				}).AddRow(
					int32(2), "750e8400-e29b-41d4-a716-446655440000", int32(2), int32(1), 
					pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO character_inventories").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(2), int32(1)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, item CharacterInventory) {
				assert.Equal(t, int32(2), item.ItemID)
				assert.Equal(t, int32(1), item.Quantity)
			},
		},
		{
			name: "create inventory item with large quantity",
			params: CreateInventoryItemParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 3, // Iron
				Quantity:           999999,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "character_id", "item_id", "quantity", "created_at", "updated_at",
				}).AddRow(
					int32(3), "750e8400-e29b-41d4-a716-446655440000", int32(3), int32(999999), 
					pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO character_inventories").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(3), int32(999999)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, item CharacterInventory) {
				assert.Equal(t, int32(3), item.ItemID)
				assert.Equal(t, int32(999999), item.Quantity)
			},
		},
		{
			name: "duplicate inventory item - unique constraint violation",
			params: CreateInventoryItemParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 1, // Already exists for this character
				Quantity:           5,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO character_inventories").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(1), int32(5)).
					WillReturnError(sql.ErrConnDone) // Simulate unique constraint violation
			},
			wantErr: true,
		},
		{
			name: "invalid character foreign key",
			params: CreateInventoryItemParams{
				CharacterID:        mustParseUUID("999e8400-e29b-41d4-a716-446655440000"), // Non-existent character
				ItemID: 1,
				Quantity:           10,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO character_inventories").
					WithArgs(mustParseUUID("999e8400-e29b-41d4-a716-446655440000"), int32(1), int32(10)).
					WillReturnError(sql.ErrConnDone) // Simulate foreign key constraint violation
			},
			wantErr: true,
		},
		{
			name: "zero quantity inventory item",
			params: CreateInventoryItemParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 4,
				Quantity:           0,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "character_id", "item_id", "quantity", "created_at", "updated_at",
				}).AddRow(
					int32(4), "750e8400-e29b-41d4-a716-446655440000", int32(4), int32(0), 
					pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("INSERT INTO character_inventories").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(4), int32(0)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, item CharacterInventory) {
				assert.Equal(t, int32(0), item.Quantity)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			item, err := queries.CreateInventoryItem(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, item)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetCharacterInventory(t *testing.T) {
	tests := []struct {
		name        string
		characterID pgtype.UUID
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, inventory []GetCharacterInventoryRow)
	}{
		{
			name:        "full inventory retrieval",
			characterID: mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "character_id", "item_id", "quantity", "created_at", "updated_at",
					"item_name", "item_description", "item_type", "rarity", "stack_size", "visual_data",
				}).
					AddRow(
						int32(3), "750e8400-e29b-41d4-a716-446655440000", int32(3), int32(50), 
						pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
						"Iron", "Metal ore", "material", "common", int32(64), []byte(`{"sprite": "iron"}`),
					).
					AddRow(
						int32(2), "750e8400-e29b-41d4-a716-446655440000", int32(2), int32(25), 
						pgtype.Timestamp{Time: now.Add(-time.Hour), Valid: true}, pgtype.Timestamp{Time: now.Add(-time.Hour), Valid: true},
						"Stone", "Common stone", "material", "common", int32(64), []byte(`{"sprite": "stone"}`),
					).
					AddRow(
						int32(1), "750e8400-e29b-41d4-a716-446655440000", int32(1), int32(100), 
						pgtype.Timestamp{Time: now.Add(-2*time.Hour), Valid: true}, pgtype.Timestamp{Time: now.Add(-2*time.Hour), Valid: true},
						"Wood", "Basic wood", "material", "common", int32(64), []byte(`{"sprite": "wood"}`),
					)
				mock.ExpectQuery("SELECT (.+) FROM character_inventories ci JOIN items i ON ci.item_id = i.id WHERE ci.character_id = \\$1 ORDER BY i.name").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000")).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, inventory []GetCharacterInventoryRow) {
				assert.Len(t, inventory, 3)
				// Verify ordering by item name (alphabetical)
				assert.Equal(t, "Iron", inventory[0].ItemName)
				assert.Equal(t, "Stone", inventory[1].ItemName)
				assert.Equal(t, "Wood", inventory[2].ItemName)
				assert.Equal(t, int32(3), inventory[0].ItemID) // Iron
				assert.Equal(t, int32(2), inventory[1].ItemID) // Stone
				assert.Equal(t, int32(1), inventory[2].ItemID) // Wood
			},
		},
		{
			name:        "empty inventory",
			characterID: mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "character_id", "item_id", "quantity", "created_at", "updated_at",
					"item_name", "item_description", "item_type", "rarity", "stack_size", "visual_data",
				})
				mock.ExpectQuery("SELECT (.+) FROM character_inventories ci JOIN items i ON ci.item_id = i.id WHERE ci.character_id = \\$1 ORDER BY i.name").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000")).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, inventory []GetCharacterInventoryRow) {
				assert.Empty(t, inventory)
			},
		},
		{
			name:        "single inventory item",
			characterID: mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "character_id", "item_id", "quantity", "created_at", "updated_at",
					"item_name", "item_description", "item_type", "rarity", "stack_size", "visual_data",
				}).AddRow(
					int32(1), "750e8400-e29b-41d4-a716-446655440000", int32(1), int32(42), 
					pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
					"Test Item", "Test item description", "material", "common", int32(64), []byte(`{"sprite": "test"}`),
				)
				mock.ExpectQuery("SELECT (.+) FROM character_inventories ci JOIN items i ON ci.item_id = i.id WHERE ci.character_id = \\$1 ORDER BY i.name").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000")).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, inventory []GetCharacterInventoryRow) {
				assert.Len(t, inventory, 1)
				assert.Equal(t, int32(1), inventory[0].ItemID)
				assert.Equal(t, int32(42), inventory[0].Quantity)
			},
		},
		{
			name:        "non-existent character",
			characterID: mustParseUUID("999e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{
					"id", "character_id", "item_id", "quantity", "created_at", "updated_at",
				})
				mock.ExpectQuery("SELECT (.+) FROM character_inventories ci JOIN items i ON ci.item_id = i.id WHERE ci.character_id = \\$1 ORDER BY i.name").
					WithArgs(mustParseUUID("999e8400-e29b-41d4-a716-446655440000")).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, inventory []GetCharacterInventoryRow) {
				assert.Empty(t, inventory) // No inventory for non-existent character
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			inventory, err := queries.GetCharacterInventory(createTestContext(), tt.characterID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, inventory)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetInventoryItem(t *testing.T) {
	tests := []struct {
		name        string
		params      GetInventoryItemParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, item GetInventoryItemRow)
	}{
		{
			name: "existing inventory item retrieval",
			params: GetInventoryItemParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 1,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "character_id", "item_id", "quantity", "created_at", "updated_at",
					"item_name", "item_description", "item_type", "rarity", "stack_size", "visual_data",
				}).AddRow(
					int32(1), "750e8400-e29b-41d4-a716-446655440000", int32(1), int32(75), 
					pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
					"Test Item", "A test item", "resource", "common", int32(100), []byte{},
				)
				mock.ExpectQuery("SELECT (.+) FROM character_inventories ci JOIN items i ON ci.item_id = i.id WHERE ci.character_id = \\$1 AND ci.item_id = \\$2").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(1)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, item GetInventoryItemRow) {
				assert.Equal(t, mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), item.CharacterID)
				assert.Equal(t, int32(1), item.ItemID)
				assert.Equal(t, int32(75), item.Quantity)
				assert.Equal(t, "Test Item", item.ItemName)
				assert.Equal(t, "resource", item.ItemType)
			},
		},
		{
			name: "non-existent inventory item",
			params: GetInventoryItemParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 999, // Non-existent resource type
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("SELECT (.+) FROM character_inventories ci JOIN items i ON ci.item_id = i.id WHERE ci.character_id = \\$1 AND ci.item_id = \\$2").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(999)).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			item, err := queries.GetInventoryItem(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, item)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestUpdateInventoryItemQuantity(t *testing.T) {
	tests := []struct {
		name        string
		params      UpdateInventoryItemQuantityParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, item CharacterInventory)
	}{
		{
			name: "successful quantity update",
			params: UpdateInventoryItemQuantityParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 1,
				Quantity:           150,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				createdAt := now.Add(-time.Hour)
				rows := pgxmock.NewRows([]string{
					"id", "character_id", "item_id", "quantity", "created_at", "updated_at",
				}).AddRow(
					int32(1), "750e8400-e29b-41d4-a716-446655440000", int32(1), int32(150), 
					pgtype.Timestamp{Time: createdAt, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("UPDATE character_inventories SET quantity = \\$3, updated_at = NOW\\(\\) WHERE character_id = \\$1 AND item_id = \\$2").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(1), int32(150)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, item CharacterInventory) {
				assert.Equal(t, int32(150), item.Quantity)
				assert.True(t, item.UpdatedAt.Valid)
				// Updated at should be more recent than created at
				assert.True(t, item.UpdatedAt.Time.After(item.CreatedAt.Time))
			},
		},
		{
			name: "update to zero quantity",
			params: UpdateInventoryItemQuantityParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 1,
				Quantity:           0,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "character_id", "item_id", "quantity", "created_at", "updated_at",
				}).AddRow(
					int32(1), "750e8400-e29b-41d4-a716-446655440000", int32(1), int32(0), 
					pgtype.Timestamp{Time: now.Add(-time.Hour), Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("UPDATE character_inventories SET quantity = \\$3, updated_at = NOW\\(\\) WHERE character_id = \\$1 AND item_id = \\$2").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(1), int32(0)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, item CharacterInventory) {
				assert.Equal(t, int32(0), item.Quantity)
			},
		},
		{
			name: "update non-existent inventory item",
			params: UpdateInventoryItemQuantityParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 999,
				Quantity:           50,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("UPDATE character_inventories SET quantity = \\$3, updated_at = NOW\\(\\) WHERE character_id = \\$1 AND item_id = \\$2").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(999), int32(50)).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			item, err := queries.UpdateInventoryItemQuantity(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, item)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestAddInventoryItemQuantity(t *testing.T) {
	tests := []struct {
		name        string
		params      AddInventoryItemQuantityParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, item CharacterInventory)
	}{
		{
			name: "successful quantity addition",
			params: AddInventoryItemQuantityParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 1,
				Quantity:           25, // Adding 25 to existing quantity
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				createdAt := now.Add(-time.Hour)
				rows := pgxmock.NewRows([]string{
					"id", "character_id", "item_id", "quantity", "created_at", "updated_at",
				}).AddRow(
					int32(1), "750e8400-e29b-41d4-a716-446655440000", int32(1), int32(125), // Assuming was 100, now 125
					pgtype.Timestamp{Time: createdAt, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("UPDATE character_inventories SET quantity = quantity \\+ \\$3, updated_at = NOW\\(\\) WHERE character_id = \\$1 AND item_id = \\$2").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(1), int32(25)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, item CharacterInventory) {
				assert.Equal(t, int32(125), item.Quantity)
				assert.True(t, item.UpdatedAt.Valid)
			},
		},
		{
			name: "add large quantity",
			params: AddInventoryItemQuantityParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 2,
				Quantity:           100000,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "character_id", "item_id", "quantity", "created_at", "updated_at",
				}).AddRow(
					int32(2), "750e8400-e29b-41d4-a716-446655440000", int32(2), int32(100000), 
					pgtype.Timestamp{Time: now.Add(-time.Hour), Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("UPDATE character_inventories SET quantity = quantity \\+ \\$3, updated_at = NOW\\(\\) WHERE character_id = \\$1 AND item_id = \\$2").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(2), int32(100000)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, item CharacterInventory) {
				assert.Equal(t, int32(100000), item.Quantity)
			},
		},
		{
			name: "add to non-existent inventory item",
			params: AddInventoryItemQuantityParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 999,
				Quantity:           10,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("UPDATE character_inventories SET quantity = quantity \\+ \\$3, updated_at = NOW\\(\\) WHERE character_id = \\$1 AND item_id = \\$2").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(999), int32(10)).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			item, err := queries.AddInventoryItemQuantity(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, item)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestRemoveInventoryItemQuantity(t *testing.T) {
	tests := []struct {
		name        string
		params      RemoveInventoryItemQuantityParams
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		checkResult func(t *testing.T, item CharacterInventory)
	}{
		{
			name: "successful quantity removal",
			params: RemoveInventoryItemQuantityParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 1,
				Quantity:           20, // Removing 20 from existing quantity
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				createdAt := now.Add(-time.Hour)
				rows := pgxmock.NewRows([]string{
					"id", "character_id", "item_id", "quantity", "created_at", "updated_at",
				}).AddRow(
					int32(1), "750e8400-e29b-41d4-a716-446655440000", int32(1), int32(80), // Assuming was 100, now 80
					pgtype.Timestamp{Time: createdAt, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("UPDATE character_inventories SET quantity = quantity - \\$3, updated_at = NOW\\(\\) WHERE character_id = \\$1 AND item_id = \\$2 AND quantity >= \\$3").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(1), int32(20)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, item CharacterInventory) {
				assert.Equal(t, int32(80), item.Quantity)
				assert.True(t, item.UpdatedAt.Valid)
			},
		},
		{
			name: "remove exact quantity - should result in zero",
			params: RemoveInventoryItemQuantityParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 1,
				Quantity:           50, // Removing all 50
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				now := time.Now()
				rows := pgxmock.NewRows([]string{
					"id", "character_id", "item_id", "quantity", "created_at", "updated_at",
				}).AddRow(
					int32(1), "750e8400-e29b-41d4-a716-446655440000", int32(1), int32(0), 
					pgtype.Timestamp{Time: now.Add(-time.Hour), Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
				)
				mock.ExpectQuery("UPDATE character_inventories SET quantity = quantity - \\$3, updated_at = NOW\\(\\) WHERE character_id = \\$1 AND item_id = \\$2 AND quantity >= \\$3").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(1), int32(50)).
					WillReturnRows(rows)
			},
			wantErr: false,
			checkResult: func(t *testing.T, item CharacterInventory) {
				assert.Equal(t, int32(0), item.Quantity)
			},
		},
		{
			name: "insufficient quantity - cannot remove more than available",
			params: RemoveInventoryItemQuantityParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 1,
				Quantity:           200, // Trying to remove more than available
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("UPDATE character_inventories SET quantity = quantity - \\$3, updated_at = NOW\\(\\) WHERE character_id = \\$1 AND item_id = \\$2 AND quantity >= \\$3").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(1), int32(200)).
					WillReturnError(sql.ErrNoRows) // No rows affected due to quantity constraint
			},
			wantErr: true,
		},
		{
			name: "remove from non-existent inventory item",
			params: RemoveInventoryItemQuantityParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 999,
				Quantity:           10,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("UPDATE character_inventories SET quantity = quantity - \\$3, updated_at = NOW\\(\\) WHERE character_id = \\$1 AND item_id = \\$2 AND quantity >= \\$3").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(999), int32(10)).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			item, err := queries.RemoveInventoryItemQuantity(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, item)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestDeleteInventoryItem(t *testing.T) {
	tests := []struct {
		name      string
		params    DeleteInventoryItemParams
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   bool
	}{
		{
			name: "successful inventory item deletion",
			params: DeleteInventoryItemParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 1,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM character_inventories WHERE character_id = \\$1 AND item_id = \\$2").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(1)).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name: "delete non-existent inventory item",
			params: DeleteInventoryItemParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 999,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM character_inventories WHERE character_id = \\$1 AND item_id = \\$2").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(999)).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: false, // DELETE doesn't fail on non-existent records
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			err = queries.DeleteInventoryItem(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestDeleteEmptyInventoryItems(t *testing.T) {
	tests := []struct {
		name        string
		characterID pgtype.UUID
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
	}{
		{
			name:        "successful empty items cleanup",
			characterID: mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM character_inventories WHERE character_id = \\$1 AND quantity <= 0").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000")).
					WillReturnResult(pgxmock.NewResult("DELETE", 3)) // Deleted 3 empty items
			},
			wantErr: false,
		},
		{
			name:        "no empty items to delete",
			characterID: mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM character_inventories WHERE character_id = \\$1 AND quantity <= 0").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000")).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			err = queries.DeleteEmptyInventoryItems(createTestContext(), tt.characterID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestInventoryItemExists(t *testing.T) {
	tests := []struct {
		name      string
		params    InventoryItemExistsParams
		setupMock func(mock pgxmock.PgxPoolIface)
		wantErr   bool
		expected  bool
	}{
		{
			name: "inventory item exists",
			params: InventoryItemExistsParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 1,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(1)).
					WillReturnRows(rows)
			},
			wantErr:  false,
			expected: true,
		},
		{
			name: "inventory item does not exist",
			params: InventoryItemExistsParams{
				CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
				ItemID: 999,
			},
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery("SELECT EXISTS").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(999)).
					WillReturnRows(rows)
			},
			wantErr:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			exists, err := queries.InventoryItemExists(createTestContext(), tt.params)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, exists)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetInventoryItemCount(t *testing.T) {
	tests := []struct {
		name        string
		characterID pgtype.UUID
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		expected    int64
	}{
		{
			name:        "character with multiple inventory items",
			characterID: mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"count"}).AddRow(int64(5))
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM character_inventories WHERE character_id = \\$1").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000")).
					WillReturnRows(rows)
			},
			wantErr:  false,
			expected: 5,
		},
		{
			name:        "character with empty inventory",
			characterID: mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"count"}).AddRow(int64(0))
				mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM character_inventories WHERE character_id = \\$1").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000")).
					WillReturnRows(rows)
			},
			wantErr:  false,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			count, err := queries.GetInventoryItemCount(createTestContext(), tt.characterID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, count)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

func TestGetInventoryItemTotalQuantity(t *testing.T) {
	tests := []struct {
		name        string
		characterID pgtype.UUID
		setupMock   func(mock pgxmock.PgxPoolIface)
		wantErr     bool
		expected    interface{}
	}{
		{
			name:        "character with inventory items",
			characterID: mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"coalesce"}).AddRow(int64(275)) // Sum of all quantities
				mock.ExpectQuery("SELECT COALESCE\\(SUM\\(quantity\\), 0\\) FROM character_inventories WHERE character_id = \\$1").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000")).
					WillReturnRows(rows)
			},
			wantErr:  false,
			expected: int64(275),
		},
		{
			name:        "character with empty inventory",
			characterID: mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
			setupMock: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"coalesce"}).AddRow(int64(0))
				mock.ExpectQuery("SELECT COALESCE\\(SUM\\(quantity\\), 0\\) FROM character_inventories WHERE character_id = \\$1").
					WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000")).
					WillReturnRows(rows)
			},
			wantErr:  false,
			expected: int64(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPool, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mockPool.Close()

			queries := New(mockPool)
			tt.setupMock(mockPool)

			totalQuantity, err := queries.GetInventoryItemTotalQuantity(createTestContext(), tt.characterID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, totalQuantity)
			}

			assert.NoError(t, mockPool.ExpectationsWereMet())
		})
	}
}

// Edge case tests for business logic validation
func TestInventoryBusinessLogic(t *testing.T) {
	t.Run("inventory quantity boundaries", func(t *testing.T) {
		// Test with maximum int32 quantity
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		params := CreateInventoryItemParams{
			CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
			ItemID: 1,
			Quantity:           2147483647, // Max int32
		}

		now := time.Now()
		rows := pgxmock.NewRows([]string{
			"id", "character_id", "item_id", "quantity", "created_at", "updated_at",
		}).AddRow(
			int32(1), "750e8400-e29b-41d4-a716-446655440000", int32(1), int32(2147483647), 
			pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
		)

		mockPool.ExpectQuery("INSERT INTO character_inventories").
			WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(1), int32(2147483647)).
			WillReturnRows(rows)

		item, err := queries.CreateInventoryItem(createTestContext(), params)
		require.NoError(t, err)
		assert.Equal(t, int32(2147483647), item.Quantity)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("concurrent inventory operations", func(t *testing.T) {
		// Simulate race condition where two operations try to modify the same inventory item
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		params := RemoveInventoryItemQuantityParams{
			CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
			ItemID: 1,
			Quantity:           50,
		}

		// Simulate concurrent modification - quantity constraint not met
		mockPool.ExpectQuery("UPDATE character_inventories SET quantity = quantity - \\$3, updated_at = NOW\\(\\) WHERE character_id = \\$1 AND item_id = \\$2 AND quantity >= \\$3").
			WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(1), int32(50)).
			WillReturnError(sql.ErrNoRows) // Another process already modified the quantity

		_, err = queries.RemoveInventoryItemQuantity(createTestContext(), params)
		assert.Error(t, err) // Should fail due to concurrent modification

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})

	t.Run("negative resource node type IDs", func(t *testing.T) {
		// Test that negative resource node type IDs are handled
		mockPool, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mockPool.Close()

		queries := New(mockPool)

		params := CreateInventoryItemParams{
			CharacterID:        mustParseUUID("750e8400-e29b-41d4-a716-446655440000"),
			ItemID: -1, // Negative resource type ID
			Quantity:           10,
		}

		now := time.Now()
		rows := pgxmock.NewRows([]string{
			"id", "character_id", "item_id", "quantity", "created_at", "updated_at",
		}).AddRow(
			int32(1), "750e8400-e29b-41d4-a716-446655440000", int32(-1), int32(10), 
			pgtype.Timestamp{Time: now, Valid: true}, pgtype.Timestamp{Time: now, Valid: true},
		)

		mockPool.ExpectQuery("INSERT INTO character_inventories").
			WithArgs(mustParseUUID("750e8400-e29b-41d4-a716-446655440000"), int32(-1), int32(10)).
			WillReturnRows(rows)

		item, err := queries.CreateInventoryItem(createTestContext(), params)
		require.NoError(t, err)
		assert.Equal(t, int32(-1), item.ItemID)

		assert.NoError(t, mockPool.ExpectationsWereMet())
	})
}

// Benchmark tests for performance validation
func BenchmarkCreateInventoryItem(b *testing.B) {
	b.Skip("Benchmark tests require real database connection")
}

func BenchmarkGetCharacterInventory(b *testing.B) {
	b.Skip("Benchmark tests require real database connection")
}

func BenchmarkAddInventoryItemQuantity(b *testing.B) {
	b.Skip("Benchmark tests require real database connection")
}