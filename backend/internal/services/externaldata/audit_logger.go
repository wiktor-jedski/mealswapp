package externaldata

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"mealswapp/backend/internal/repositories"

	"github.com/google/uuid"
)

type AuditStore interface {
	Create(ctx context.Context, entry repositories.AuditLogEntity) (uuid.UUID, error)
}

type AuditLogger struct {
	store AuditStore
	now   func() time.Time
}

type AuditEvent struct {
	ActorID   *uuid.UUID     `json:"actorId,omitempty"`
	Action    string         `json:"action"`
	Target    string         `json:"target"`
	RequestID string         `json:"requestId"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type AuditEntryView struct {
	ID        uuid.UUID      `json:"id"`
	ActorID   *uuid.UUID     `json:"actorId,omitempty"`
	Action    string         `json:"action"`
	Target    string         `json:"target"`
	RequestID string         `json:"requestId"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

func NewAuditLogger(store AuditStore) AuditLogger {
	return AuditLogger{store: store, now: time.Now}
}

func (logger AuditLogger) Log(ctx context.Context, event AuditEvent) (AuditEntryView, error) {
	metadata := redactMetadata(event.Metadata)
	metadata["requestId"] = strings.TrimSpace(event.RequestID)
	metadata["recordedAt"] = logger.now().UTC().Format(time.RFC3339Nano)
	payload, err := json.Marshal(metadata)
	if err != nil {
		return AuditEntryView{}, err
	}
	id, err := logger.store.Create(ctx, repositories.AuditLogEntity{
		ActorID:  event.ActorID,
		Action:   strings.TrimSpace(event.Action),
		Target:   strings.TrimSpace(event.Target),
		Metadata: payload,
	})
	if err != nil {
		return AuditEntryView{}, err
	}
	return AuditEntryView{
		ID:        id,
		ActorID:   event.ActorID,
		Action:    strings.TrimSpace(event.Action),
		Target:    strings.TrimSpace(event.Target),
		RequestID: strings.TrimSpace(event.RequestID),
		Metadata:  metadata,
		CreatedAt: logger.now().UTC(),
	}, nil
}

func redactMetadata(input map[string]any) map[string]any {
	output := make(map[string]any, len(input)+2)
	for key, value := range input {
		if isSensitiveAuditKey(key) {
			output[key] = "[redacted]"
			continue
		}
		if nested, ok := value.(map[string]any); ok {
			output[key] = redactMetadata(nested)
			continue
		}
		output[key] = value
	}
	return output
}

func isSensitiveAuditKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	return strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "credential") ||
		strings.Contains(normalized, "hash")
}
