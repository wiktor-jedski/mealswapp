package providerregistry

import (
	"errors"
	"testing"
)

// Implements DESIGN-012 DataNormalizer provider registry verification.
func TestDefaultRegistryCanonicalizesProviderIdentities(t *testing.T) {
	tests := []struct {
		provider string
		id       string
		want     Identity
	}{
		{provider: " USDA ", id: " 00171265 ", want: Identity{Provider: "usda", ExternalID: "171265"}},
		{provider: " OpenFoodFacts ", id: " 0012345678905 ", want: Identity{Provider: "openfoodfacts", ExternalID: "0012345678905"}},
	}
	for _, test := range tests {
		got, err := Default().Normalize(test.provider, test.id)
		if err != nil || got != test.want {
			t.Fatalf("Normalize(%q, %q) = %#v, %v; want %#v", test.provider, test.id, got, err, test.want)
		}
	}
}

func TestRegistryRejectsDuplicateAndInvalidRegistrations(t *testing.T) {
	normalizer := func(value string) (string, error) {
		if value == "fixture-1" {
			return value, nil
		}
		return "", errors.New("invalid fixture id")
	}
	registry, err := New(Definition{Name: "fixture", NormalizeExternalID: normalizer})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := registry.Normalize(" FIXTURE ", "fixture-1"); err != nil || got.Provider != "fixture" {
		t.Fatalf("fixture registration = %#v, %v", got, err)
	}
	if _, err := New(
		Definition{Name: "fixture", NormalizeExternalID: normalizer},
		Definition{Name: " FIXTURE ", NormalizeExternalID: normalizer},
	); err == nil {
		t.Fatal("duplicate registration accepted")
	}
	for _, identity := range []Identity{{Provider: "unknown", ExternalID: "1"}, {Provider: "usda", ExternalID: ""}, {Provider: "usda", ExternalID: "bad id"}} {
		if _, err := Default().Normalize(identity.Provider, identity.ExternalID); err == nil {
			t.Fatalf("invalid identity accepted: %#v", identity)
		}
	}
}
