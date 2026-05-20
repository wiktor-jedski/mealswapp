package http

import (
	"context"
	"mealswapp/backend/internal/config"
	"mealswapp/backend/internal/http/apperrors"
	"mealswapp/backend/internal/http/handlers"
	"net/http"
	"testing"
)

func TestProfileControllerReadsAndUpdatesProfile(t *testing.T) {
	service := &fakeProfileService{
		profile: handlers.Profile{
			ID:            "user-1",
			Email:         "user@example.com",
			EmailVerified: true,
			DisplayName:   "User",
			DietarySettings: map[string]any{
				"diet": "vegetarian",
			},
			Metadata: map[string]any{"source": "test"},
		},
	}
	app := NewRouter(ServiceDependencies{Config: config.Config{Environment: "test"}, ProfileService: service})

	getRes := performJSONRequest(t, app, http.MethodGet, "/api/v1/profile", "", "access-token", false)
	defer getRes.Body.Close()
	if getRes.StatusCode != http.StatusOK {
		t.Fatalf("expected profile read 200, got %d", getRes.StatusCode)
	}
	getPayload := decodeEnvelope(t, getRes)
	if !getPayload.Success {
		t.Fatalf("expected profile success, got %#v", getPayload)
	}

	updateRes := performJSONRequest(t, app, http.MethodPatch, "/api/v1/profile", `{"displayName":" Updated User ","dietarySettings":{"diet":"vegan"},"metadata":{"timezone":"UTC"}}`, "access-token", true)
	defer updateRes.Body.Close()
	if updateRes.StatusCode != http.StatusOK {
		t.Fatalf("expected profile update 200, got %d", updateRes.StatusCode)
	}
	if service.updated.DisplayName == nil || *service.updated.DisplayName != "Updated User" {
		t.Fatalf("expected trimmed display name update, got %#v", service.updated)
	}
}

func TestProfileControllerRejectsUnauthorizedAccess(t *testing.T) {
	app := NewRouter(ServiceDependencies{Config: config.Config{Environment: "test"}, ProfileService: &fakeProfileService{}})

	res := performJSONRequest(t, app, http.MethodGet, "/api/v1/profile", "", "", false)
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized profile read, got %d", res.StatusCode)
	}
}

func TestProfileControllerPropagatesForbiddenServiceError(t *testing.T) {
	app := NewRouter(ServiceDependencies{
		Config: config.Config{Environment: "test"},
		ProfileService: &fakeProfileService{
			err: apperrors.Forbidden("Forbidden"),
		},
	})

	res := performJSONRequest(t, app, http.MethodGet, "/api/v1/profile", "", "access-token", false)
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected forbidden profile read, got %d", res.StatusCode)
	}
}

type fakeProfileService struct {
	profile handlers.Profile
	updated handlers.ProfileUpdate
	err     error
}

func (service *fakeProfileService) GetProfile(ctx context.Context, accessToken string) (handlers.Profile, error) {
	if service.err != nil {
		return handlers.Profile{}, service.err
	}
	return service.profile, nil
}

func (service *fakeProfileService) UpdateProfile(ctx context.Context, accessToken string, update handlers.ProfileUpdate) (handlers.Profile, error) {
	if service.err != nil {
		return handlers.Profile{}, service.err
	}
	service.updated = update
	if update.DisplayName != nil {
		service.profile.DisplayName = *update.DisplayName
	}
	service.profile.DietarySettings = update.DietarySettings
	service.profile.Metadata = update.Metadata
	return service.profile, nil
}
