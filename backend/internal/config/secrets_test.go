package config

import (
	"context"
	"testing"
)

func TestLoadSecretReturnsRawLocalSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "local-secret")

	value, err := LoadSecret(context.Background(), "JWT_SECRET", nil)
	if err != nil {
		t.Fatal(err)
	}
	if value != "local-secret" {
		t.Fatalf("expected raw secret, got %q", value)
	}
}

func TestLoadSecretResolvesSecretManagerResource(t *testing.T) {
	t.Setenv("JWT_SECRET", "projects/mealswapp/secrets/jwt/versions/latest")
	manager := &fakeSecretManager{value: "resolved-secret"}

	value, err := LoadSecret(context.Background(), "JWT_SECRET", manager)
	if err != nil {
		t.Fatal(err)
	}
	if value != "resolved-secret" || manager.resource != "projects/mealswapp/secrets/jwt/versions/latest" {
		t.Fatalf("unexpected resolved secret value=%q resource=%q", value, manager.resource)
	}
}

type fakeSecretManager struct {
	value    string
	resource string
}

func (manager *fakeSecretManager) AccessSecret(ctx context.Context, resource string) (string, error) {
	manager.resource = resource
	return manager.value, nil
}
