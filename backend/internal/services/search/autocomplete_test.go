package search

import (
	"fmt"
	"testing"
	"time"
)

func TestRankAutocompleteOrdersByExactDistanceAndLength(t *testing.T) {
	candidates := []AutocompleteCandidate{
		{ItemID: "3", Label: "tofu scramble"},
		{ItemID: "2", Label: "tof"},
		{ItemID: "1", Label: "Tofu"},
		{ItemID: "4", Label: "tofu firm"},
	}

	ranked := RankAutocomplete(" tofu ", candidates)
	labels := labelsOf(ranked)
	expected := []string{"Tofu", "tof", "tofu firm", "tofu scramble"}
	assertLabels(t, labels, expected)

	if !ranked[0].ExactMatch || ranked[0].LevenshteinDistance != 0 || ranked[0].Length != 4 || ranked[0].Rank != 1 {
		t.Fatalf("unexpected exact-match metadata: %#v", ranked[0])
	}
}

func TestRankAutocompleteBreaksTiesDeterministically(t *testing.T) {
	candidates := []AutocompleteCandidate{
		{ItemID: "b", Label: "abd"},
		{ItemID: "d", Label: "aac"},
		{ItemID: "c", Label: "aac"},
		{ItemID: "a", Label: "abe"},
	}

	ranked := RankAutocompleteLimit("abc", candidates, 10)
	expectedLabels := []string{"aac", "aac", "abd", "abe"}
	assertLabels(t, labelsOf(ranked), expectedLabels)
	if ranked[0].ItemID != "c" || ranked[1].ItemID != "d" {
		t.Fatalf("expected same-label tie to sort by item id, got %#v", ranked[:2])
	}
}

func TestRankAutocompleteNormalizesCasingAndWhitespace(t *testing.T) {
	candidates := []AutocompleteCandidate{
		{ItemID: "2", Label: "Red   Lentils"},
		{ItemID: "1", Label: "red lentil"},
	}

	ranked := RankAutocomplete(" RED lentils ", candidates)
	if len(ranked) != 2 {
		t.Fatalf("expected two ranked candidates, got %d", len(ranked))
	}
	if ranked[0].Label != "Red   Lentils" || !ranked[0].ExactMatch {
		t.Fatalf("expected normalized exact match first, got %#v", ranked[0])
	}
}

func TestRankAutocompleteAppliesLimitAndSkipsEmptyInput(t *testing.T) {
	candidates := []AutocompleteCandidate{
		{ItemID: "1", Label: "tofu"},
		{ItemID: "2", Label: "tofu firm"},
		{ItemID: "3", Label: "   "},
	}

	ranked := RankAutocompleteLimit("tofu", candidates, 1)
	if len(ranked) != 1 || ranked[0].Label != "tofu" {
		t.Fatalf("unexpected limited result: %#v", ranked)
	}
	if empty := RankAutocompleteLimit(" ", candidates, 10); len(empty) != 0 {
		t.Fatalf("expected empty query to return no suggestions, got %#v", empty)
	}
	if empty := RankAutocompleteLimit("tofu", candidates, 0); len(empty) != 0 {
		t.Fatalf("expected zero limit to return no suggestions, got %#v", empty)
	}
}

func TestRankAutocompleteLatencyOnSeededData(t *testing.T) {
	candidates := make([]AutocompleteCandidate, 0, 5000)
	for i := 0; i < 5000; i++ {
		candidates = append(candidates, AutocompleteCandidate{
			ItemID: fmt.Sprintf("%06d", i),
			Label:  fmt.Sprintf("seeded ingredient %04d", i),
		})
	}
	candidates = append(candidates, AutocompleteCandidate{ItemID: "exact", Label: "tomato"})

	started := time.Now()
	ranked := RankAutocomplete("tomato", candidates)
	elapsed := time.Since(started)

	if len(ranked) == 0 || ranked[0].ItemID != "exact" {
		t.Fatalf("expected exact seeded candidate first, got %#v", ranked)
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("autocomplete ranking exceeded 100ms target: %s", elapsed)
	}
}

func labelsOf(ranked []RankedAutocomplete) []string {
	labels := make([]string, 0, len(ranked))
	for _, suggestion := range ranked {
		labels = append(labels, suggestion.Label)
	}
	return labels
}

func assertLabels(t *testing.T, actual []string, expected []string) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("expected labels %v, got %v", expected, actual)
	}
	for i := range expected {
		if actual[i] != expected[i] {
			t.Fatalf("expected labels %v, got %v", expected, actual)
		}
	}
}
