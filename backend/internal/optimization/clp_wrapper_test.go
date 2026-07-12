// Implements DESIGN-004 LPSolverWrapper verification.
package optimization

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLPSolverWrapperMapsOptimalOutputAndUsesGeneratedArguments(t *testing.T) {
	model, objective := wrapperFixture()
	var capturedArgs []string
	var capturedModel []byte
	var jobDir string
	solver := NewLPSolverWrapper(CLPConfig{
		Executable:      "configured-clp",
		ExpectedVersion: SupportedCLPVersion,
		TempRoot:        t.TempDir(),
		Runner: func(_ context.Context, executable string, args []string, stdout, stderr io.Writer) error {
			if executable != "configured-clp" {
				t.Fatalf("executable = %q, want configured-clp", executable)
			}
			capturedArgs = append([]string(nil), args...)
			jobDir = filepath.Dir(args[0])
			var err error
			capturedModel, err = os.ReadFile(args[0])
			if err != nil {
				t.Fatalf("read generated model: %v", err)
			}
			if err := os.WriteFile(args[3], []byte("Optimal - objective value 1\n      0 x000001 1 0\n"), 0o600); err != nil {
				t.Fatalf("write generated solution: %v", err)
			}
			_, _ = io.WriteString(stdout, "CLP completed\n")
			_, _ = io.WriteString(stderr, "diagnostic\n")
			return nil
		},
	})

	got, err := solver.Solve(context.Background(), model, objective)
	if err != nil {
		t.Fatalf("Solve() error = %v", err)
	}
	if got["meal;caller-id"] != 1 {
		t.Fatalf("solution = %#v, want generated value mapped to original ID", got)
	}
	if len(capturedArgs) != 5 || capturedArgs[1] != "-solve" || capturedArgs[2] != "-solution" || capturedArgs[4] != "-quit" {
		t.Fatalf("CLP args = %#v, want fixed solve arguments", capturedArgs)
	}
	if strings.Contains(string(capturedModel), "meal;caller-id") || strings.Contains(strings.Join(capturedArgs, " "), "meal;caller-id") {
		t.Fatal("caller-controlled item ID crossed the subprocess argument/model-name boundary")
	}
	if _, err := os.Stat(jobDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporary solver directory still exists: %v", err)
	}
}

func TestLPSolverWrapperMapsTerminalStatuses(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   error
	}{
		{name: "infeasible", output: "Infeasible - objective value 0\n", want: ErrSolverInfeasible},
		{name: "unbounded", output: "Unbounded - objective value 0\n", want: ErrSolverUnbounded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, objective := wrapperFixture()
			solver := NewLPSolverWrapper(CLPConfig{Runner: func(_ context.Context, _ string, args []string, _, _ io.Writer) error {
				return os.WriteFile(args[3], []byte(tt.output), 0o600)
			}})
			_, err := solver.Solve(context.Background(), model, objective)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Solve() error = %v, want errors.Is(..., %v)", err, tt.want)
			}
		})
	}
}

func TestLPSolverWrapperMapsCanceledTimeoutMalformedMissingAndNonZero(t *testing.T) {
	model, objective := wrapperFixture()
	canceledContext, cancel := context.WithCancel(context.Background())
	canceledSolver := NewLPSolverWrapper(CLPConfig{Runner: func(_ context.Context, _ string, _ []string, _, _ io.Writer) error {
		cancel()
		return context.Canceled
	}})
	if _, err := canceledSolver.Solve(canceledContext, model, objective); !errors.Is(err, ErrSolverCanceled) {
		t.Fatalf("canceled Solve() error = %v, want cancellation error", err)
	}

	tests := []struct {
		name   string
		config CLPConfig
		want   error
	}{
		{
			name: "timeout",
			config: CLPConfig{Timeout: 10 * time.Millisecond, Runner: func(ctx context.Context, _ string, _ []string, _, _ io.Writer) error {
				<-ctx.Done()
				return ctx.Err()
			}},
			want: ErrSolverTimeout,
		},
		{
			name: "malformed",
			config: CLPConfig{Runner: func(_ context.Context, _ string, args []string, _, _ io.Writer) error {
				return os.WriteFile(args[3], []byte("Optimal\n"), 0o600)
			}},
			want: ErrSolverMalformed,
		},
		{
			name: "missing executable",
			config: CLPConfig{Runner: func(_ context.Context, _ string, _ []string, _, _ io.Writer) error {
				return exec.ErrNotFound
			}},
			want: ErrSolverUnavailable,
		},
		{
			name: "nonzero exit",
			config: CLPConfig{Runner: func(_ context.Context, _ string, _ []string, _, _ io.Writer) error {
				return errors.New("exit status 7")
			}},
			want: ErrSolverNonZero,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewLPSolverWrapper(tt.config).Solve(context.Background(), model, objective)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Solve() error = %v, want errors.Is(..., %v)", err, tt.want)
			}
		})
	}
}

func TestLPSolverWrapperTerminatesRealChildAndCleansDeadlineDirectory(t *testing.T) {
	script := filepath.Join(t.TempDir(), "clp-fixture")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexec sleep 30\n"), 0o700); err != nil {
		t.Fatalf("write timeout fixture: %v", err)
	}
	tempRoot := t.TempDir()
	model, objective := wrapperFixture()
	started := time.Now()
	_, err := NewLPSolverWrapper(CLPConfig{
		Executable: script,
		TempRoot:   tempRoot,
		Timeout:    20 * time.Millisecond,
	}).Solve(context.Background(), model, objective)
	if !errors.Is(err, ErrSolverTimeout) {
		t.Fatalf("Solve() error = %v, want timeout error", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("deadline cleanup took %v, child was not terminated promptly", elapsed)
	}
	entries, err := os.ReadDir(tempRoot)
	if err != nil {
		t.Fatalf("read solver temp root: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("solver temp root = %v, want unconditional cleanup", entries)
	}
}

func TestLPSolverWrapperBoundsAndSanitizesOutput(t *testing.T) {
	model, objective := wrapperFixture()
	solver := NewLPSolverWrapper(CLPConfig{Runner: func(_ context.Context, _ string, _ []string, _, stderr io.Writer) error {
		_, _ = io.WriteString(stderr, "bad\x1b[31m"+strings.Repeat("x", MaxSolverOutputBytes+1))
		return errors.New("exit status 1")
	}})
	_, err := solver.Solve(context.Background(), model, objective)
	if !errors.Is(err, ErrSolverOutputLimit) {
		t.Fatalf("Solve() error = %v, want output-limit error", err)
	}
	if strings.Contains(err.Error(), "\x1b") || len(err.Error()) > maxSolverDiagnostic+100 {
		t.Fatalf("diagnostic is not bounded and sanitized: %q", err)
	}
}

func TestLPSolverWrapperChecksPinnedVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr error
	}{
		{name: "accepted", version: "Clp version 1.17.11\n"},
		{name: "mismatch", version: "Clp version 1.17.5\n", wantErr: ErrSolverVersion},
		{name: "malformed", version: "Clp build unknown\n", wantErr: ErrSolverVersion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			solver := NewLPSolverWrapper(CLPConfig{Runner: func(_ context.Context, _ string, args []string, stdout, _ io.Writer) error {
				if len(args) != 1 || args[0] != "-version" {
					t.Fatalf("version args = %#v", args)
				}
				_, _ = io.WriteString(stdout, tt.version)
				return nil
			}})
			err := solver.CheckVersion(context.Background())
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("CheckVersion() error = %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CheckVersion() error = %v, want errors.Is(..., %v)", err, tt.wantErr)
			}
		})
	}
}

func TestLPSolverWrapperRunsPackagedExecutableWhenAvailable(t *testing.T) {
	executable := os.Getenv("MEALSWAPP_CLP_EXECUTABLE")
	if executable == "" {
		executable, _ = exec.LookPath(DefaultCLPExecutable)
	}
	if executable == "" {
		t.Skip("native CLP executable is not installed; packaged worker CI supplies it")
	}

	expectedVersion := os.Getenv("MEALSWAPP_CLP_VERSION")
	if expectedVersion == "" {
		expectedVersion = SupportedCLPVersion
	}
	solver := NewLPSolverWrapper(CLPConfig{Executable: executable, ExpectedVersion: expectedVersion})
	if err := solver.CheckVersion(context.Background()); err != nil {
		t.Fatalf("packaged CLP version check: %v", err)
	}
	model, objective := wrapperFixture()
	if _, err := solver.Solve(context.Background(), model, objective); err != nil {
		t.Fatalf("packaged CLP solve: %v", err)
	}
}

func wrapperFixture() (LPModel, ObjectiveFunction) {
	return LPModel{
			Variables: []LPVariable{{
				ItemID:          "meal;caller-id",
				LowerBound:      0,
				UpperBound:      10,
				CaloriesPerUnit: 1,
				ProteinPerUnit:  1,
			}},
			Constraints: []LPConstraint{{
				Name:         "caller;constraint",
				LowerBound:   1,
				UpperBound:   1,
				Coefficients: map[string]float64{"meal;caller-id": 1},
			}},
		}, ObjectiveFunction{
			Coefficients: map[string]float64{"meal;caller-id": 1},
		}
}
