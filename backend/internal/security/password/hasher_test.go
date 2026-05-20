package password

import (
	"errors"
	"strings"
	"testing"
)

func TestHasherHashesAndVerifiesPassword(t *testing.T) {
	hasher := NewHasher(testParameters())

	hash, err := hasher.Hash("CorrectHorse1!")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Fatalf("expected argon2id PHC hash, got %q", hash)
	}

	ok, err := hasher.Verify("CorrectHorse1!", hash)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected password verification")
	}
}

func TestHasherRejectsInvalidPassword(t *testing.T) {
	hasher := NewHasher(testParameters())
	hash, err := hasher.Hash("CorrectHorse1!")
	if err != nil {
		t.Fatal(err)
	}

	ok, err := hasher.Verify("wrong-password", hash)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected invalid password")
	}
}

func TestHasherRejectsWeakPasswords(t *testing.T) {
	hasher := NewHasher(testParameters())

	_, err := hasher.Hash("short")
	if !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("expected weak password error, got %v", err)
	}

	_, err = hasher.Hash("longbutmissingclasses")
	if !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("expected weak password error, got %v", err)
	}
}

func TestHasherDetectsRehashNeed(t *testing.T) {
	oldHasher := NewHasher(Parameters{Memory: 4 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 8, KeyLength: 16})
	newHasher := NewHasher(testParameters())

	hash, err := oldHasher.Hash("CorrectHorse1!")
	if err != nil {
		t.Fatal(err)
	}

	needsRehash, err := newHasher.NeedsRehash(hash)
	if err != nil {
		t.Fatal(err)
	}
	if !needsRehash {
		t.Fatal("expected old parameters to need rehash")
	}

	currentHash, err := newHasher.Hash("CorrectHorse1!")
	if err != nil {
		t.Fatal(err)
	}
	needsRehash, err = newHasher.NeedsRehash(currentHash)
	if err != nil {
		t.Fatal(err)
	}
	if needsRehash {
		t.Fatal("expected current parameters to not need rehash")
	}
}

func TestHasherRejectsMalformedHash(t *testing.T) {
	hasher := NewHasher(testParameters())

	_, err := hasher.Verify("CorrectHorse1!", "not-a-hash")
	if !errors.Is(err, ErrInvalidHash) {
		t.Fatalf("expected invalid hash error, got %v", err)
	}
}

func testParameters() Parameters {
	return Parameters{Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 8, KeyLength: 16}
}
