package exporter

import (
	"context"
	"encoding/json"
	"time"

	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
)

type Source interface {
	GetUser(ctx context.Context, userID uuid.UUID) (repositories.UserEntity, error)
	GetPreferences(ctx context.Context, userID uuid.UUID) (repositories.PreferenceEntity, error)
	GetEntitlement(ctx context.Context, userID uuid.UUID) (repositories.EntitlementEntity, error)
	ListSavedData(ctx context.Context, userID uuid.UUID) ([]repositories.SavedDataEntity, error)
	ListConsentRecords(ctx context.Context, userID uuid.UUID) ([]repositories.ConsentRecordEntity, error)
}

type Exporter struct {
	source Source
	now    func() time.Time
}

type Export struct {
	ExportedAt  time.Time                          `json:"exportedAt"`
	Profile     ProfileExport                      `json:"profile"`
	Preferences repositories.PreferenceEntity      `json:"preferences"`
	Entitlement repositories.EntitlementEntity     `json:"entitlement"`
	SavedData   []repositories.SavedDataEntity     `json:"savedData"`
	Consents    []repositories.ConsentRecordEntity `json:"consents"`
}

type ProfileExport struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	Role        string    `json:"role"`
	Disabled    bool      `json:"disabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func New(source Source) Exporter {
	return Exporter{source: source, now: time.Now}
}

func (exporter Exporter) ExportUserData(ctx context.Context, userID uuid.UUID) (Export, error) {
	user, err := exporter.source.GetUser(ctx, userID)
	if err != nil {
		return Export{}, err
	}
	preferences, err := exporter.source.GetPreferences(ctx, userID)
	if err != nil {
		return Export{}, err
	}
	entitlement, err := exporter.source.GetEntitlement(ctx, userID)
	if err != nil {
		return Export{}, err
	}
	savedData, err := exporter.source.ListSavedData(ctx, userID)
	if err != nil {
		return Export{}, err
	}
	savedData = redactSavedPayloads(savedData)
	consents, err := exporter.source.ListConsentRecords(ctx, userID)
	if err != nil {
		return Export{}, err
	}

	return Export{
		ExportedAt: exporter.now().UTC(),
		Profile: ProfileExport{
			ID:          user.ID,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			Role:        user.Role,
			Disabled:    user.Disabled,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
		},
		Preferences: preferences,
		Entitlement: entitlement,
		SavedData:   savedData,
		Consents:    consents,
	}, nil
}

func redactSavedPayloads(items []repositories.SavedDataEntity) []repositories.SavedDataEntity {
	redacted := make([]repositories.SavedDataEntity, len(items))
	for i, item := range items {
		redacted[i] = item
		redacted[i].Payload = redactJSONPayload(item.Payload)
	}
	return redacted
}

func redactJSONPayload(payload []byte) []byte {
	if len(payload) == 0 {
		return payload
	}

	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		return payload
	}
	value = redactValue(value)
	redacted, err := json.Marshal(value)
	if err != nil {
		return payload
	}
	return redacted
}

func redactValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if sensitiveKey(key) {
				typed[key] = "[redacted]"
				continue
			}
			typed[key] = redactValue(child)
		}
		return typed
	case []any:
		for i, child := range typed {
			typed[i] = redactValue(child)
		}
		return typed
	default:
		return value
	}
}

func sensitiveKey(key string) bool {
	switch key {
	case "token", "accessToken", "refreshToken", "password", "passwordHash":
		return true
	default:
		return false
	}
}
