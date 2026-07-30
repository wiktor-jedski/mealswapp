package repository

import (
	"strings"
	"testing"
)

// Implements DESIGN-008 AccountDeleter permanent custom-item deletion verification.
func TestCustomFoodDeletionSQLContracts(t *testing.T) {
	for name, query := range map[string]string{
		"lock":        customFoodDeleteLockSQL,
		"references":  customFoodDeleteReferencesSQL,
		"hard delete": customFoodDeleteHardSQL,
	} {
		if strings.TrimSpace(query) == "" {
			t.Fatalf("%s SQL is empty", name)
		}
	}
	if !strings.Contains(strings.ToUpper(customFoodDeleteLockSQL), "FOR UPDATE") {
		t.Fatal("deletion must lock the private item row")
	}
	if !strings.Contains(customFoodDeleteReferencesSQL, "custom_food_item_id=$1") || !strings.Contains(customFoodDeleteReferencesSQL, "d.user_id=$2") {
		t.Fatal("deletion references must be owner-scoped to the item")
	}
	if !strings.Contains(customFoodDeleteRetryMarkersSQL, "response_body->>'id' = $2::text") {
		t.Fatal("retry markers must be limited to the deleted item")
	}
	if !strings.Contains(customFoodPurgeMarkersSQL, "expires_at <= now()") {
		t.Fatal("expired retry markers must have a purge path")
	}
}
