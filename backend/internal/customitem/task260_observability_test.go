package customitem

// Implements DESIGN-008 ProfileController and DESIGN-014 MetricsCollector Task 260 gate.

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

func TestTask260CustomItemLifecycleTelemetryHasNoIdentityLabels(t *testing.T) {
	sink := &observability.MemorySink{}
	items := &memoryItems{items: map[uuid.UUID]repository.CustomFoodItemEntity{}}
	service := NewService(items).WithTelemetry(observability.NewAdminExternalTelemetry(sink, sink))
	owner := uuid.New()
	request := CreateRequest{Request: solidRequest("private item name"), IdempotencyKey: "private-idempotency-key"}
	created, err := service.Create(context.Background(), owner, request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(context.Background(), owner, request); err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(context.Background(), owner, created.Item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(context.Background(), owner, created.Item.ID); err == nil {
		t.Fatal("deleted item remained visible")
	}
	metrics, _ := sink.Snapshot()
	want := []string{"succeeded", "replayed", "succeeded", "not_found"}
	if len(metrics) != len(want) {
		t.Fatalf("metrics=%+v", metrics)
	}
	for index, point := range metrics {
		if len(point.Labels) != 2 || point.Labels["outcome"] != want[index] {
			t.Fatalf("metric[%d]=%+v", index, point)
		}
	}
}

func TestTask260CustomItemResourceConflictUsesBoundedConflictOutcome(t *testing.T) {
	sink := &observability.MemorySink{}
	conflict := repository.NewError(repository.ErrorKindConflict, "duplicate private item name", nil)
	service := NewService(&memoryItems{items: map[uuid.UUID]repository.CustomFoodItemEntity{}, claimErr: conflict}).WithTelemetry(observability.NewAdminExternalTelemetry(sink, sink))

	_, err := service.Create(context.Background(), uuid.New(), CreateRequest{Request: solidRequest("private duplicate name"), IdempotencyKey: "private-conflict-key"})
	if err != conflict {
		t.Fatalf("create error = %v, want unchanged resource conflict", err)
	}
	metrics, logs := sink.Snapshot()
	if len(metrics) != 1 || metrics[0].Name != observability.MetricCustomItemLifecycleOutcomes || len(metrics[0].Labels) != 2 || metrics[0].Labels["operation"] != "create" || metrics[0].Labels["outcome"] != "conflict" {
		t.Fatalf("conflict metric = %+v", metrics)
	}
	if len(logs) != 1 || logs[0].Message != "custom_item_lifecycle" || len(logs[0].Fields) != 2 || logs[0].Fields["operation"] != "create" || logs[0].Fields["outcome"] != "conflict" {
		t.Fatalf("conflict log = %+v", logs)
	}
	encoded, marshalErr := json.Marshal(struct {
		Metrics []observability.MetricPoint
		Logs    []observability.LogEvent
	}{metrics, logs})
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	for _, forbidden := range []string{"private duplicate name", "private-conflict-key", conflict.Error()} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("conflict telemetry leaked %q: %s", forbidden, encoded)
		}
	}
}
