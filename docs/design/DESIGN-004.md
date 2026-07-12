## FILE: DESIGN-004.md
**Traceability:** ARCH-004

**Static aspects covered:** LPSolverWrapper, ConstraintBuilder, ObjectiveFunction, DiversityPenalizer, SolutionValidator, JobQueueManager, JobStatusTracker.

### 0. Static Aspect Responsibilities
- `LPSolverWrapper`: owns pure-Go serialization of validated LP models, invocation of the pinned native COIN-OR CLP executable as an OS child process, context deadlines, bounded output parsing, and solver result conversion. It does not link a CLP library through CGO and is never called from the Fiber API process.
- `ConstraintBuilder`: owns protein, carbohydrate, fat, exclusion, and multi-solution constraints.
- `ObjectiveFunction`: owns calorie minimization coefficients.
- `DiversityPenalizer`: owns penalty coefficients for meals already present in the original diet.
- `SolutionValidator`: owns feasibility, macro tolerance, excluded ID, and finite quantity checks.
- `JobQueueManager`: owns Redis-backed enqueue, reservation, retry, and worker execution.
- `JobStatusTracker`: owns job status persistence, polling responses, result TTL, and failure messages.

### 1. Data Structures & Types
- `type JobStatus = "queued" | "processing" | "completed" | "failed" | "cancelled"`
- `interface DietOptimizationRequest { originalMeals: Meal[]; targetMacros: MacroTarget; excludedIds: string[]; tolerancePercent: number }`
- `interface MacroTarget { protein: number; carbs: number; fat: number }`
- `interface OptimizationJob { jobId: string; userId: string; request: DietOptimizationRequest; status: JobStatus; createdAt: time.Time; startedAt?: time.Time; finishedAt?: time.Time; error?: string }`
- `interface LPVariable { itemId: string; quantity: number; caloriesPerUnit: number; proteinPerUnit: number; carbsPerUnit: number; fatPerUnit: number; diversityPenalty: number }`
- `interface LPConstraint { name: string; lowerBound: number; upperBound: number; coefficients: map[string]float64 }`
- `interface DietAlternative { meals: MealQuantity[]; macros: MacroTarget; calories: number; similarityScore: number }`

### 2. Logic & Algorithms (Step-by-Step)
1. API handler validates the request and writes an `OptimizationJob` with `queued` status.
2. Enqueue the server-created job ID with `XADD` in a Redis Stream; return `202 Accepted` with poll URL.
3. A dedicated worker reserves deliveries through one `XREADGROUP` consumer group, reclaiming abandoned pending entries with `XAUTOCLAIM`, and loads eligible candidate meals from ARCH-005.
4. Build one LP variable per candidate meal quantity.
5. Create protein, carbohydrate, and fat constraints using target macros and tolerance bands.
6. Build the objective as total calories plus diversity penalties for meals from the original diet.
7. Run the pinned native `clp` executable with `exec.CommandContext` and a hard 30-second worker deadline. The wrapper writes a private per-job LP file and solution file, then unconditionally removes the temporary directory.
8. Validate the solution: finite quantities, macro tolerance satisfied, excluded IDs absent, and calories present.
9. Generate up to 3 alternatives by adding exclusion constraints for previously selected high-weight items and solving again.
10. Store completed or failed result in Redis with 1-hour TTL and update job status for polling.

### 3. State Management & Error Handling
- `queued`: accepted but not yet started.
- `processing`: worker owns the job; poll returns progress metadata only.
- `completed`: result exists and can be returned until TTL expires.
- `failed_validation`: request cannot produce valid constraints; fail without enqueueing.
- `solver_timeout`: mark failed after 30 seconds and include partial alternatives if any passed validation.
- `solver_infeasible`: mark failed with a user-safe message that no combination matches the targets.
- `queue_unavailable`: return 503 from submission; do not run synchronously.
- `worker_crash`: Redis visibility timeout longer than the 30-second solver deadline returns the pending delivery through `XAUTOCLAIM`; `XACK` is terminal and the queue stops after three attempts.

The worker performs a bounded startup `clp -version` check and fails readiness unless the packaged executable reports the supported pinned version. Model IDs and constraint names are mapped to generated solver names; they never become filenames or subprocess arguments. CLP stdout, stderr, and solution files are bounded and control-character sanitized before diagnostics are retained.

### 4. Component Interfaces
- `func (c *OptimizationController) Submit(ctx *fiber.Ctx) error`
- `func (c *OptimizationController) GetJob(ctx *fiber.Ctx) error`
- `func EnqueueOptimizationJob(ctx context.Context, job OptimizationJob) error`
- `func ProcessOptimizationJob(ctx context.Context, jobID string) error`
- `func (q *JobQueueManager) Enqueue(ctx context.Context, jobID string) (string, error)`
- `func (q *JobQueueManager) Reserve(ctx context.Context) (Job, error)`
- `func (q *JobQueueManager) Reclaim(ctx context.Context, minIdle time.Duration) ([]Job, error)`
- `func (q *JobQueueManager) Ack(ctx context.Context, job Job) error`
- `func BuildConstraints(req DietOptimizationRequest, vars []LPVariable) []LPConstraint`
- `func BuildObjective(vars []LPVariable) ObjectiveFunction`
- `func (s *LPSolverWrapper) CheckVersion(ctx context.Context) error`
- `func (s *LPSolverWrapper) Solve(ctx context.Context, model LPModel, objective ObjectiveFunction) (LPSolution, error)`
- `func ValidateSolution(solution LPSolution, req DietOptimizationRequest) (DietAlternative, error)`
- `func GenerateAlternatives(ctx context.Context, req DietOptimizationRequest, limit int) ([]DietAlternative, error)`
