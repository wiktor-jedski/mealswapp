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
- `JobStatusTracker`: owns per-user admission, job status persistence, polling responses, result TTL, and failure messages.

### 1. Data Structures & Types
- `type JobStatus = "queued" | "processing" | "completed" | "failed" | "cancelled"`
- `interface SavedDietOptimizationRequest { dailyDietId: UUID; excludedIds: string[]; tolerancePercent: number }`
- `interface MacroTarget { protein: number; carbs: number; fat: number }`
- `interface OptimizationJob { jobId: string; userId: string; request: DietOptimizationRequest; status: JobStatus; createdAt: time.Time; startedAt?: time.Time; finishedAt?: time.Time; error?: string }`
- `interface LPVariable { itemId: string; quantity: number; caloriesPerUnit: number; proteinPerUnit: number; carbsPerUnit: number; fatPerUnit: number; diversityPenalty: number }`
- `interface LPConstraint { name: string; lowerBound: number; upperBound: number; coefficients: map[string]float64 }`
- `interface DietAlternative { meals: MealQuantity[]; macros: MacroTarget; calories: number; similarityScore: number }`

### 2. Logic & Algorithms (Step-by-Step)
1. API handler validates the request and entitlement, replays an existing idempotency result without consuming capacity, then uses Redis `SET NX` to reserve the authenticated user's single active-job slot. A fixed-hour Redis counter admits at most 10 newly accepted jobs; rejected admission creates no idempotency row, job record, or stream entry.
2. Enqueue the server-created job ID with `XADD` in a Redis Stream; return `202 Accepted` with poll URL.
3. A dedicated worker reserves deliveries through one `XREADGROUP` consumer group, reclaiming abandoned pending entries with `XAUTOCLAIM`, and loads eligible candidate meals from ARCH-005.
4. Build one LP variable per candidate meal quantity.
5. Derive Protein, Carbohydrate, and Fat targets from the current server-owned saved-diet entries, then create constraints using those targets and the client-selected tolerance band. Standalone client-authored macro targets are deferred to Phase 10.
6. Build the objective as total calories plus diversity penalties for meals from the original diet.
7. Apply one hard 30-second deadline to repository loading and all alternative-generation solver attempts for the job. Run each pinned native `clp` invocation with `exec.CommandContext` under that shared context and retain the wrapper's 30-second per-process ceiling as a defensive upper bound. The wrapper writes a private per-job LP file and solution file, then unconditionally removes the temporary directory.
8. Validate the solution: finite quantities, macro tolerance satisfied, excluded IDs absent, and calories present.
9. Generate up to 3 alternatives by adding exclusion constraints for previously selected high-weight items and solving again.
10. Store completed or failed result in Redis with 1-hour TTL and update job status for polling.

### 3. State Management & Error Handling
- `queued`: accepted but not yet started.
- `processing`: worker owns the job; poll returns progress metadata only.
- `completed`: result exists and can be returned until TTL expires.
- `failed_validation`: request cannot produce valid constraints; fail without enqueueing.
- `solver_timeout`: mark failed after the single 30-second whole-job deadline and include partial alternatives if any passed validation; use a separate bounded finalization context so the terminal state can be published before ownership visibility expires.
- `solver_infeasible`: mark failed with a user-safe message that no combination matches the targets.
- `queue_unavailable`: return 503 from submission; do not run synchronously.
- `optimization_in_progress`: return 429 with `Retry-After` when the authenticated user already owns another queued or processing job.
- `optimization_rate_limited`: return 429 with `Retry-After` after 10 newly accepted jobs in one fixed UTC hour; exact idempotency repair does not increment the counter.
- `worker_crash`: Redis visibility timeout longer than the 30-second solver deadline returns the pending delivery through `XAUTOCLAIM`; `XACK` is terminal and the queue stops after three attempts.

The worker performs a bounded startup `clp -version` check and fails readiness unless the packaged executable reports the supported pinned version. The dedicated `linux/amd64` optimizer image contains the CGO-disabled Go worker and the checksum-pinned official CLP `1.17.11` Ubuntu 24 release artifact on an Ubuntu 24.04 runtime; CLP is a child process inside that container, not a separate network service. Model IDs and constraint names are mapped to generated solver names; they never become filenames or subprocess arguments. CLP stdout, stderr, and solution files are bounded and control-character sanitized before diagnostics are retained.

### 4. Component Interfaces
- `func (c *OptimizationController) Submit(ctx *fiber.Ctx) error`
- `func (c *OptimizationController) GetJob(ctx *fiber.Ctx) error`
- `func (g *RedisOptimizationAdmissionGate) Acquire(ctx context.Context, req OptimizationAdmissionRequest) (OptimizationAdmissionDecision, error)`
- `func (g *RedisOptimizationAdmissionGate) Release(ctx context.Context, userID UUID, jobID UUID) error`
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
