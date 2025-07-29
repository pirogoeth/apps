package seeder

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pirogoeth/apps/orba/database"
	"github.com/sirupsen/logrus"
)

func SeedDatabase(ctx context.Context, seeds *SeedConfig, db *database.DbWrapper) error {
	if seeds == nil {
		return nil
	}

	if err := seedSources(ctx, seeds.Sources, db); err != nil {
		return fmt.Errorf("could not seed sources: %w", err)
	}

	return nil
}

func seedSources(ctx context.Context, sources []*SourceSeedsConfig, db *database.DbWrapper) error {
	for _, source := range sources {
		logrus.Debugf("Seed source %s", source.Id)
		if _, err := db.Queries.CreateSource(ctx, database.CreateSourceParams{
			ID:          source.Id,
			Description: sql.NullString{String: source.Description},
		}); err != nil {
			// Is "already exists?"
			return fmt.Errorf("could not create seed source: %s: %w", source.Id, err)
		}
	}

	return nil
}
