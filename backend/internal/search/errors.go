package search

// SimilarityUnavailableError reports substitution ranking failures without leaking technical detail.
// Implements DESIGN-017 ErrorMessageMapper and DESIGN-002 SearchController similarity_unavailable state.
type SimilarityUnavailableError struct {
	Cause error
}

// Error returns a stable service-level failure code.
// Implements DESIGN-017 ErrorMessageMapper.
func (e SimilarityUnavailableError) Error() string {
	return "similarity_unavailable"
}

// Unwrap returns the technical cause for server-side diagnostics.
// Implements DESIGN-017 ErrorMessageMapper.
func (e SimilarityUnavailableError) Unwrap() error {
	return e.Cause
}
