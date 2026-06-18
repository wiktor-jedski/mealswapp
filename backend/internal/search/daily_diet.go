package search

// ErrDailyDietIDRequired reports a daily-diet alternative request missing its saved-diet seed.
// Implements DESIGN-002 QueryParser daily diet alternative boundary.
var ErrDailyDietIDRequired = errDailyDietIDRequired{}

// errDailyDietIDRequired identifies missing saved-diet seed input.
// Implements DESIGN-002 QueryParser daily diet alternative boundary.
type errDailyDietIDRequired struct{}

// Error returns the stable daily-diet validation message.
// Implements DESIGN-002 QueryParser daily diet alternative boundary.
func (errDailyDietIDRequired) Error() string {
	return "daily diet id is required for daily diet alternative search"
}

// PreparedSearch carries parsed query and repository-ready filters for API dispatch.
// Implements DESIGN-002 QueryParser daily diet alternative boundary.
type PreparedSearch struct {
	ParsedQuery ParsedQuery
	Filters     ProcessedFilters
	Rejection   *SearchRejection
}

// PrepareSearchRequest validates API-boundary search shape before repository or worker dispatch.
// Implements DESIGN-002 QueryParser daily diet alternative boundary.
func PrepareSearchRequest(req SearchRequest, dailyDietData DailyDietDataStatus) (PreparedSearch, error) {
	parsed, err := BuildParsedQuery(req)
	if err != nil {
		return PreparedSearch{}, err
	}
	if parsed.Strategy == SearchStrategyDailyDietAlternative {
		if req.DailyDietID == nil {
			return PreparedSearch{}, ErrDailyDietIDRequired
		}
		if dailyDietData != DailyDietDataAvailable {
			return PreparedSearch{
				ParsedQuery: parsed,
				Rejection: &SearchRejection{
					Code:    "phase_07_saved_diet_unavailable",
					Message: "Daily diet alternatives require saved diet data that is not available yet.",
					Field:   "dailyDietId",
				},
			}, nil
		}
	}
	filters, rejection := ApplyFilters(parsed, req.Filters)
	if rejection != nil {
		return PreparedSearch{ParsedQuery: parsed, Rejection: rejection}, nil
	}
	return PreparedSearch{ParsedQuery: parsed, Filters: filters}, nil
}
