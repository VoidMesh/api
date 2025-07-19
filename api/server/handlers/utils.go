package handlers

import (
	"context"
	"strconv"

	"github.com/VoidMesh/platform/api/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

// getWorldSeed retrieves the world seed from database
func getWorldSeed(pool *pgxpool.Pool) (int64, error) {
	ctx := context.Background()
	setting, err := db.New(pool).GetWorldSetting(ctx, "seed")
	if err != nil {
		return 0, err
	}

	seed, err := strconv.ParseInt(setting.Value, 10, 64)
	if err != nil {
		return 0, err
	}

	return seed, nil
}
