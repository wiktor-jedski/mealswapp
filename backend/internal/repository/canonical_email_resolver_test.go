package repository

// Implements DESIGN-006 AuthController, DESIGN-009 UserAdminPanel, and DESIGN-013 InputNormalizer collision-safe resolution verification.

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type resolverIdentity struct {
	id uuid.UUID
}

func TestResolveCanonicalEmailIdentity(t *testing.T) {
	ctx := context.Background()
	canonical := LookupDigest{KeyVersion: "v1", Value: "canonical"}
	legacy := LookupDigest{KeyVersion: "v1", Value: "legacy"}
	notFound := NewError(ErrorKindNotFound, "missing", nil)

	t.Run("canonical only", func(t *testing.T) {
		id := uuid.New()
		identity, err := ResolveCanonicalEmailIdentity(ctx, canonical, nil, func(context.Context, LookupDigest) (resolverIdentity, error) {
			return resolverIdentity{id: id}, nil
		}, func(identity resolverIdentity) uuid.UUID { return identity.id }, func(context.Context, uuid.UUID, LookupDigest) error {
			t.Fatal("unexpected reindex")
			return nil
		})
		if err != nil || identity.id != id {
			t.Fatalf("identity=%+v err=%v", identity, err)
		}
	})

	t.Run("legacy reindex", func(t *testing.T) {
		id := uuid.New()
		var reindexed uuid.UUID
		identity, err := ResolveCanonicalEmailIdentity(ctx, canonical, &legacy, func(_ context.Context, digest LookupDigest) (resolverIdentity, error) {
			if digest == canonical {
				return resolverIdentity{}, notFound
			}
			return resolverIdentity{id: id}, nil
		}, func(identity resolverIdentity) uuid.UUID { return identity.id }, func(_ context.Context, userID uuid.UUID, digest LookupDigest) error {
			reindexed = userID
			if digest != canonical {
				t.Fatalf("reindex digest=%+v", digest)
			}
			return nil
		})
		if err != nil || identity.id != id || reindexed != id {
			t.Fatalf("identity=%+v reindexed=%s err=%v", identity, reindexed, err)
		}
	})

	t.Run("collision", func(t *testing.T) {
		canonicalID, legacyID := uuid.New(), uuid.New()
		_, err := ResolveCanonicalEmailIdentity(ctx, canonical, &legacy, func(_ context.Context, digest LookupDigest) (resolverIdentity, error) {
			if digest == canonical {
				return resolverIdentity{id: canonicalID}, nil
			}
			return resolverIdentity{id: legacyID}, nil
		}, func(identity resolverIdentity) uuid.UUID { return identity.id }, func(context.Context, uuid.UUID, LookupDigest) error {
			return nil
		})
		if !errors.Is(err, ErrCanonicalEmailCollision) {
			t.Fatalf("error=%v", err)
		}
	})

	t.Run("same identity and missing legacy", func(t *testing.T) {
		id := uuid.New()
		for _, legacyResult := range []error{nil, notFound} {
			identity, err := ResolveCanonicalEmailIdentity(ctx, canonical, &legacy, func(_ context.Context, digest LookupDigest) (resolverIdentity, error) {
				if digest == legacy && legacyResult != nil {
					return resolverIdentity{}, legacyResult
				}
				return resolverIdentity{id: id}, nil
			}, func(identity resolverIdentity) uuid.UUID { return identity.id }, func(context.Context, uuid.UUID, LookupDigest) error {
				t.Fatal("unexpected reindex")
				return nil
			})
			if err != nil || identity.id != id {
				t.Fatalf("identity=%+v err=%v", identity, err)
			}
		}
	})

	t.Run("lookup errors", func(t *testing.T) {
		unavailable := errors.New("lookup unavailable")
		for _, failureDigest := range []LookupDigest{canonical, legacy} {
			_, err := ResolveCanonicalEmailIdentity(ctx, canonical, &legacy, func(_ context.Context, digest LookupDigest) (resolverIdentity, error) {
				if digest == failureDigest {
					return resolverIdentity{}, unavailable
				}
				if failureDigest == legacy {
					return resolverIdentity{id: uuid.New()}, nil
				}
				return resolverIdentity{}, notFound
			}, func(identity resolverIdentity) uuid.UUID { return identity.id }, func(context.Context, uuid.UUID, LookupDigest) error {
				return nil
			})
			if !errors.Is(err, unavailable) {
				t.Fatalf("failure digest=%+v error=%v", failureDigest, err)
			}
		}
	})

	t.Run("reindex conflict", func(t *testing.T) {
		_, err := ResolveCanonicalEmailIdentity(ctx, canonical, &legacy, func(_ context.Context, digest LookupDigest) (resolverIdentity, error) {
			if digest == canonical {
				return resolverIdentity{}, notFound
			}
			return resolverIdentity{id: uuid.New()}, nil
		}, func(identity resolverIdentity) uuid.UUID { return identity.id }, func(context.Context, uuid.UUID, LookupDigest) error {
			return NewError(ErrorKindConflict, "conflict", nil)
		})
		if !errors.Is(err, ErrCanonicalEmailCollision) {
			t.Fatalf("error=%v", err)
		}
	})

	t.Run("reindex error", func(t *testing.T) {
		unavailable := errors.New("reindex unavailable")
		_, err := ResolveCanonicalEmailIdentity(ctx, canonical, &legacy, func(_ context.Context, digest LookupDigest) (resolverIdentity, error) {
			if digest == canonical {
				return resolverIdentity{}, notFound
			}
			return resolverIdentity{id: uuid.New()}, nil
		}, func(identity resolverIdentity) uuid.UUID { return identity.id }, func(context.Context, uuid.UUID, LookupDigest) error {
			return unavailable
		})
		if !errors.Is(err, unavailable) {
			t.Fatalf("error=%v", err)
		}
	})
}
