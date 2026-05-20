package similarity

type IndicatorTier string

const (
	IndicatorGreen      IndicatorTier = "green"
	IndicatorLightGreen IndicatorTier = "light_green"
	IndicatorYellow     IndicatorTier = "yellow"
	IndicatorRed        IndicatorTier = "red"
)

type Indicator struct {
	Tier     IndicatorTier `json:"tier"`
	Label    string        `json:"label"`
	AssetURL string        `json:"assetUrl"`
}

func MapSimilarityIndicator(score float64) Indicator {
	switch {
	case score >= 0.85:
		return indicator(IndicatorGreen, "Excellent match")
	case score >= 0.70:
		return indicator(IndicatorLightGreen, "Good match")
	case score >= 0.55:
		return indicator(IndicatorYellow, "Moderate match")
	default:
		return indicator(IndicatorRed, "Low match")
	}
}

func indicator(tier IndicatorTier, label string) Indicator {
	return Indicator{
		Tier:     tier,
		Label:    label,
		AssetURL: IndicatorAssetURL(tier),
	}
}

func IndicatorAssetURL(tier IndicatorTier) string {
	return "/static/similarity/" + string(tier) + ".svg"
}
