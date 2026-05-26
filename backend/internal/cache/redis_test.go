package cache

// Implements DESIGN-011 RedisCache connection factory verification.

import "testing"

// TestOpenRejectsInvalidRedisURL verifies DESIGN-011 RedisCache invalid URL handling.
func TestOpenRejectsInvalidRedisURL(t *testing.T) {
	if _, err := Open("not a redis url"); err == nil {
		t.Fatal("Open() error = nil, want invalid URL error")
	}
}

// TestOpenAcceptsRedisURL verifies DESIGN-011 RedisCache URL parsing.
func TestOpenAcceptsRedisURL(t *testing.T) {
	client, err := Open("redis://localhost:6379/0")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer client.Close()
}
