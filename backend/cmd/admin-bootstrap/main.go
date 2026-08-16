package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/wiktor-jedski/mealswapp/backend/internal/adminbootstrap"
	"github.com/wiktor-jedski/mealswapp/backend/internal/auth"
	"github.com/wiktor-jedski/mealswapp/backend/internal/config"
	"github.com/wiktor-jedski/mealswapp/backend/internal/repository"
	"github.com/wiktor-jedski/mealswapp/backend/internal/security"
)

// main runs the operator-only administrator bootstrap CLI.
// Implements DESIGN-009 AdminController operator-only administrator bootstrap CLI.
func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, executeBootstrap))
}

// bootstrapExecutor is the injectable CLI application boundary.
// Implements DESIGN-009 AdminController operator-only administrator bootstrap CLI.
type bootstrapExecutor func(context.Context, adminbootstrap.Request) (repository.AdministratorBootstrapResult, error)

// run parses safe CLI input and emits a privacy-minimized result.
// Implements DESIGN-009 AdminController operator-only administrator bootstrap CLI.
func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, execute bootstrapExecutor) int {
	flags := flag.NewFlagSet("admin-bootstrap", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	environment := flags.String("environment", "", "")
	email := flags.String("email", "", "")
	userIDValue := flags.String("user-id", "", "")
	confirmProduction := flags.Bool("confirm-production", false, "")
	if err := flags.Parse(args); err != nil || flags.NArg() != 0 || strings.TrimSpace(*environment) == "" {
		fmt.Fprintln(stderr, "bootstrap denied: supply --environment and one supported selector")
		return 2
	}

	var userID *uuid.UUID
	if strings.TrimSpace(*userIDValue) != "" {
		parsed, err := uuid.Parse(*userIDValue)
		if err != nil {
			fmt.Fprintln(stderr, "bootstrap denied: user identifier is invalid")
			return 2
		}
		userID = &parsed
	}
	if strings.TrimSpace(*email) != "" && userID != nil {
		fmt.Fprintln(stderr, "bootstrap denied: choose email or user identifier")
		return 2
	}
	if strings.TrimSpace(*email) == "" && userID == nil {
		fmt.Fprint(stderr, "Administrator email: ")
		scanner := bufio.NewScanner(io.LimitReader(stdin, 1025))
		scanner.Buffer(make([]byte, 256), 1024)
		if !scanner.Scan() {
			fmt.Fprintln(stderr, "\nbootstrap denied: email input is required")
			return 2
		}
		*email = scanner.Text()
	}

	result, err := execute(context.Background(), adminbootstrap.Request{
		TargetEnvironment: *environment,
		ConfirmProduction: *confirmProduction,
		Email:             *email,
		UserID:            userID,
		RequestID:         uuid.NewString(),
	})
	if err != nil {
		fmt.Fprintf(stderr, "bootstrap denied: %s\n", safeBootstrapError(err))
		return 1
	}
	status := "created"
	if result.Replayed {
		status = "unchanged"
	}
	auditID := "none"
	if result.AuditID != nil {
		auditID = result.AuditID.String()
	}
	fmt.Fprintf(stdout, "result=%s user_id=%s audit_id=%s actor=operator reauthentication_required=true\n", status, result.UserID, auditID)
	return 0
}

// executeBootstrap composes configured lookup and PostgreSQL persistence.
// Implements DESIGN-009 AdminController operator-only administrator bootstrap CLI.
func executeBootstrap(ctx context.Context, request adminbootstrap.Request) (repository.AdministratorBootstrapResult, error) {
	cfg, err := config.Load()
	if err != nil {
		return repository.AdministratorBootstrapResult{}, errors.New("configuration unavailable")
	}
	request.ConfiguredEnvironment = cfg.Environment
	keys, err := newLookupKeyLoader(cfg.Environment)
	if err != nil {
		return repository.AdministratorBootstrapResult{}, errors.New("lookup configuration unavailable")
	}
	conn, err := pgx.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return repository.AdministratorBootstrapResult{}, errors.New("database unavailable")
	}
	defer conn.Close(ctx)
	service := adminbootstrap.NewService(
		repository.NewPostgresAdministratorBootstrapRepository(conn, auth.IsUsablePasswordCredential),
		security.NewLookupDigestService(keys),
	)
	return service.Bootstrap(ctx, request)
}

// safeBootstrapError maps internal failures to a closed non-PII vocabulary.
// Implements DESIGN-009 AdminController operator-only administrator bootstrap CLI.
func safeBootstrapError(err error) string {
	switch {
	case errors.Is(err, repository.ErrAdministratorBootstrapTargetNotFound):
		return "target not found"
	case errors.Is(err, repository.ErrAdministratorBootstrapUnverified):
		return "target is not verified"
	case errors.Is(err, repository.ErrAdministratorBootstrapNoCredential):
		return "target has no usable credential"
	case errors.Is(err, repository.ErrAdministratorAlreadyExists):
		return "another administrator exists"
	case errors.Is(err, repository.ErrCanonicalEmailCollision):
		return "email identity conflict"
	default:
		return "request rejected"
	}
}

// lookupKeyLoader supplies the configured deterministic email lookup key.
// Implements DESIGN-009 AdminController encrypted-email bootstrap lookup.
type lookupKeyLoader struct {
	key []byte
}

// newLookupKeyLoader loads local/deployed lookup material without exposing it.
// Implements DESIGN-009 AdminController encrypted-email bootstrap lookup.
func newLookupKeyLoader(environment string) (lookupKeyLoader, error) {
	key := os.Getenv("MEALSWAPP_LOCAL_SECRET_KEY")
	if key == "" && environment != "production" {
		key = "dev-local-secret-key-32-bytes-ok!"
	}
	if len([]byte(key)) < 32 {
		return lookupKeyLoader{}, errors.New("lookup key is unavailable")
	}
	return lookupKeyLoader{key: []byte(key)[:32]}, nil
}

// ActiveLookupKey returns the active local lookup-key version and material.
// Implements DESIGN-009 AdminController encrypted-email bootstrap lookup.
func (l lookupKeyLoader) ActiveLookupKey(context.Context) (string, []byte, error) {
	return "local-v1", l.key, nil
}

// LookupKey returns configured local lookup material.
// Implements DESIGN-009 AdminController encrypted-email bootstrap lookup.
func (l lookupKeyLoader) LookupKey(context.Context, string) ([]byte, error) {
	return l.key, nil
}
