## Phase 7: Linear Programming Optimizer

**Goal:** Implement async diet optimization with job queue

### Components & Static Aspects

#### ARCH-004 - Linear Programming Optimizer
| Static Aspect | Description | File |
|:--------------|:------------|:-----|
| **LPSolverWrapper** | Wrapper for go-coinor/clp solver | `optimizer/solver.go` |
| **ConstraintBuilder** | Build P, C, F constraints with tolerance bands | `optimizer/constraints.go` |
| **ObjectiveFunction** | Minimize total calories | `optimizer/objective.go` |
| **DiversityPenalizer** | Penalty weight for overlapping meal IDs | `optimizer/diversity.go` |
| **SolutionValidator** | Validate LP solutions meet all constraints | `optimizer/validator.go` |
| **JobQueueManager** | Redis-backed job queue (go-redis/queue or machinery) | `optimizer/job_manager.go` |
| **JobStatusTracker** | Track job status (queued/processing/completed/failed) | `optimizer/status_tracker.go` |

### Testing
- [ ] Job submission returns 202 immediately (non-blocking)
- [ ] Worker processes jobs from Redis queue
- [ ] LP constraints enforce P, C, F tolerance bands
- [ ] Calorie minimization objective works
- [ ] Diversity penalty reduces overlap with original diet
- [ ] Max 3 alternative combinations returned
- [ ] 30-second timeout terminates long jobs (marks as failed)
- [ ] Job status polling returns correct states
- [ ] WebSocket notifications fire on completion
- [ ] Results cached with 1-hour TTL

---

