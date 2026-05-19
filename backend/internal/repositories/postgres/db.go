package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Store struct {
	pool *pgxpool.Pool
}

type RepositorySet struct {
	FoodItems               FoodItemRepository
	Meals                   MealRepository
	Recipes                 RecipeRepository
	Tags                    TagRepository
	MicronutrientVocabulary MicronutrientVocabularyRepository
	Users                   UserRepository
	Preferences             PreferenceRepository
	Entitlements            EntitlementRepository
	SavedData               SavedDataRepository
	AuditLogs               AuditLogRepository
	Imports                 ImportRepository
}

func NewStore(pool *pgxpool.Pool) Store {
	return Store{pool: pool}
}

func (store Store) Repositories() RepositorySet {
	return repositorySetForDB(store.pool, store.pool)
}

func (store Store) WithTx(ctx context.Context, fn func(RepositorySet) error) error {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := fn(repositorySetForDB(tx, nil)); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func repositorySetForDB(db DBTX, pool *pgxpool.Pool) RepositorySet {
	return RepositorySet{
		FoodItems:               NewFoodItemRepositoryWithDB(db),
		Meals:                   NewMealRepositoryWithDB(db, pool),
		Recipes:                 NewRecipeRepositoryWithDB(db, pool),
		Tags:                    NewTagRepositoryWithDB(db),
		MicronutrientVocabulary: NewMicronutrientVocabularyRepositoryWithDB(db),
		Users:                   NewUserRepositoryWithDB(db),
		Preferences:             NewPreferenceRepositoryWithDB(db),
		Entitlements:            NewEntitlementRepositoryWithDB(db),
		SavedData:               NewSavedDataRepositoryWithDB(db),
		AuditLogs:               NewAuditLogRepositoryWithDB(db),
		Imports:                 NewImportRepositoryWithDB(db),
	}
}
