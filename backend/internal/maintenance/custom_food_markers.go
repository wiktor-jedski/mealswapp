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
	backoff := interval
	if backoff > time.Second {
		backoff = time.Second
	}
	for {
		err := PurgeDeletedCustomFoodCreateKeys(ctx, repo)
		wait := interval
		if err != nil {
			wait = backoff
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil
		case <-timer.C:
		}
	}
}
