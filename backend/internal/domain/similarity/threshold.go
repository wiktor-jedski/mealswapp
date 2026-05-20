package similarity

const (
	DefaultMinimumSimilarity = 0.40
	MaxSimilarityPageSize    = 10
)

type Page struct {
	Items      []ScoredCandidate
	Total      int
	Page       int
	PageSize   int
	TotalPages int
}

func FilterThreshold(scored []ScoredCandidate, minScore float64, page int, pageSize int) Page {
	if minScore == 0 {
		minScore = DefaultMinimumSimilarity
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > MaxSimilarityPageSize {
		pageSize = MaxSimilarityPageSize
	}

	filtered := make([]ScoredCandidate, 0, len(scored))
	for _, candidate := range scored {
		if candidate.Score >= minScore {
			filtered = append(filtered, candidate)
		}
	}

	total := len(filtered)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	return Page{
		Items:      filtered[start:end],
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}
