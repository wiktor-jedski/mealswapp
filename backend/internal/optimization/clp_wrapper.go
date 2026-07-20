package optimization

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
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

// commandRunner is a trusted package-only test seam. Implementations must not
// return before the supplied context settles; production always uses
// exec.CommandContext through runOSCommand.
// Implements DESIGN-004 LPSolverWrapper.
type commandRunner func(ctx context.Context, executable string, args []string, stdout, stderr io.Writer) error

// CLPConfig configures the native CLP executable. Executable and ExpectedVersion
// are deployment configuration, not request data. Timeout may be shortened for
// tests but cannot exceed SolverDeadline.
// Implements DESIGN-004 LPSolverWrapper deployment configuration.
type CLPConfig struct {
	Executable      string
	ExpectedVersion string
	TempRoot        string
	Timeout         time.Duration
	runner          commandRunner
	removeAll       func(string) error
	observeCleanup  func(string)
}

// LPSolverWrapper invokes a pinned native CLP executable without CGO.
// Implements DESIGN-004 LPSolverWrapper.
type LPSolverWrapper struct {
	config CLPConfig
}

// NewLPSolverWrapper constructs a pure-Go CLP child-process wrapper.
// Implements DESIGN-004 LPSolverWrapper.
func NewLPSolverWrapper(config CLPConfig) *LPSolverWrapper {
	return &LPSolverWrapper{config: config}
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
	defer cleanupSolverDirectory(config, jobDir)

	modelPath := filepath.Join(jobDir, "model.lp")
	solutionPath := filepath.Join(jobDir, "solution.txt")
	if err := os.WriteFile(modelPath, modelData, 0o600); err != nil {
		return nil, solverProcessErrorWithDiagnostic(SolverErrorUnavailable, "cannot write solver model")
	}

	runContext, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()
	var stdout, stderr limitedBuffer
	runError := config.runner(runContext, config.Executable, []string{
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
	runError := config.runner(runContext, config.Executable, []string{"-version"}, &stdout, &stderr)
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
	if config.runner == nil {
		config.runner = runOSCommand
	}
	if config.removeAll == nil {
		config.removeAll = os.RemoveAll
	}
	if config.observeCleanup == nil {
		config.observeCleanup = func(diagnostic string) {
			log.Printf("CLP cleanup failure: %s", diagnostic)
		}
	}
	return config, nil
}

// cleanupSolverDirectory reports one bounded, sanitized cleanup failure while
// preserving the Solve result selected before the deferred cleanup runs.
// Implements DESIGN-004 LPSolverWrapper cleanup observability.
func cleanupSolverDirectory(config CLPConfig, jobDir string) {
	if err := config.removeAll(jobDir); err != nil {
		config.observeCleanup(solverDiagnostic(jobDir, []byte(err.Error())))
	}
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
	return serializeLPWithLimit(model, objective, maxSolverModelBytes)
}

// serializeLPWithLimit renders a validated model under an explicit byte ceiling.
// Implements DESIGN-004 LPSolverWrapper bounded LP serialization.
func serializeLPWithLimit(model LPModel, objective ObjectiveFunction, limit int) ([]byte, map[string]string, error) {
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

	output := boundedModelWriter{limit: limit}
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
		output.AppendByte(' ')
		output.WriteString(canonicalConstraintName(index))
		output.WriteString(": ")
		writeExpression(&output, constraint.Coefficients, variableNames, model.Variables)
		if constraint.LowerBound == constraint.UpperBound {
			output.WriteString(" = ")
			output.WriteString(formatSolverFloat(constraint.LowerBound))
		} else {
			output.WriteString(" >= ")
			output.WriteString(formatSolverFloat(constraint.LowerBound))
			output.WriteString("\n ")
			output.WriteString(canonicalConstraintName(index))
			output.WriteString("_upper: ")
			writeExpression(&output, constraint.Coefficients, variableNames, model.Variables)
			output.WriteString(" <= ")
			output.WriteString(formatSolverFloat(constraint.UpperBound))
		}
		output.AppendByte('\n')
		if output.Err() != nil {
			return nil, nil, serializedModelLimitError()
		}
	}
	output.WriteString("Bounds\n")
	for _, variable := range model.Variables {
		output.AppendByte(' ')
		output.WriteString(formatSolverFloat(variable.LowerBound))
		output.WriteString(" <= ")
		output.WriteString(variableNames[variable.ItemID])
		output.WriteString(" <= ")
		output.WriteString(formatSolverFloat(variable.UpperBound))
		output.AppendByte('\n')
		if output.Err() != nil {
			return nil, nil, serializedModelLimitError()
		}
	}
	output.WriteString("End\n")
	if output.Err() != nil {
		return nil, nil, serializedModelLimitError()
	}
	return output.Bytes(), variableNames, nil
}

// boundedModelWriter rejects a model as soon as the next write would exceed
// the serialization limit, keeping allocation bounded by maxSolverModelBytes.
// Implements DESIGN-004 LPSolverWrapper bounded LP serialization.
type boundedModelWriter struct {
	buffer bytes.Buffer
	err    error
	limit  int
}

// WriteString appends a string unless it would exceed the model byte ceiling.
// Implements DESIGN-004 LPSolverWrapper bounded LP serialization.
func (w *boundedModelWriter) WriteString(value string) {
	if w.err != nil || len(value) > w.limit-w.buffer.Len() {
		w.err = ErrSolverOutputLimit
		return
	}
	_, _ = w.buffer.WriteString(value)
}

// AppendByte appends one byte unless it would exceed the model byte ceiling.
// Implements DESIGN-004 LPSolverWrapper bounded LP serialization.
func (w *boundedModelWriter) AppendByte(value byte) {
	if w.err != nil || w.buffer.Len() == w.limit {
		w.err = ErrSolverOutputLimit
		return
	}
	_ = w.buffer.WriteByte(value)
}

// Bytes returns the bounded serialized model.
// Implements DESIGN-004 LPSolverWrapper bounded LP serialization.
func (w *boundedModelWriter) Bytes() []byte { return w.buffer.Bytes() }

// Err reports whether serialization exceeded its byte ceiling.
// Implements DESIGN-004 LPSolverWrapper bounded LP serialization.
func (w *boundedModelWriter) Err() error { return w.err }

// canonicalConstraintName returns the generated name for a constraint index.
// Implements DESIGN-004 LPSolverWrapper generated solver names.
func canonicalConstraintName(index int) string { return fmt.Sprintf("c%06d", index+1) }

// serializedModelLimitError classifies an LP serialization limit failure.
// Implements DESIGN-004 LPSolverWrapper bounded LP serialization.
func serializedModelLimitError() error {
	return solverProcessErrorWithDiagnostic(SolverErrorOutputLimit, "serialized model exceeded the safety limit")
}

// writeExpression writes a deterministic generated-name linear expression.
// Implements DESIGN-004 LPSolverWrapper.
func writeExpression(output *boundedModelWriter, coefficients map[string]float64, variableNames map[string]string, variables []LPVariable) {
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
		output.AppendByte(' ')
		output.WriteString(variableNames[variable.ItemID])
		first = false
	}
	if first {
		output.AppendByte('0')
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
	for token := range strings.FieldsSeq(string(output)) {
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
	for line := range bytes.Lines(output) {
		lineStatus, rowName, rowValue, isRow, err := parseCLPSolutionLine(line)
		if err != nil {
			return "", nil, err
		}
		if lineStatus != "" {
			if status != "" {
				if status != lineStatus {
					return "", nil, errors.New("conflicting solver statuses")
				}
				return "", nil, errors.New("duplicate solver status")
			}
			status = lineStatus
		}
		if !isRow {
			continue
		}
		itemID, known := variableNames[rowName]
		if !known {
			return "", nil, errors.New("unknown solver variable")
		}
		sawVariable = true
		if _, duplicate := seenVariable[rowName]; duplicate {
			return "", nil, errors.New("duplicate solver variable")
		}
		seenVariable[rowName] = struct{}{}
		if rowValue > 0 {
			result[itemID] = rowValue
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

// parseCLPSolutionLine accepts only CLP's terminal header and indexed solution
// row grammar. Human-oriented diagnostics are otherwise ignored.
// Implements DESIGN-004 LPSolverWrapper exact solution codec.
func parseCLPSolutionLine(line []byte) (SolverStatus, string, float64, bool, error) {
	var fields [6][]byte
	fieldCount := 0
	extraFields := false
	for field := range bytes.FieldsSeq(line) {
		if fieldCount == len(fields) {
			extraFields = true
			continue
		}
		fields[fieldCount] = field
		fieldCount++
	}
	if fieldCount == 0 {
		return "", "", 0, false, nil
	}
	first := fields[0]
	if lineStatus := exactCLPStatus(first); lineStatus != "" {
		value, valueErr := strconv.ParseFloat(string(fields[4]), 64)
		if fieldCount != 5 || extraFields || !bytes.Equal(fields[1], []byte("-")) || !bytes.Equal(fields[2], []byte("objective")) || !bytes.Equal(fields[3], []byte("value")) || valueErr != nil || !finite(value) {
			return "", "", 0, false, errors.New("invalid solver status header")
		}
		return lineStatus, "", 0, false, nil
	}
	rowOffset := 0
	if bytes.Equal(first, []byte("**")) {
		if fieldCount != 5 || extraFields {
			return "", "", 0, false, errors.New("invalid solver solution row")
		}
		rowOffset = 1
		first = fields[rowOffset]
	}
	if _, err := strconv.ParseUint(string(first), 10, 64); err != nil {
		return "", "", 0, false, nil
	}
	if fieldCount <= rowOffset+1 || !solverVariableName.Match(fields[rowOffset+1]) {
		return "", "", 0, false, nil
	}
	if fieldCount != rowOffset+4 {
		if fieldCount == rowOffset+2 && solverVariableName.Match(fields[rowOffset+1]) {
			return "", "", 0, false, errors.New("missing solver quantity")
		}
		return "", "", 0, false, errors.New("invalid solver solution row")
	}
	name := fields[rowOffset+1]
	quantity, quantityErr := strconv.ParseFloat(string(fields[rowOffset+2]), 64)
	reducedCost, reducedCostErr := strconv.ParseFloat(string(fields[rowOffset+3]), 64)
	if quantityErr != nil || reducedCostErr != nil || !finite(quantity) || !finite(reducedCost) {
		return "", "", 0, false, errors.New("invalid solver quantity")
	}
	if quantity < -quantityTolerance(quantity) {
		return "", "", 0, false, errors.New("invalid solver quantity")
	}
	if quantity < 0 {
		quantity = 0
	}
	return "", string(name), quantity, true, nil
}

// exactCLPStatus recognizes only the supported CLP terminal status tokens.
// Implements DESIGN-004 LPSolverWrapper exact solution codec.
func exactCLPStatus(token []byte) SolverStatus {
	switch string(token) {
	case "Optimal":
		return SolverStatusOptimal
	case "Infeasible":
		return SolverStatusInfeasible
	case "Unbounded":
		return SolverStatusUnbounded
	default:
		return ""
	}
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
