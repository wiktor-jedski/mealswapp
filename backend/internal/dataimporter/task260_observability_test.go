package dataimporter

// Implements DESIGN-009 DataImporter and DESIGN-014 MetricsCollector Task 260 gate.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/externaldata"
	"github.com/wiktor-jedski/mealswapp/backend/internal/observability"
	"github.com/wiktor-jedski/mealswapp/backend/internal/providerregistry"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
)

func TestTask260ImportTelemetryDistinguishesCreatedAndConflict(t *testing.T) {
	sink := &observability.MemorySink{}
	store := &importStoreStub{result: repository.CuratedImportConfirmationResult{ImportID: uuid.New(), Item: repository.FoodItemEntity{ID: uuid.New(), Name: "private name", PhysicalState: repository.PhysicalStateSolid}}}
	service := NewService(store).WithTelemetry(observability.NewAdminExternalTelemetry(sink, sink))
	req := Request{SelectedRecord: providerregistry.Identity{Provider: "usda", ExternalID: "private-id"}, Request: validRequest("private name")}
	result, err := service.Confirm(context.Background(), importExecutorStub{}, uuid.New(), "private-idempotency-key", req)
	if err != nil {
		t.Fatal(err)
	}
	if metrics, _ := sink.Snapshot(); len(metrics) != 0 {
		t.Fatalf("success emitted before audit commit: %+v", metrics)
	}
	service.RecordCommittedOutcome(context.Background(), req.SelectedRecord.Provider, result)
	store.err = repository.ErrCuratedImportIdentityConflict
	if _, err := service.Confirm(context.Background(), importExecutorStub{}, uuid.New(), "private-idempotency-key", req); err != ErrProviderConflict {
		t.Fatalf("conflict err=%v", err)
	}
	metrics, _ := sink.Snapshot()
	if len(metrics) != 2 || metrics[0].Labels["outcome"] != "created" || metrics[1].Labels["outcome"] != "provider_conflict" {
		t.Fatalf("import metrics=%+v", metrics)
	}
}

// Implements DESIGN-012 DataNormalizer and DESIGN-014 MetricsCollector retryable evidence telemetry.
func TestTask299EvidenceOutageEmitsDependencyFailedTelemetry(t *testing.T) {
	sink := &observability.MemorySink{}
	service := NewService(&importStoreStub{}, evidenceResolverStub{err: externaldata.ErrRecordEvidenceUnavailable}).WithTelemetry(observability.NewAdminExternalTelemetry(sink, sink))
	request := Request{ExternalRecordToken: "opaque", Request: validRequest("Food")}
	if _, err := service.Confirm(context.Background(), importExecutorStub{}, uuid.New(), "", request); err != ErrExternalRecordEvidenceUnavailable {
		t.Fatalf("error=%v, want evidence dependency failure", err)
	}
	metrics, _ := sink.Snapshot()
	if len(metrics) != 1 || metrics[0].Name != observability.MetricAdminImportOutcomes || metrics[0].Labels["outcome"] != "dependency_failed" {
		t.Fatalf("evidence outage telemetry=%+v", metrics)
	}
}
