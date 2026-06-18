package auth

// Implements DESIGN-006 PasswordHasher verification.

import (
	"strings"
	"testing"
)

func testPasswordHash() (string, string) {
	return "argon2id$v=19$m=19456,t=1,p=1$u67X4pB7vrPK0wZMLU3SXg", "dGVzdC1maXh0dXJlLXNhbHQ"
}

// TestPasswordHasherHashesAndVerifies verifies DESIGN-006 PasswordHasher behavior.
func TestPasswordHasherHashesAndVerifies(t *testing.T) {
	hasher, err := NewPasswordHasher(PasswordHashParams{MemoryKiB: 19 * 1024, Iterations: 1, Parallelism: 1, KeyLength: 32, SaltLength: 16, MinLength: 12})
	if err != nil {
		t.Fatal(err)
	}
	firstHash, firstSalt, err := hasher.HashPassword("StrongerPassword1!")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	secondHash, secondSalt, err := hasher.HashPassword("StrongerPassword1!")
	if err != nil {
		t.Fatalf("HashPassword() second error = %v", err)
	}
	if firstSalt == secondSalt || firstHash == secondHash {
		t.Fatalf("salt/hash reuse: %q %q", firstSalt, secondSalt)
	}
	if !hasher.VerifyPassword("StrongerPassword1!", firstHash, firstSalt) {
		t.Fatal("VerifyPassword() rejected valid password")
	}
	if hasher.VerifyPassword("WrongPassword1!", firstHash, firstSalt) {
		t.Fatal("VerifyPassword() accepted invalid password")
	}
}

// TestPasswordHasherRejectsMalformedInputs verifies DESIGN-006 PasswordHasher fail-closed parsing.
func TestPasswordHasherRejectsMalformedInputs(t *testing.T) {
	hasher, err := NewPasswordHasher(PasswordHashParams{MemoryKiB: 19 * 1024, Iterations: 1, Parallelism: 1, KeyLength: 32, SaltLength: 16, MinLength: 12})
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []struct {
		hash string
		salt string
	}{
		{"", ""},
		{"argon2i$v=19$m=19456,t=1,p=1$abcd", "salt"},
		{"argon2id$v=18$m=19456,t=1,p=1$abcd", "salt"},
		{"argon2id$v=19$m=0,t=1,p=1$abcd", "salt"},
		{"argon2id$v=19$m=19456,t=0,p=1$abcd", "salt"},
		{"argon2id$v=19$m=19456,t=1,p=0$abcd", "salt"},
		{"argon2id$v=19$m=19456,t=1,p=999$abcd", "salt"},
		{"argon2id$v=19$m=19456,t=1,p=1$not base64", "salt"},
		{"argon2id$v=19$m=19456,t=1,p=1$abcd", "not base64"},
	} {
		if hasher.VerifyPassword("StrongerPassword1!", input.hash, input.salt) {
			t.Fatalf("VerifyPassword() accepted malformed input %+v", input)
		}
	}
}

// TestPasswordHasherPolicyAndFixtures verifies DESIGN-006 PasswordHasher policy boundaries.
func TestPasswordHasherPolicyAndFixtures(t *testing.T) {
	if _, err := NewPasswordHasher(PasswordHashParams{MemoryKiB: 1, Iterations: 1, Parallelism: 1, KeyLength: 32, SaltLength: 16, MinLength: 12}); err == nil {
		t.Fatal("NewPasswordHasher() accepted weak memory")
	}
	hasher, err := NewPasswordHasher(PasswordHashParams{MemoryKiB: 19 * 1024, Iterations: 1, Parallelism: 1, KeyLength: 32, SaltLength: 16, MinLength: 14})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := hasher.HashPassword("Short1!"); err == nil || strings.Contains(err.Error(), "Short1") {
		t.Fatalf("HashPassword() policy error = %v", err)
	}
	fixtureHash, fixtureSalt := testPasswordHash()
	if !strings.HasPrefix(fixtureHash, "argon2id$v=19$") || fixtureSalt == "" {
		t.Fatalf("fixture hash/salt = %q %q", fixtureHash, fixtureSalt)
	}
}

// TestPasswordHasherRandomnessFailure verifies DESIGN-006 PasswordHasher salt generation failures.
func TestPasswordHasherRandomnessFailure(t *testing.T) {
	hasher, err := NewPasswordHasher(PasswordHashParams{MemoryKiB: 19 * 1024, Iterations: 1, Parallelism: 1, KeyLength: 32, SaltLength: 16, MinLength: 12})
	if err != nil {
		t.Fatal(err)
	}
	hasher.randomness = strings.NewReader("")
	if _, _, err := hasher.HashPassword("StrongerPassword1!"); err == nil {
		t.Fatal("HashPassword() accepted randomness failure")
	}
}

func TestDefaultPasswordHasherAndParserBoundaries(t *testing.T) {
	params := DefaultPasswordHashParams()
	if params.MemoryKiB == 0 || params.Iterations == 0 || params.SaltLength < 16 || NewDefaultPasswordHasher() == nil {
		t.Fatalf("default params = %+v", params)
	}
	hasher, err := NewPasswordHasher(PasswordHashParams{MemoryKiB: 19 * 1024, Iterations: 1, Parallelism: 1, KeyLength: 32, SaltLength: 16, MinLength: 12})
	if err != nil {
		t.Fatal(err)
	}
	hash, _ := testPasswordHash()
	if hasher.VerifyPassword("StrongerPassword1!", hash, "") {
		t.Fatal("VerifyPassword() accepted an empty salt")
	}
	for _, raw := range []string{
		"m=19456,t=1,bad",
		"m=bad,t=1,p=1",
		"m=19456,t=1,x=1",
	} {
		if _, err := parseHashParams(raw); err == nil {
			t.Fatalf("parseHashParams(%q) accepted", raw)
		}
	}
}
