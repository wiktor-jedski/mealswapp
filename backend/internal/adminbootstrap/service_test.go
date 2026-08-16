package adminbootstrap

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

type testKeys struct{}

func (testKeys) ActiveLookupKey(context.Context) (string, []byte, error) {
	return "lookup-v1", []byte("12345678901234567890123456789012"), nil
}
func (testKeys) LookupKey(context.Context, string) ([]byte, error) {
	return []byte("12345678901234567890123456789012"), nil
}

type recordingRepository struct {
	selector repository.AdministratorBootstrapSelector
	result   repository.AdministratorBootstrapResult
	err      error
}

func (r *recordingRepository) BootstrapAdministrator(_ context.Context, selector repository.AdministratorBootstrapSelector, _ string) (repository.AdministratorBootstrapResult, error) {
	r.selector = selector
	return r.result, r.err
}

// Implements DESIGN-009 AdminController environment and normalized lookup gates.
func TestBootstrapValidatesEnvironmentAndNormalizesEmail(t *testing.T) {
	repo := &recordingRepository{result: repository.AdministratorBootstrapResult{UserID: uuid.New()}}
	service := NewService(repo, security.NewLookupDigestService(testKeys{}))
	request := Request{ConfiguredEnvironment: "development", TargetEnvironment: "development", Email: "  ADMIN@Example.TEST ", RequestID: "request"}
	if _, err := service.Bootstrap(context.Background(), request); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	want, _ := security.NewLookupDigestService(testKeys{}).DigestForWrite(context.Background(), []byte("admin@example.test"))
	legacy, _ := security.NewLookupDigestService(testKeys{}).DigestForWrite(context.Background(), []byte("ADMIN@Example.TEST"))
	if repo.selector.EmailDigest == nil || repo.selector.EmailDigest.Value != want.Value || repo.selector.LegacyEmailDigest == nil || repo.selector.LegacyEmailDigest.Value != legacy.Value || repo.selector.UserID != nil {
		t.Fatalf("selector = %#v", repo.selector)
	}

	for _, invalid := range []Request{
		{ConfiguredEnvironment: "development", TargetEnvironment: "production", Email: "admin@example.test"},
		{ConfiguredEnvironment: "production", TargetEnvironment: "production", Email: "admin@example.test"},
		{ConfiguredEnvironment: "development", TargetEnvironment: "development"},
		{ConfiguredEnvironment: "development", TargetEnvironment: "development", Email: "bad"},
	} {
		if _, err := service.Bootstrap(context.Background(), invalid); err == nil {
			t.Fatalf("Bootstrap(%#v) error = nil", invalid)
		}
	}
}

// Implements DESIGN-009 AdminController optional advanced UUID selector.
func TestBootstrapSupportsOnlyOneUUIDSelector(t *testing.T) {
	id := uuid.New()
	repo := &recordingRepository{}
	service := NewService(repo, security.NewLookupDigestService(testKeys{}))
	if _, err := service.Bootstrap(context.Background(), Request{ConfiguredEnvironment: "production", TargetEnvironment: "production", ConfirmProduction: true, UserID: &id, RequestID: "request"}); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if repo.selector.UserID == nil || *repo.selector.UserID != id || repo.selector.EmailDigest != nil {
		t.Fatalf("selector = %#v", repo.selector)
	}
	if _, err := service.Bootstrap(context.Background(), Request{ConfiguredEnvironment: "development", TargetEnvironment: "development", Email: "admin@example.test", UserID: &id}); err == nil {
		t.Fatal("Bootstrap() accepted two selectors")
	}
}

type failingKeys struct{}

func (failingKeys) ActiveLookupKey(context.Context) (string, []byte, error) {
	return "", nil, errors.New("secret")
}
func (failingKeys) LookupKey(context.Context, string) ([]byte, error) {
	return nil, errors.New("secret")
}

func TestBootstrapHidesLookupFailureAndPropagatesRepositoryOutcome(t *testing.T) {
	service := NewService(&recordingRepository{}, security.NewLookupDigestService(failingKeys{}))
	if _, err := service.Bootstrap(context.Background(), Request{ConfiguredEnvironment: "development", TargetEnvironment: "development", Email: "admin@example.test"}); err == nil || err.Error() != "bootstrap email lookup is unavailable" {
		t.Fatalf("lookup error = %v", err)
	}
	want := repository.ErrAdministratorAlreadyExists
	service = NewService(&recordingRepository{err: want}, security.NewLookupDigestService(testKeys{}))
	if _, err := service.Bootstrap(context.Background(), Request{ConfiguredEnvironment: "development", TargetEnvironment: "development", Email: "admin@example.test"}); !errors.Is(err, want) {
		t.Fatalf("repository error = %v", err)
	}
}
