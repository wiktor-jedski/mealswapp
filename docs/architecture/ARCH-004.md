# [ARCH-004] - Linear Programming Optimizer

**Description:** Asynchronous optimization service that uses linear programming to generate alternative diet combinations matching target macronutrient profiles while minimizing total calories. Operates as a job queue to handle CPU-intensive calculations without blocking the API.

| Attribute | Value |
| :--- | :--- |
| **Type** | Service (Asynchronous Job Queue) |
| **Static Aspects** | LPSolverWrapper (pure-Go wrapper around the pinned native COIN-OR CLP child process), ConstraintBuilder, ObjectiveFunction, DiversityPenalizer, SolutionValidator, JobQueueManager, JobStatusTracker |
| **Dependencies** | ARCH-003 (Similarity Engine), ARCH-005 (Data Repository), Redis Streams via `github.com/redis/go-redis/v9`, ARCH-010 (API Gateway) |
| **Traceability** | SW-REQ-021, SW-REQ-022, SW-REQ-023, SW-REQ-030, SW-REQ-080, SW-REQ-082 |

**Dynamic Behavior:**

- **Job Submission:** Client submits optimization request. API returns `202 Accepted` with a `jobId` immediately, without blocking. The server-created job ID is appended with `XADD` to a Redis Stream.
- **Asynchronous Processing:** Dedicated workers reserve through one `XREADGROUP` consumer group, reclaim abandoned pending entries with `XAUTOCLAIM`, and acknowledge terminal deliveries with `XACK`. LP solving occurs asynchronously, preventing blocking under concurrent load.
- **Constraint Setup:** Builds linear constraints for target Protein, Carbohydrate, and Fat values with configurable tolerance bands.
- **Objective Minimization:** Defines calorie count as primary objective function to minimize.
- **Diversity Weighting:** Applies penalty weights to meal IDs present in original diet to encourage diverse alternatives.
- **Multi-Solution Generation:** Iteratively solves LP with exclusion constraints to produce up to 3 distinct alternatives.
- **Result Retrieval:** Client polls `GET /jobs/{jobId}` endpoint or subscribes via WebSocket for completion notification. Results cached in Redis with 1-hour TTL.
- **Timeout Handling:** Jobs exceeding 30 seconds are terminated through `exec.CommandContext`, their private temporary solver directory is cleaned up, and they are marked as failed. Client receives partial results if available.

**Solver Boundary:** The dedicated worker serializes only validated internal `LPModel` and `ObjectiveFunction` values to a generated-name LP file. `LPSolverWrapper` invokes the packaged `clp` executable with separate fixed arguments, reads a bounded solution/status stream, sanitizes diagnostics, and converts generated variable names back to meal IDs. Worker readiness performs an exact `1.17.11` version check. The Go worker builds with `CGO_ENABLED=0`; the Fiber API process has no solver dependency or execution path.

**Interface Definition:**

- `Input`: POST /api/v1/diet/optimize -> DietOptimizationRequest { originalMeals: Meal[], targetMacros: MacroTarget, excludedIds: string[] }
- `Output (Immediate)`: { jobId: string, status: 'queued', pollUrl: string }
- `Output (Poll)`: GET /api/v1/jobs/{jobId} -> { status: 'queued'|'processing'|'completed'|'failed', result?: DietAlternative[], error?: string }
- `Output (WebSocket)`: Event { jobId: string, status: 'completed', result: DietAlternative[] }

**Alternative Analysis (BP6):**

- *Chosen Approach:* Asynchronous Job Queue with a dedicated worker pool; each worker invokes the pinned native COIN-OR CLP executable through an injectable pure-Go wrapper
- *Alternative Considered:* Synchronous LP execution within API request lifecycle
- *Trade-off:* Synchronous execution would block the Go Fiber event loop during CPU-intensive LP solving. With 1000 concurrent users (SW-REQ-082) and 200+ simultaneous diet searches, this creates a self-inflicted DoS condition, failing SW-REQ-080 (<2s response) and SW-REQ-081 (99.9% availability). Asynchronous queue isolates CPU work, maintains API responsiveness, and allows horizontal scaling of worker processes independently. A child process avoids CGO and library ABI coupling while the fixed deadline and bounded parser contain solver failures.

**Reference Documentation:** 
- 02_APPENDIX_A.md
