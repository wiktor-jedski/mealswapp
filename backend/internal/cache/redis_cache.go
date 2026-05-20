package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

const (
	DefaultSchemaVersion = "v1"
	keyPrefix            = "mealswapp"
)

type Namespace string

const (
	NamespaceSearch     Namespace = "search"
	NamespaceItem       Namespace = "item"
	NamespaceSimilarity Namespace = "similarity"
	NamespaceSession    Namespace = "session"
	NamespaceJob        Namespace = "job"
	NamespaceUser       Namespace = "user"
)

type RedisCacheKey struct {
	Namespace Namespace `json:"namespace"`
	ID        string    `json:"id"`
	Version   string    `json:"version"`
}

func (key RedisCacheKey) String() string {
	version := strings.TrimSpace(key.Version)
	if version == "" {
		version = DefaultSchemaVersion
	}
	return keyPrefix + ":" + version + ":" + string(key.Namespace) + ":" + key.ID
}

func BuildKey(namespace Namespace, id string, version string) RedisCacheKey {
	return RedisCacheKey{
		Namespace: namespace,
		ID:        StableHash(strings.TrimSpace(id)),
		Version:   normalizedVersion(version),
	}
}

func BuildRawIDKey(namespace Namespace, id string, version string) RedisCacheKey {
	return RedisCacheKey{
		Namespace: namespace,
		ID:        normalizeID(id),
		Version:   normalizedVersion(version),
	}
}

func BuildSearchCacheKey(request any) (RedisCacheKey, error) {
	return BuildHashedPayloadKey(NamespaceSearch, request, DefaultSchemaVersion)
}

func BuildHashedPayloadKey(namespace Namespace, payload any, version string) (RedisCacheKey, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return RedisCacheKey{}, err
	}
	return RedisCacheKey{
		Namespace: namespace,
		ID:        StableHash(string(encoded)),
		Version:   normalizedVersion(version),
	}, nil
}

func StableHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func TTL(namespace Namespace) time.Duration {
	switch namespace {
	case NamespaceSearch:
		return 5 * time.Minute
	case NamespaceItem:
		return 30 * time.Minute
	case NamespaceSimilarity:
		return 15 * time.Minute
	case NamespaceSession:
		return 24 * time.Hour
	case NamespaceJob:
		return time.Hour
	case NamespaceUser:
		return 10 * time.Minute
	default:
		return 5 * time.Minute
	}
}

func TagsForKey(key RedisCacheKey, extraTags ...string) []string {
	tags := []string{"namespace:" + string(key.Namespace)}
	for _, tag := range extraTags {
		normalized := strings.TrimSpace(tag)
		if normalized != "" {
			tags = append(tags, normalized)
		}
	}
	return tags
}

func normalizedVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return DefaultSchemaVersion
	}
	return version
}

func normalizeID(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	id = strings.ReplaceAll(id, " ", "-")
	return id
}
