// Package maintenance owns scheduled application maintenance tasks.
package maintenance

import (
	"context"
	"time"

	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

// PurgeDeletedCustomFoodCreateKeys runs one marker purge through the application boundary.
// Implements DESIGN-008 AccountDeleter marker retention.
func PurgeDeletedCustomFoodCreateKeys(ctx context.Context, repo repository.CustomFoodItemMaintenanceRepository) error {
	if repo == nil {
		return repository.NewError(repository.ErrorKindValidation, "custom food maintenance repository is required", nil)
	}
	return repo.PurgeExpiredDeletedCustomFoodCreateKeys(ctx)
}

// RunDeletedCustomFoodCreateKeyPurger schedules bounded marker purges until cancellation.
// Implements DESIGN-008 AccountDeleter marker retention scheduler.
func RunDeletedCustomFoodCreateKeyPurger(ctx context.Context, repo repository.CustomFoodItemMaintenanceRepository, interval time.Duration) error {
	if interval <= 0 {
		return repository.NewError(repository.ErrorKindValidation, "maintenance interval must be positive", nil)
	}
	if err := PurgeDeletedCustomFoodCreateKeys(ctx, repo); err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := PurgeDeletedCustomFoodCreateKeys(ctx, repo); err != nil {
				return err
			}
		}
	}
}
