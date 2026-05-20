package similarity

import "testing"

func TestMapSimilarityIndicatorBoundaries(t *testing.T) {
	cases := []struct {
		name  string
		score float64
		tier  IndicatorTier
	}{
		{name: "green at 85", score: 0.85, tier: IndicatorGreen},
		{name: "green above 85", score: 0.95, tier: IndicatorGreen},
		{name: "light green at 70", score: 0.70, tier: IndicatorLightGreen},
		{name: "light green below 85", score: 0.849, tier: IndicatorLightGreen},
		{name: "yellow at 55", score: 0.55, tier: IndicatorYellow},
		{name: "yellow below 70", score: 0.699, tier: IndicatorYellow},
		{name: "red below 55", score: 0.549, tier: IndicatorRed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MapSimilarityIndicator(tc.score)
			if got.Tier != tc.tier {
				t.Fatalf("expected tier %s, got %#v", tc.tier, got)
			}
			if got.Label == "" {
				t.Fatal("expected display label")
			}
			if got.AssetURL != IndicatorAssetURL(tc.tier) {
				t.Fatalf("expected asset URL for tier %s, got %#v", tc.tier, got)
			}
		})
	}
}
