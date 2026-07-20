// Implements DESIGN-004 LPSolverWrapper verification.
package optimization

import (
	"context"
	"errors"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
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
		runner: func(_ context.Context, executable string, args []string, stdout, stderr io.Writer) error {
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

func TestLPSolverWrapperUsesSolutionFileAsAuthoritativeResult(t *testing.T) {
	model, objective := wrapperFixture()
	tests := []struct {
		name          string
		writeSolution bool
		solution      string
		stdout        string
		want          LPSolution
		wantErr       error
	}{
		{
			name:          "solution file ignores misleading stdout",
			writeSolution: true,
			solution:      "Optimal - objective value 1\n0 x000001 1 0\n",
			stdout:        "Infeasible - objective value 0\n0 x000001 2 0\n",
			want:          LPSolution{"meal;caller-id": 1},
		},
		{
			name:   "absent solution file deliberately falls back to stdout",
			stdout: "Optimal - objective value 2\n0 x000001 2 0\n",
			want:   LPSolution{"meal;caller-id": 2},
		},
		{
			name:          "present empty solution file does not fall back",
			writeSolution: true,
			stdout:        "Optimal - objective value 2\n0 x000001 2 0\n",
			wantErr:       ErrSolverMalformed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			solver := NewLPSolverWrapper(CLPConfig{runner: func(_ context.Context, _ string, args []string, stdout, _ io.Writer) error {
				if tt.writeSolution {
					if err := os.WriteFile(args[3], []byte(tt.solution), 0o600); err != nil {
						return err
					}
				}
				_, _ = io.WriteString(stdout, tt.stdout)
				return nil
			}})
			got, err := solver.Solve(context.Background(), model, objective)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Solve() error = %v, want errors.Is(..., %v)", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Solve() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestLPSolverWrapperCleanupFailureIsBoundedAndPreservesPrimaryResult(t *testing.T) {
	model, objective := wrapperFixture()
	for _, primaryError := range []bool{false, true} {
		t.Run("primary_error_"+strconv.FormatBool(primaryError), func(t *testing.T) {
			var observed string
			solver := NewLPSolverWrapper(CLPConfig{
				runner: func(_ context.Context, _ string, args []string, _, _ io.Writer) error {
					if primaryError {
						return errors.New("exit status 9")
					}
					return os.WriteFile(args[3], []byte("Optimal - objective value 1\n0 x000001 1 0\n"), 0o600)
				},
				removeAll: func(path string) error {
					return errors.New(path + "\x1b" + strings.Repeat("x", maxSolverDiagnostic+100))
				},
				observeCleanup: func(diagnostic string) { observed = diagnostic },
			})
			got, err := solver.Solve(context.Background(), model, objective)
			if primaryError {
				if !errors.Is(err, ErrSolverNonZero) {
					t.Fatalf("Solve() error = %v, want primary nonzero error", err)
				}
			} else if err != nil || got["meal;caller-id"] != 1 {
				t.Fatalf("Solve() = %#v, %v, want successful primary result", got, err)
			}
			if observed == "" || strings.Contains(observed, "mealswapp-clp-") || strings.Contains(observed, "\x1b") || len(observed) > maxSolverDiagnostic+3 {
				t.Fatalf("cleanup observation is not redacted, sanitized, and bounded: %q", observed)
			}
		})
	}
}

func TestLPSolverWrapperTrustedRunnerContractDoesNotLeakDeadlineGoroutine(t *testing.T) {
	model, objective := wrapperFixture()
	solver := NewLPSolverWrapper(CLPConfig{
		Timeout: 5 * time.Millisecond,
		runner: func(_ context.Context, _ string, _ []string, _, _ io.Writer) error {
			time.Sleep(25 * time.Millisecond)
			return nil
		},
	})
	started := time.Now()
	_, err := solver.Solve(context.Background(), model, objective)
	if !errors.Is(err, ErrSolverTimeout) {
		t.Fatalf("Solve() error = %v, want timeout after trusted runner returns", err)
	}
	if elapsed := time.Since(started); elapsed < 20*time.Millisecond {
		t.Fatalf("Solve() returned in %v; package-only runner contract was not exercised", elapsed)
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
			solver := NewLPSolverWrapper(CLPConfig{runner: func(_ context.Context, _ string, args []string, _, _ io.Writer) error {
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
	canceledSolver := NewLPSolverWrapper(CLPConfig{runner: func(_ context.Context, _ string, _ []string, _, _ io.Writer) error {
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
			config: CLPConfig{Timeout: 10 * time.Millisecond, runner: func(ctx context.Context, _ string, _ []string, _, _ io.Writer) error {
				<-ctx.Done()
				return ctx.Err()
			}},
			want: ErrSolverTimeout,
		},
		{
			name: "malformed",
			config: CLPConfig{runner: func(_ context.Context, _ string, args []string, _, _ io.Writer) error {
				return os.WriteFile(args[3], []byte("Optimal\n"), 0o600)
			}},
			want: ErrSolverMalformed,
		},
		{
			name: "missing executable",
			config: CLPConfig{runner: func(_ context.Context, _ string, _ []string, _, _ io.Writer) error {
				return exec.ErrNotFound
			}},
			want: ErrSolverUnavailable,
		},
		{
			name: "nonzero exit",
			config: CLPConfig{runner: func(_ context.Context, _ string, _ []string, _, _ io.Writer) error {
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
	solver := NewLPSolverWrapper(CLPConfig{runner: func(_ context.Context, _ string, _ []string, _, stderr io.Writer) error {
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
			solver := NewLPSolverWrapper(CLPConfig{runner: func(_ context.Context, _ string, args []string, stdout, _ io.Writer) error {
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

func TestCLPVersionUsesFirstExactPunctuatedToken(t *testing.T) {
	tests := []struct {
		output string
		want   string
	}{
		{output: "Clp version (1.17.11); build 9.9.9", want: "1.17.11"},
		{output: "noise 1.17 then 1.17.12, 2.0.0", want: "1.17.12"},
		{output: "Clp version 1.17.11.1", want: "unknown"},
		{output: "Clp build unknown", want: "unknown"},
	}
	for _, tt := range tests {
		if got := clpVersion([]byte(tt.output)); got != tt.want {
			t.Errorf("clpVersion(%q) = %q, want %q", tt.output, got, tt.want)
		}
	}
}

func TestSerializeLPIsDeterministicAndUsesCanonicalGeneratedNames(t *testing.T) {
	model := LPModel{
		Variables: []LPVariable{
			{ItemID: "caller-z", LowerBound: 0, UpperBound: 5},
			{ItemID: "caller-a", LowerBound: 1, UpperBound: 9},
			{ItemID: "caller-zero", LowerBound: 0, UpperBound: 3},
		},
		Constraints: []LPConstraint{
			{LowerBound: 2, UpperBound: 2, Coefficients: map[string]float64{"caller-a": -2, "caller-z": 1, "caller-zero": 0}},
			{LowerBound: 1, UpperBound: 4, Coefficients: map[string]float64{"caller-a": 3, "caller-z": -1}},
		},
	}
	objective := ObjectiveFunction{Coefficients: map[string]float64{"caller-z": 1, "caller-a": -2, "caller-zero": 0}}
	want := "Minimize\n obj: 1 x000001 - 2 x000002\nSubject To\n" +
		" c000001: 1 x000001 - 2 x000002 = 2\n" +
		" c000002: -1 x000001 + 3 x000002 >= 1\n" +
		" c000002_upper: -1 x000001 + 3 x000002 <= 4\n" +
		"Bounds\n 0 <= x000001 <= 5\n 1 <= x000002 <= 9\n 0 <= x000003 <= 3\nEnd\n"
	for range 3 {
		got, names, err := serializeLP(model, objective)
		if err != nil {
			t.Fatalf("serializeLP() error = %v", err)
		}
		if string(got) != want {
			t.Fatalf("serializeLP() =\n%s\nwant:\n%s", got, want)
		}
		if !reflect.DeepEqual(names, map[string]string{"caller-z": "x000001", "caller-a": "x000002", "caller-zero": "x000003"}) {
			t.Fatalf("generated names = %#v", names)
		}
	}
}

func TestSerializeLPEnforcesLimitWhileWriting(t *testing.T) {
	model, objective := wrapperFixture()
	serialized, _, err := serializeLP(model, objective)
	if err != nil {
		t.Fatalf("serializeLP() error = %v", err)
	}
	if got, _, err := serializeLPWithLimit(model, objective, len(serialized)); err != nil || !reflect.DeepEqual(got, serialized) {
		t.Fatalf("exact-limit serialization = %q, %v", got, err)
	}
	if _, _, err := serializeLPWithLimit(model, objective, len(serialized)-1); !errors.Is(err, ErrSolverOutputLimit) {
		t.Fatalf("over-limit serialization error = %v, want output limit", err)
	}

	model.Constraints = make([]LPConstraint, 100)
	for index := range model.Constraints {
		model.Constraints[index] = LPConstraint{LowerBound: 0, UpperBound: 1, Coefficients: map[string]float64{"meal;caller-id": 1}}
	}
	model.Constraints[len(model.Constraints)-1].Coefficients = map[string]float64{"unknown": 1}
	if _, _, err := serializeLPWithLimit(model, objective, 200); !errors.Is(err, ErrSolverOutputLimit) {
		t.Fatalf("early over-limit serialization error = %v, want output limit before later invalid constraint", err)
	}
}

func TestSerializeLPRejectsInvalidReferencesBoundsAndCoefficients(t *testing.T) {
	model, _ := wrapperFixture()
	tests := []struct {
		name      string
		mutate    func(*LPModel, *ObjectiveFunction)
		wantError string
	}{
		{name: "invalid bounds", mutate: func(model *LPModel, _ *ObjectiveFunction) { model.Variables[0].LowerBound = -1 }, wantError: "bounds"},
		{name: "nonfinite coefficient", mutate: func(model *LPModel, _ *ObjectiveFunction) {
			model.Constraints[0].Coefficients[model.Variables[0].ItemID] = math.NaN()
		}, wantError: "coefficient"},
		{name: "unknown objective reference", mutate: func(_ *LPModel, objective *ObjectiveFunction) {
			delete(objective.Coefficients, "meal;caller-id")
			objective.Coefficients["unknown"] = 1
		}, wantError: "objective"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			candidateModel := model
			candidateModel.Variables = append([]LPVariable(nil), model.Variables...)
			candidateModel.Constraints = []LPConstraint{{LowerBound: model.Constraints[0].LowerBound, UpperBound: model.Constraints[0].UpperBound, Coefficients: map[string]float64{"meal;caller-id": 1}}}
			candidateObjective := ObjectiveFunction{Coefficients: map[string]float64{"meal;caller-id": 1}}
			tt.mutate(&candidateModel, &candidateObjective)
			_, _, err := serializeLP(candidateModel, candidateObjective)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("serializeLP() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}

func TestParseCLPSolutionUsesExactHeadersAndRows(t *testing.T) {
	variables := map[string]string{"x000001": "meal-a", "x000002": "meal-b"}
	tests := []struct {
		name       string
		output     string
		wantStatus SolverStatus
		want       LPSolution
		wantErr    string
	}{
		{name: "optimal indexed rows", output: "Optimal - objective value 3\n0 x000001 1.5 0\n1 x000002 0 -2\n", wantStatus: SolverStatusOptimal, want: LPSolution{"meal-a": 1.5}},
		{name: "marked row and tiny negative clamp", output: "Optimal - objective value 0\n** 0 x000001 -0.0000000005 2\n", wantStatus: SolverStatusOptimal, want: LPSolution{}},
		{name: "infeasible header", output: "Infeasible - objective value 0\n", wantStatus: SolverStatusInfeasible, want: LPSolution{}},
		{name: "unbounded header", output: "Unbounded - objective value -1\n", wantStatus: SolverStatusUnbounded, want: LPSolution{}},
		{name: "diagnostic generated token ignored", output: "diagnostic x999999 4\nOptimal - objective value 1\n0 x000001 1 0\n", wantStatus: SolverStatusOptimal, want: LPSolution{"meal-a": 1}},
		{name: "prefix lookalike", output: "optimality - objective value 1\n0 x000001 1 0\n", wantErr: "status is missing"},
		{name: "hyphenated lookalike", output: "Infeasible-data - objective value 1\n", wantErr: "status is missing"},
		{name: "malformed exact header", output: "Optimal objective value 1\n0 x000001 1 0\n", wantErr: "status header"},
		{name: "unknown indexed variable", output: "Optimal - objective value 1\n0 x999999 1 0\n", wantErr: "unknown solver variable"},
		{name: "duplicate variable", output: "Optimal - objective value 1\n0 x000001 1 0\n1 x000001 2 0\n", wantErr: "duplicate solver variable"},
		{name: "duplicate status", output: "Optimal - objective value 1\nOptimal - objective value 1\n0 x000001 1 0\n", wantErr: "duplicate solver status"},
		{name: "conflicting status", output: "Optimal - objective value 1\nInfeasible - objective value 1\n0 x000001 1 0\n", wantErr: "conflicting solver statuses"},
		{name: "missing quantity", output: "Optimal - objective value 1\n0 x000001\n", wantErr: "missing solver quantity"},
		{name: "malformed quantity", output: "Optimal - objective value 1\n0 x000001 nope 0\n", wantErr: "invalid solver quantity"},
		{name: "nonfinite quantity", output: "Optimal - objective value 1\n0 x000001 NaN 0\n", wantErr: "invalid solver quantity"},
		{name: "material negative", output: "Optimal - objective value 1\n0 x000001 -0.000001 0\n", wantErr: "invalid solver quantity"},
		{name: "missing variables", output: "Optimal - objective value 0\n", wantErr: "no variables"},
		{name: "missing status", output: "0 x000001 1 0\n", wantErr: "status is missing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, got, err := parseCLPSolution([]byte(tt.output), variables)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parseCLPSolution() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil || status != tt.wantStatus || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseCLPSolution() = %q, %#v, %v; want %q, %#v", status, got, err, tt.wantStatus, tt.want)
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
