// Package providerregistry owns canonical external-food provider identities.
package providerregistry

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// Identity is a canonical provider and provider-owned food identifier.
// Implements DESIGN-012 DataNormalizer trusted external provenance.
type Identity struct {
	Provider   string
	ExternalID string
}

// Definition registers one provider and its identifier canonicalizer.
// Implements DESIGN-012 DataNormalizer external-data provider registry.
type Definition struct {
	Name                string
	NormalizeExternalID func(string) (string, error)
}

// Registry is an immutable set of uniquely named provider definitions.
// Implements DESIGN-012 DataNormalizer external-data provider registry.
type Registry struct {
	definitions map[string]Definition
}

// Implements DESIGN-012 DataNormalizer immutable default provider registry.
var (
	defaultOnce     sync.Once
	defaultRegistry *Registry
)

// New creates a registry and rejects duplicate or malformed registrations.
// Implements DESIGN-012 DataNormalizer provider registration boundary.
func New(definitions ...Definition) (*Registry, error) {
	registry := &Registry{definitions: make(map[string]Definition, len(definitions))}
	for _, definition := range definitions {
		name := strings.ToLower(strings.TrimSpace(definition.Name))
		if !validProviderName(name) || definition.NormalizeExternalID == nil {
			return nil, errors.New("external provider registration is invalid")
		}
		if _, exists := registry.definitions[name]; exists {
			return nil, errors.New("external provider is already registered")
		}
		definition.Name = name
		registry.definitions[name] = definition
	}
	return registry, nil
}

// Default returns the single application registry for USDA and OpenFoodFacts.
// Implements DESIGN-012 DataNormalizer one provider registration point.
func Default() *Registry {
	defaultOnce.Do(func() {
		defaultRegistry, _ = New(
			Definition{Name: "usda", NormalizeExternalID: normalizeUSDAIdentifier},
			Definition{Name: "openfoodfacts", NormalizeExternalID: normalizeOpenFoodFactsIdentifier},
		)
	})
	return defaultRegistry
}

// Normalize returns a canonical registered provider identity.
// Implements DESIGN-012 DataNormalizer provider and identifier normalization.
func (r *Registry) Normalize(provider, externalID string) (Identity, error) {
	if r == nil {
		return Identity{}, errors.New("external provider registry is unavailable")
	}
	name := strings.ToLower(strings.TrimSpace(provider))
	definition, exists := r.definitions[name]
	if !exists {
		return Identity{}, errors.New("external provider is not registered")
	}
	identifier, err := definition.NormalizeExternalID(externalID)
	if err != nil {
		return Identity{}, err
	}
	return Identity{Provider: name, ExternalID: identifier}, nil
}

// NormalizeProvider returns a canonical provider name without accepting an identifier.
// Implements DESIGN-012 DataNormalizer provider selection validation.
func (r *Registry) NormalizeProvider(provider string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(provider))
	if strings.NewReplacer("-", "_", " ", "_").Replace(name) == "open_food_facts" {
		name = "openfoodfacts"
	}
	if r == nil {
		return "", errors.New("external provider registry is unavailable")
	}
	if _, exists := r.definitions[name]; !exists {
		return "", errors.New("external provider is not registered")
	}
	return name, nil
}

// normalizeUSDAIdentifier canonicalizes USDA identifiers while preserving nonnumeric provider keys.
// Implements DESIGN-012 DataNormalizer USDA identifier normalization.
func normalizeUSDAIdentifier(value string) (string, error) {
	value, err := normalizeIdentifier(value)
	if err != nil {
		return "", err
	}
	if allDigits(value) {
		number, parseErr := strconv.ParseUint(value, 10, 64)
		if parseErr != nil {
			return "", errors.New("USDA food identifier is invalid")
		}
		return strconv.FormatUint(number, 10), nil
	}
	return value, nil
}

// normalizeOpenFoodFactsIdentifier canonicalizes OpenFoodFacts identifiers.
// Implements DESIGN-012 DataNormalizer OpenFoodFacts identifier normalization.
func normalizeOpenFoodFactsIdentifier(value string) (string, error) {
	return normalizeIdentifier(value)
}

// normalizeIdentifier enforces shared external identifier bounds.
// Implements DESIGN-012 DataNormalizer external identifier validation.
func normalizeIdentifier(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || utf8.RuneCountInString(value) > 200 {
		return "", errors.New("external food identifier is invalid")
	}
	for _, char := range value {
		if unicode.IsControl(char) || unicode.IsSpace(char) {
			return "", errors.New("external food identifier is invalid")
		}
	}
	return value, nil
}

// validProviderName reports whether a registration name is canonical and bounded.
// Implements DESIGN-012 DataNormalizer provider registration validation.
func validProviderName(value string) bool {
	if value == "" || len(value) > 64 {
		return false
	}
	for index, char := range value {
		if char < 'a' || char > 'z' {
			if index == 0 || char != '-' && (char < '0' || char > '9') {
				return false
			}
		}
	}
	return true
}

// allDigits reports whether an identifier can use numeric canonicalization.
// Implements DESIGN-012 DataNormalizer USDA identifier normalization.
func allDigits(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
