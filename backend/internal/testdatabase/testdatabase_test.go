package testdatabase

import "testing"

// TestConfiguredURLIgnoresApplicationDatabase locks down the original data-loss path.
// Implements DESIGN-005 RepositoryInterfaces isolated integration-test persistence.
func TestConfiguredURLIgnoresApplicationDatabase(t *testing.T) {
	t.Setenv("MEALSWAPP_DATABASE_URL", "postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable")
	t.Setenv(EnvironmentVariable, "")
	if got := configuredURL(); got != DefaultURL {
		t.Fatalf("configuredURL() = %q, want dedicated default %q", got, DefaultURL)
	}
}

// TestValidateURLRejectsDevelopmentDatabase locks down the destructive-test regression.
// Implements DESIGN-005 RepositoryInterfaces isolated integration-test persistence.
func TestValidateURLRejectsDevelopmentDatabase(t *testing.T) {
	unsafe := []string{
		"postgres://mealswapp:mealswapp@localhost:5432/mealswapp?sslmode=disable",
		"postgresql://example.test/production",
		"https://example.test/mealswapp_test",
	}
	for _, databaseURL := range unsafe {
		if _, _, err := validateURL(databaseURL); err == nil {
			t.Fatalf("validateURL(%q) error = nil, want unsafe database rejection", databaseURL)
		}
	}
}

// TestValidateURLAcceptsDedicatedTestDatabase verifies custom isolated database support.
// Implements DESIGN-005 RepositoryInterfaces isolated integration-test persistence.
func TestValidateURLAcceptsDedicatedTestDatabase(t *testing.T) {
	const databaseURL = "postgres://user:password@example.test:5432/mealswapp_ci_test?sslmode=require"
	gotURL, gotName, err := validateURL(databaseURL)
	if err != nil {
		t.Fatalf("validateURL() error = %v", err)
	}
	if gotURL != databaseURL || gotName != "mealswapp_ci_test" {
		t.Fatalf("validateURL() = (%q, %q), want (%q, mealswapp_ci_test)", gotURL, gotName, databaseURL)
	}
}
