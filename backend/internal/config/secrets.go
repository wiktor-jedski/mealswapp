package config

import (
	"context"
	"os"
	"strings"
)

type SecretManager interface {
	AccessSecret(ctx context.Context, resource string) (string, error)
}

func LoadSecret(ctx context.Context, envKey string, manager SecretManager) (string, error) {
	value := strings.TrimSpace(os.Getenv(envKey))
	if value == "" {
		return "", nil
	}
	if !isSecretManagerResource(value) {
		return value, nil
	}
	if manager == nil {
		return "", nil
	}
	return manager.AccessSecret(ctx, value)
}

func isSecretManagerResource(value string) bool {
	return strings.HasPrefix(value, "projects/") && strings.Contains(value, "/secrets/") && strings.Contains(value, "/versions/")
}
