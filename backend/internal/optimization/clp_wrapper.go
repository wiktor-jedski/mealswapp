package optimization

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Implements DESIGN-004 LPSolverWrapper as a pure-Go child-process boundary.
const (
	// DefaultCLPExecutable is the default native solver command resolved by the worker.
	DefaultCLPExecutable = "clp"
	// SupportedCLPVersion is the exact native solver version accepted at readiness.
	SupportedCLPVersion = "1.17.11"
	// SolverDeadline is the maximum duration allowed for one CLP invocation.
	SolverDeadline = 30 * time.Second
	// MaxSolverOutputBytes bounds solver-controlled stdout and stderr capture.
	MaxSolverOutputBytes = 64 * 1024
	maxSolverModelBytes  = 1 << 20
	maxSolverDiagnostic  = 4 * 1024
)

// SolverStatus is the machine-readable terminal status emitted by CLP.
// Implements DESIGN-004 LPSolverWrapper.
type SolverStatus string

// Implements DESIGN-004 LPSolverWrapper solver statuses.
const (
	// SolverStatusOptimal identifies a solved model with an optimal solution.
	SolverStatusOptimal SolverStatus = "optimal"
	// SolverStatusInfeasible identifies a model with no feasible solution.
	SolverStatusInfeasible SolverStatus = "infeasible"
	// SolverStatusUnbounded identifies a model with an unbounded objective.
	SolverStatusUnbounded SolverStatus = "unbounded"
)

// SolverErrorKind classifies failures at the solver process boundary.
// Implements DESIGN-004 LPSolverWrapper error handling.
type SolverErrorKind string

// Implements DESIGN-004 LPSolverWrapper error kinds.
const (
	// SolverErrorUnavailable identifies a missing or inaccessible CLP executable.
	SolverErrorUnavailable SolverErrorKind = "unavailable"
	// SolverErrorVersion identifies an unsupported CLP executable version.
	SolverErrorVersion SolverErrorKind = "version"
	// SolverErrorInfeasible identifies an infeasible CLP result.
	SolverErrorInfeasible SolverErrorKind = "infeasible"
	// SolverErrorUnbounded identifies an unbounded CLP result.
	SolverErrorUnbounded SolverErrorKind = "unbounded"
	// SolverErrorCanceled identifies a solver invocation cancelled by its context.
	SolverErrorCanceled SolverErrorKind = "canceled"
	// SolverErrorTimeout identifies a solver invocation exceeding its deadline.
	SolverErrorTimeout SolverErrorKind = "timeout"
	// SolverErrorMalformed identifies solver output that cannot be safely decoded.
	SolverErrorMalformed SolverErrorKind = "malformed_output"
	// SolverErrorNonZero identifies an unsuccessful CLP process exit.
	SolverErrorNonZero SolverErrorKind = "nonzero_exit"
	// SolverErrorOutputLimit identifies solver output exceeding its safety bound.
	SolverErrorOutputLimit SolverErrorKind = "output_limit"
	// SolverErrorValidation identifies an invalid model or wrapper configuration.
	SolverErrorValidation SolverErrorKind = "validation"
)

// Implements DESIGN-004 LPSolverWrapper sentinel errors.
var (
	// ErrSolverUnavailable is the sentinel for an unavailable CLP executable.
	ErrSolverUnavailable = errors.New("solver executable unavailable")
	// ErrSolverVersion is the sentinel for an unsupported CLP version.
	ErrSolverVersion = errors.New("solver executable version unsupported")
	// ErrSolverInfeasible is the sentinel for an infeasible solver result.
	ErrSolverInfeasible = errors.New("solver reported infeasible")
	// ErrSolverUnbounded is the sentinel for an unbounded solver result.
	ErrSolverUnbounded = errors.New("solver reported unbounded")
	// ErrSolverCanceled is the sentinel for a cancelled solver invocation.
	ErrSolverCanceled = errors.New("solver canceled")
	// ErrSolverTimeout is the sentinel for an expired solver deadline.
	ErrSolverTimeout = errors.New("solver deadline exceeded")
	// ErrSolverMalformed is the sentinel for malformed solver output.
	ErrSolverMalformed = errors.New("solver output malformed")
	// ErrSolverNonZero is the sentinel for an unsuccessful solver exit.
	ErrSolverNonZero = errors.New("solver exited unsuccessfully")
	// ErrSolverOutputLimit is the sentinel for solver output exceeding its bound.
	ErrSolverOutputLimit = errors.New("solver output exceeded limit")
)

// SolverError is a safe, classified error from the CLP boundary. Diagnostic
// contains only bounded, sanitized process output and never the raw exec error.
// Implements DESIGN-004 LPSolverWrapper.
type SolverError struct {
	Kind       SolverErrorKind
	Status     SolverStatus
	Diagnostic string
}

// Error returns a user-safe solver error string.
// Implements DESIGN-004 LPSolverWrapper error handling.
func (e *SolverError) Error() string {
	if e == nil {
		return ""
	}
	message := "CLP solver failed"
	switch e.Kind {
	case SolverErrorUnavailable:
		message = "CLP executable is unavailable"
	case SolverErrorVersion:
		message = "CLP executable version is unsupported"
	case SolverErrorInfeasible:
		message = "CLP reported an infeasible model"
	case SolverErrorUnbounded:
		message = "CLP reported an unbounded model"
	case SolverErrorCanceled:
		message = "CLP solve was canceled"
	case SolverErrorTimeout:
		message = "CLP solve exceeded its deadline"
	case SolverErrorMalformed:
		message = "CLP returned malformed output"
	case SolverErrorNonZero:
		message = "CLP exited unsuccessfully"
	case SolverErrorOutputLimit:
		message = "CLP output exceeded the safety limit"
	case SolverErrorValidation:
		message = "CLP model validation failed"
	}
	if e.Diagnostic != "" {
		return message + ": " + e.Diagnostic
	}
	return message
}

// Unwrap lets callers use errors.Is without exposing process details.
// Implements DESIGN-004 LPSolverWrapper error handling.
func (e *SolverError) Unwrap() error {
	if e == nil {
		return nil
	}
	switch e.Kind {
	case SolverErrorUnavailable:
		return ErrSolverUnavailable
	case SolverErrorVersion:
		return ErrSolverVersion
	case SolverErrorInfeasible:
		return ErrSolverInfeasible
	case SolverErrorUnbounded:
		return ErrSolverUnbounded
	case SolverErrorCanceled:
		return ErrSolverCanceled
	case SolverErrorTimeout:
		return ErrSolverTimeout
	case SolverErrorMalformed:
		return ErrSolverMalformed
	case SolverErrorNonZero:
		return ErrSolverNonZero
	case SolverErrorOutputLimit:
		return ErrSolverOutputLimit
	default:
		return nil
	}
}

// CommandRunner is the injectable OS command boundary used by LPSolverWrapper.
// A production runner must use exec.CommandContext; tests can write fixture
// output to stdout or stderr and inspect the fixed argument list.
// Implements DESIGN-004 LPSolverWrapper.
type CommandRunner func(ctx context.Context, executable string, args []string, stdout, stderr io.Writer) error

// CLPConfig configures the native CLP executable. Executable and ExpectedVersion
// are deployment configuration, not request data. Timeout may be shortened for
// tests but cannot exceed SolverDeadline.
// Implements DESIGN-004 LPSolverWrapper deployment configuration.
type CLPConfig struct {
	Executable      string
	ExpectedVersion string
	TempRoot        string
	Timeout         time.Duration
	Runner          CommandRunner
}

// LPSolverWrapper invokes a pinned native CLP executable without CGO.
// Implements DESIGN-004 LPSolverWrapper.
type LPSolverWrapper struct {
	config CLPConfig
}

// CLPSolver is an expressive alias for LPSolverWrapper.
// Implements DESIGN-004 LPSolverWrapper.
type CLPSolver = LPSolverWrapper

// NewLPSolverWrapper constructs a pure-Go CLP child-process wrapper.
// Implements DESIGN-004 LPSolverWrapper.
func NewLPSolverWrapper(config CLPConfig) *LPSolverWrapper {
	return &LPSolverWrapper{config: config}
}

// NewCLPSolver constructs a pure-Go CLP child-process wrapper.
// Implements DESIGN-004 LPSolverWrapper.
func NewCLPSolver(config CLPConfig) *LPSolverWrapper {
	return NewLPSolverWrapper(config)
}

// Solve serializes a validated model, runs CLP with a hard worker deadline, and
// converts the bounded solution file back to repository meal IDs.
// Implements DESIGN-004 LPSolverWrapper and SW-REQ-021/SW-REQ-022.
func (s *LPSolverWrapper) Solve(ctx context.Context, model LPModel, objective ObjectiveFunction) (LPSolution, error) {
	if ctx == nil {
		return nil, solverValidationError("solver context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, contextSolverError(err, "")
	}
	config, err := s.validatedConfig()
	if err != nil {
		return nil, err
	}
	modelData, variableNames, err := serializeLP(model, objective)
	if err != nil {
		return nil, err
	}

	jobDir, err := os.MkdirTemp(config.TempRoot, "mealswapp-clp-")
	if err != nil {
		return nil, solverProcessErrorWithDiagnostic(SolverErrorUnavailable, "cannot create private solver directory")
	}
	defer os.RemoveAll(jobDir)

	modelPath := filepath.Join(jobDir, "model.lp")
	solutionPath := filepath.Join(jobDir, "solution.txt")
	if err := os.WriteFile(modelPath, modelData, 0o600); err != nil {
		return nil, solverProcessErrorWithDiagnostic(SolverErrorUnavailable, "cannot write solver model")
	}

	runContext, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()
	var stdout, stderr limitedBuffer
	runError := config.Runner(runContext, config.Executable, []string{
		modelPath,
		"-solve",
		"-solution",
		solutionPath,
		"-quit",
	}, &stdout, &stderr)
	diagnostic := solverDiagnostic(jobDir, stdout.Bytes(), stderr.Bytes())
	if runContext.Err() != nil {
		return nil, contextSolverError(runContext.Err(), diagnostic)
	}
	if runError != nil {
		if errors.Is(runError, exec.ErrNotFound) {
			return nil, solverProcessErrorWithDiagnostic(SolverErrorUnavailable, diagnostic)
		}
		if stdout.Truncated() || stderr.Truncated() {
			return nil, solverProcessErrorWithDiagnostic(SolverErrorOutputLimit, diagnostic)
		}
		return nil, solverProcessErrorWithDiagnostic(SolverErrorNonZero, diagnostic)
	}
	if stdout.Truncated() || stderr.Truncated() {
		return nil, solverProcessErrorWithDiagnostic(SolverErrorOutputLimit, diagnostic)
	}

	solutionData, readError := readBoundedFile(solutionPath)
	if errors.Is(readError, ErrSolverOutputLimit) {
		return nil, solverProcessErrorWithDiagnostic(SolverErrorOutputLimit, diagnostic)
	}
	if readError != nil && !errors.Is(readError, os.ErrNotExist) {
		return nil, solverProcessErrorWithDiagnostic(SolverErrorMalformed, diagnostic)
	}
	if errors.Is(readError, os.ErrNotExist) {
		solutionData = nil
	}
	if len(solutionData) > 0 && len(stdout.Bytes()) > 0 {
		solutionData = append(solutionData, '\n')
		solutionData = append(solutionData, stdout.Bytes()...)
	} else if len(solutionData) == 0 {
		solutionData = append([]byte(nil), stdout.Bytes()...)
	}
	if len(solutionData) == 0 {
		return nil, solverProcessErrorWithDiagnostic(SolverErrorMalformed, diagnostic)
	}

	solverVariables := make(map[string]string, len(variableNames))
	for itemID, solverName := range variableNames {
		solverVariables[solverName] = itemID
	}
	status, solution, parseError := parseCLPSolution(solutionData, solverVariables)
	if parseError != nil {
		return nil, solverProcessErrorWithDiagnostic(SolverErrorMalformed, solverDiagnostic(jobDir, []byte(parseError.Error()), nil))
	}
	switch status {
	case SolverStatusInfeasible:
		return nil, solverStatusError(SolverErrorInfeasible, status, diagnostic)
	case SolverStatusUnbounded:
		return nil, solverStatusError(SolverErrorUnbounded, status, diagnostic)
	case SolverStatusOptimal:
		return solution, nil
	default:
		return nil, solverProcessErrorWithDiagnostic(SolverErrorMalformed, diagnostic)
	}
}

// CheckVersion verifies the exact CLP version before a worker accepts jobs.
// Implements DESIGN-004 LPSolverWrapper startup readiness.
func (s *LPSolverWrapper) CheckVersion(ctx context.Context) error {
	if ctx == nil {
		return solverValidationError("solver context is required")
	}
	config, err := s.validatedConfig()
	if err != nil {
		return err
	}
	runContext, cancel := context.WithTimeout(ctx, SolverDeadline)
	defer cancel()
	var stdout, stderr limitedBuffer
	runError := config.Runner(runContext, config.Executable, []string{"-version"}, &stdout, &stderr)
	diagnostic := solverDiagnostic("", stdout.Bytes(), stderr.Bytes())
	if runContext.Err() != nil {
		return contextSolverError(runContext.Err(), diagnostic)
	}
	if runError != nil {
		if errors.Is(runError, exec.ErrNotFound) {
			return solverProcessErrorWithDiagnostic(SolverErrorUnavailable, diagnostic)
		}
		return solverProcessErrorWithDiagnostic(SolverErrorNonZero, diagnostic)
	}
	if stdout.Truncated() || stderr.Truncated() {
		return solverProcessErrorWithDiagnostic(SolverErrorOutputLimit, diagnostic)
	}
	version := clpVersion(append(stdout.Bytes(), stderr.Bytes()...))
	if version != config.ExpectedVersion {
		return solverProcessErrorWithDiagnostic(SolverErrorVersion, "expected "+config.ExpectedVersion+", found "+version)
	}
	return nil
}

// StartupCheck is a descriptive alias used by worker readiness code.
// Implements DESIGN-004 LPSolverWrapper startup readiness.
func (s *LPSolverWrapper) StartupCheck(ctx context.Context) error {
	return s.CheckVersion(ctx)
}

// validatedConfig applies safe deployment defaults and rejects unsafe command configuration.
// Implements DESIGN-004 LPSolverWrapper.
func (s *LPSolverWrapper) validatedConfig() (CLPConfig, error) {
	if s == nil {
		return CLPConfig{}, solverValidationError("solver wrapper is required")
	}
	config := s.config
	if config.Executable == "" {
		config.Executable = DefaultCLPExecutable
	}
	if strings.TrimSpace(config.Executable) != config.Executable || strings.ContainsAny(config.Executable, "\x00\r\n\t ") || strings.HasPrefix(config.Executable, "-") {
		return CLPConfig{}, solverValidationError("CLP executable must be a single configured path")
	}
	if config.ExpectedVersion == "" {
		config.ExpectedVersion = SupportedCLPVersion
	}
	if !versionPattern.MatchString(config.ExpectedVersion) {
		return CLPConfig{}, solverValidationError("CLP expected version must be major.minor.patch")
	}
	if config.Timeout == 0 {
		config.Timeout = SolverDeadline
	}
	if config.Timeout < 0 || config.Timeout > SolverDeadline {
		return CLPConfig{}, solverValidationError("CLP timeout must be between zero and 30 seconds")
	}
	if config.Runner == nil {
		config.Runner = runOSCommand
	}
	return config, nil
}

// runOSCommand executes CLP without a shell so context cancellation reaches the child process.
// Implements DESIGN-004 LPSolverWrapper.
func runOSCommand(ctx context.Context, executable string, args []string, stdout, stderr io.Writer) error {
	command := exec.CommandContext(ctx, executable, args...)
	command.Stdout = stdout
	command.Stderr = stderr
	return command.Run()
}

// serializeLP converts validated internal model data to generated-name LP syntax.
// Implements DESIGN-004 LPSolverWrapper.
func serializeLP(model LPModel, objective ObjectiveFunction) ([]byte, map[string]string, error) {
	if len(model.Variables) == 0 {
		return nil, nil, solverValidationError("LP model requires at least one variable")
	}
	variableNames := make(map[string]string, len(model.Variables))
	for index, variable := range model.Variables {
		if variable.ItemID == "" {
			return nil, nil, solverValidationError("LP variable item ID is required")
		}
		if _, exists := variableNames[variable.ItemID]; exists {
			return nil, nil, solverValidationError("LP variable item IDs must be unique")
		}
		if !finite(variable.LowerBound) || !finite(variable.UpperBound) || variable.LowerBound < 0 || variable.UpperBound < variable.LowerBound {
			return nil, nil, solverValidationError("LP variable bounds are invalid")
		}
		for _, coefficient := range []float64{variable.CaloriesPerUnit, variable.DiversityPenalty, variable.ProteinPerUnit, variable.CarbohydratesPerUnit, variable.FatPerUnit} {
			if !finite(coefficient) {
				return nil, nil, solverValidationError("LP variable coefficients must be finite")
			}
		}
		variableNames[variable.ItemID] = fmt.Sprintf("x%06d", index+1)
	}
	if len(objective.Coefficients) != len(model.Variables) {
		return nil, nil, solverValidationError("LP objective must define every variable")
	}
	for itemID, coefficient := range objective.Coefficients {
		if _, exists := variableNames[itemID]; !exists || !finite(coefficient) {
			return nil, nil, solverValidationError("LP objective contains an invalid variable or coefficient")
		}
	}

	var output bytes.Buffer
	output.WriteString("Minimize\n obj: ")
	writeExpression(&output, objective.Coefficients, variableNames, model.Variables)
	output.WriteString("\nSubject To\n")
	for index, constraint := range model.Constraints {
		if !finite(constraint.LowerBound) || !finite(constraint.UpperBound) || constraint.UpperBound < constraint.LowerBound || len(constraint.Coefficients) == 0 {
			return nil, nil, solverValidationError("LP constraint is invalid")
		}
		for itemID, coefficient := range constraint.Coefficients {
			if _, exists := variableNames[itemID]; !exists || !finite(coefficient) {
				return nil, nil, solverValidationError("LP constraint contains an invalid variable or coefficient")
			}
		}
		output.WriteString(fmt.Sprintf(" c%06d: ", index+1))
		writeExpression(&output, constraint.Coefficients, variableNames, model.Variables)
		if constraint.LowerBound == constraint.UpperBound {
			output.WriteString(" = " + formatSolverFloat(constraint.LowerBound))
		} else {
			output.WriteString(" >= " + formatSolverFloat(constraint.LowerBound))
			output.WriteString("\n c" + fmt.Sprintf("%06d", index+1) + "_upper: ")
			writeExpression(&output, constraint.Coefficients, variableNames, model.Variables)
			output.WriteString(" <= " + formatSolverFloat(constraint.UpperBound))
		}
		output.WriteByte('\n')
	}
	output.WriteString("Bounds\n")
	for index, variable := range model.Variables {
		name := fmt.Sprintf("x%06d", index+1)
		output.WriteString(" " + formatSolverFloat(variable.LowerBound) + " <= " + name + " <= " + formatSolverFloat(variable.UpperBound) + "\n")
	}
	output.WriteString("End\n")
	if output.Len() > maxSolverModelBytes {
		return nil, nil, solverProcessErrorWithDiagnostic(SolverErrorOutputLimit, "serialized model exceeded the safety limit")
	}
	return output.Bytes(), variableNames, nil
}

// writeExpression writes a deterministic generated-name linear expression.
// Implements DESIGN-004 LPSolverWrapper.
func writeExpression(output *bytes.Buffer, coefficients map[string]float64, variableNames map[string]string, variables []LPVariable) {
	first := true
	for _, variable := range variables {
		coefficient, exists := coefficients[variable.ItemID]
		if !exists || coefficient == 0 {
			continue
		}
		if !first {
			if coefficient < 0 {
				output.WriteString(" - ")
			} else {
				output.WriteString(" + ")
			}
		} else if coefficient < 0 {
			output.WriteString("-")
		}
		output.WriteString(formatSolverFloat(math.Abs(coefficient)))
		output.WriteByte(' ')
		output.WriteString(variableNames[variable.ItemID])
		first = false
	}
	if first {
		output.WriteByte('0')
	}
}

// Implements DESIGN-004 LPSolverWrapper parser patterns.
var (
	versionPattern     = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	solverVariableName = regexp.MustCompile(`^x\d{6}$`)
)

// clpVersion extracts one exact semantic version token from bounded startup output.
// Implements DESIGN-004 LPSolverWrapper.
func clpVersion(output []byte) string {
	for _, token := range strings.Fields(string(output)) {
		token = strings.Trim(token, ",;()")
		if versionPattern.MatchString(token) {
			return token
		}
	}
	return "unknown"
}

// parseCLPSolution parses only known generated variables and terminal statuses.
// Implements DESIGN-004 LPSolverWrapper.
func parseCLPSolution(output []byte, variableNames map[string]string) (SolverStatus, LPSolution, error) {
	var status SolverStatus
	result := make(LPSolution)
	seenVariable := make(map[string]struct{}, len(variableNames))
	sawVariable := false
	for _, line := range strings.Split(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		lineStatus := SolverStatus("")
		switch {
		case strings.HasPrefix(lower, "optimal"):
			lineStatus = SolverStatusOptimal
		case strings.HasPrefix(lower, "infeasible"):
			lineStatus = SolverStatusInfeasible
		case strings.HasPrefix(lower, "unbounded"):
			lineStatus = SolverStatusUnbounded
		}
		if lineStatus != "" {
			if status != "" && status != lineStatus {
				return "", nil, errors.New("conflicting solver statuses")
			}
			status = lineStatus
		}
		fields := strings.Fields(line)
		for index, field := range fields {
			itemID, known := variableNames[field]
			if !known {
				if solverVariableName.MatchString(field) {
					return "", nil, errors.New("unknown solver variable")
				}
				continue
			}
			sawVariable = true
			if _, duplicate := seenVariable[field]; duplicate {
				return "", nil, errors.New("duplicate solver variable")
			}
			if index+1 >= len(fields) {
				return "", nil, errors.New("missing solver quantity")
			}
			value, err := strconv.ParseFloat(fields[index+1], 64)
			if err != nil || !finite(value) || value < 0 {
				return "", nil, errors.New("invalid solver quantity")
			}
			seenVariable[field] = struct{}{}
			if value > 0 {
				result[itemID] = value
			}
		}
	}
	if status == "" {
		return "", nil, errors.New("solver status is missing")
	}
	if status == SolverStatusOptimal && !sawVariable {
		return "", nil, errors.New("optimal solver output has no variables")
	}
	return status, result, nil
}

// limitedBuffer bounds process output before it can enter memory or diagnostics.
// Implements DESIGN-004 LPSolverWrapper.
type limitedBuffer struct {
	buffer    bytes.Buffer
	truncated bool
}

// Write retains at most MaxSolverOutputBytes while reporting the full write to exec.Cmd.
// Implements DESIGN-004 LPSolverWrapper.
func (b *limitedBuffer) Write(data []byte) (int, error) {
	remaining := MaxSolverOutputBytes - b.buffer.Len()
	if remaining > 0 {
		if len(data) > remaining {
			_, _ = b.buffer.Write(data[:remaining])
			b.truncated = true
		} else {
			_, _ = b.buffer.Write(data)
		}
	} else if len(data) > 0 {
		b.truncated = true
	}
	return len(data), nil
}

// Bytes returns the bounded captured output.
// Implements DESIGN-004 LPSolverWrapper.
func (b *limitedBuffer) Bytes() []byte {
	return b.buffer.Bytes()
}

// Truncated reports whether the process attempted to exceed the output limit.
// Implements DESIGN-004 LPSolverWrapper.
func (b *limitedBuffer) Truncated() bool {
	return b.truncated
}

// readBoundedFile reads a solver solution file without allowing unbounded growth.
// Implements DESIGN-004 LPSolverWrapper.
func readBoundedFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, MaxSolverOutputBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxSolverOutputBytes {
		return nil, ErrSolverOutputLimit
	}
	return data, nil
}

// solverDiagnostic combines and sanitizes bounded child-process diagnostics.
// Implements DESIGN-004 LPSolverWrapper.
func solverDiagnostic(jobDir string, outputs ...[]byte) string {
	var combined strings.Builder
	for _, output := range outputs {
		if len(output) == 0 {
			continue
		}
		if combined.Len() > 0 {
			combined.WriteByte('\n')
		}
		combined.Write(output)
	}
	diagnostic := sanitizeSolverOutput(combined.String())
	if jobDir != "" {
		diagnostic = strings.ReplaceAll(diagnostic, jobDir, "<job-dir>")
	}
	if len(diagnostic) > maxSolverDiagnostic {
		diagnostic = diagnostic[:maxSolverDiagnostic] + "..."
	}
	return diagnostic
}

// sanitizeSolverOutput removes control characters from process diagnostics.
// Implements DESIGN-004 LPSolverWrapper.
func sanitizeSolverOutput(output string) string {
	var sanitized strings.Builder
	for _, character := range output {
		switch {
		case character == '\n' || character == '\t' || character >= ' ' && character != 0x7f:
			sanitized.WriteRune(character)
		default:
			sanitized.WriteByte(' ')
		}
	}
	return strings.TrimSpace(sanitized.String())
}

// formatSolverFloat renders finite LP coefficients in CLP-compatible form.
// Implements DESIGN-004 LPSolverWrapper.
func formatSolverFloat(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}

// solverValidationError creates a safe wrapper-input validation error.
// Implements DESIGN-004 LPSolverWrapper.
func solverValidationError(message string) error {
	return &SolverError{Kind: SolverErrorValidation, Diagnostic: sanitizeSolverOutput(message)}
}

// solverProcessErrorWithDiagnostic creates a classified process error with sanitized output.
// Implements DESIGN-004 LPSolverWrapper.
func solverProcessErrorWithDiagnostic(kind SolverErrorKind, diagnostic string) error {
	return &SolverError{Kind: kind, Diagnostic: sanitizeSolverOutput(diagnostic)}
}

// solverStatusError creates a classified terminal-status error.
// Implements DESIGN-004 LPSolverWrapper.
func solverStatusError(kind SolverErrorKind, status SolverStatus, diagnostic string) error {
	return &SolverError{Kind: kind, Status: status, Diagnostic: sanitizeSolverOutput(diagnostic)}
}

// contextSolverError maps a canceled or deadline-exceeded child process to a safe solver error.
// Implements DESIGN-004 LPSolverWrapper.
func contextSolverError(err error, diagnostic string) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return solverProcessErrorWithDiagnostic(SolverErrorTimeout, diagnostic)
	}
	return solverProcessErrorWithDiagnostic(SolverErrorCanceled, diagnostic)
}
