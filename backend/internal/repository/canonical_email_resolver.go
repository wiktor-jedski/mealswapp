package repository

import (
	"context"

	"github.com/google/uuid"
)

// ResolveCanonicalEmailIdentity resolves canonical and exact legacy digests without silently merging accounts.
// Implements DESIGN-006 AuthController, DESIGN-009 UserAdminPanel, and DESIGN-013 InputNormalizer.
func ResolveCanonicalEmailIdentity[T any](
	ctx context.Context,
	canonicalDigest LookupDigest,
	legacyDigest *LookupDigest,
	lookup func(context.Context, LookupDigest) (T, error),
	identityID func(T) uuid.UUID,
	reindex func(context.Context, uuid.UUID, LookupDigest) error,
) (T, error) {
	canonicalIdentity, canonicalErr := lookup(ctx, canonicalDigest)
	if legacyDigest == nil {
		return canonicalIdentity, canonicalErr
	}
	legacyIdentity, legacyErr := lookup(ctx, *legacyDigest)
	if canonicalErr == nil && legacyErr == nil {
		if identityID(canonicalIdentity) != identityID(legacyIdentity) {
			var zero T
			return zero, ErrCanonicalEmailCollision
		}
		return canonicalIdentity, nil
	}
	if canonicalErr == nil {
		if !IsKind(legacyErr, ErrorKindNotFound) {
			var zero T
			return zero, legacyErr
		}
		return canonicalIdentity, nil
	}
	if !IsKind(canonicalErr, ErrorKindNotFound) {
		var zero T
		return zero, canonicalErr
	}
	if legacyErr != nil {
		var zero T
		return zero, legacyErr
	}
	if err := reindex(ctx, identityID(legacyIdentity), canonicalDigest); err != nil {
		var zero T
		if IsKind(err, ErrorKindConflict) {
			return zero, ErrCanonicalEmailCollision
		}
		return zero, err
	}
	return legacyIdentity, nil
}
