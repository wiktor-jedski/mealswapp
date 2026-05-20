import type { ApiClient } from '../api/client';
import type { AppError, DietAlternative, DietOptimizationRequest, JobStatus, OptimizationJob } from '../api/types';

export type OptimizationUiStatus = 'idle' | 'submitting' | 'queued' | 'processing' | 'completed' | 'failed' | 'cancelled';

export interface OptimizationState {
  status: OptimizationUiStatus;
  jobId?: string;
  pollUrl?: string;
  progress: number;
  alternatives: DietAlternative[];
  partialAlternatives: DietAlternative[];
  error?: AppError;
  message?: string;
}

export interface OptimizationControllerOptions {
  api: Pick<ApiClient, 'submitOptimizationJob' | 'getOptimizationJob'>;
  pollIntervalMs?: number;
  setIntervalFn?: (callback: () => void, delay: number) => unknown;
  clearIntervalFn?: (timer: unknown) => void;
}

export function createDefaultOptimizationState(): OptimizationState {
  return {
    status: 'idle',
    progress: 0,
    alternatives: [],
    partialAlternatives: []
  };
}

export function createOptimizationController(options: OptimizationControllerOptions) {
  const pollIntervalMs = options.pollIntervalMs ?? 1000;
  const setPollTimer = options.setIntervalFn ?? ((callback: () => void, delay: number) => setInterval(callback, delay));
  const clearPollTimer = options.clearIntervalFn ?? ((timer: unknown) => clearInterval(timer as ReturnType<typeof setInterval>));
  let state = createDefaultOptimizationState();
  let timer: unknown;
  const listeners = new Set<(state: OptimizationState) => void>();

  function subscribe(listener: (state: OptimizationState) => void): () => void {
    listeners.add(listener);
    listener(state);
    return () => listeners.delete(listener);
  }

  function emit(next: Partial<OptimizationState>): OptimizationState {
    state = { ...state, ...next };
    for (const listener of listeners) {
      listener(state);
    }
    return state;
  }

  async function submit(request: DietOptimizationRequest): Promise<OptimizationState> {
    stopPolling();
    emit({
      status: 'submitting',
      progress: 0,
      jobId: undefined,
      pollUrl: undefined,
      alternatives: [],
      partialAlternatives: [],
      error: undefined,
      message: undefined
    });
    try {
      const accepted = await options.api.submitOptimizationJob(request);
      emit({ status: statusFromJob(accepted.status), jobId: accepted.jobId, pollUrl: accepted.pollUrl, progress: 0 });
      startPolling(accepted.jobId);
      return state;
    } catch (error) {
      return emit({
        status: 'failed',
        progress: 100,
        error: normalizeAppError(error),
        message: normalizeAppError(error).message
      });
    }
  }

  async function poll(jobId = state.jobId): Promise<OptimizationState> {
    if (!jobId || state.status === 'cancelled') {
      return state;
    }
    try {
      const job = await options.api.getOptimizationJob(jobId);
      return applyJob(job);
    } catch (error) {
      stopPolling();
      return emit({
        status: 'failed',
        progress: 100,
        error: normalizeAppError(error),
        message: normalizeAppError(error).message
      });
    }
  }

  function cancel(): OptimizationState {
    stopPolling();
    return emit({ status: 'cancelled', message: 'Optimization cancelled.', progress: 100 });
  }

  function applyJob(job: OptimizationJob): OptimizationState {
    const nextStatus = statusFromJob(job.status);
    const result = job.result ?? [];
    if (nextStatus === 'completed') {
      stopPolling();
      return emit({
        status: 'completed',
        jobId: job.jobId,
        progress: 100,
        alternatives: result,
        partialAlternatives: [],
        error: undefined,
        message: undefined
      });
    }
    if (nextStatus === 'failed') {
      stopPolling();
      return emit({
        status: 'failed',
        jobId: job.jobId,
        progress: 100,
        alternatives: [],
        partialAlternatives: result,
        error: job.error ? { category: job.error.toLowerCase().includes('timed out') ? 'timeout' : 'server', code: 'optimization_failed', message: job.error, retryable: true } : undefined,
        message: timeoutMessage(job.error)
      });
    }
    return emit({
      status: nextStatus,
      jobId: job.jobId,
      progress: job.progress ?? progressForStatus(job.status),
      error: undefined,
      message: undefined
    });
  }

  function startPolling(jobId: string): void {
    stopPolling();
    timer = setPollTimer(() => {
      void poll(jobId);
    }, pollIntervalMs);
  }

  function stopPolling(): void {
    if (timer) {
      clearPollTimer(timer);
      timer = undefined;
    }
  }

  function getState(): OptimizationState {
    return state;
  }

  return { subscribe, getState, submit, poll, cancel, stopPolling };
}

function statusFromJob(status: JobStatus): OptimizationUiStatus {
  if (status === 'completed' || status === 'failed' || status === 'cancelled' || status === 'processing' || status === 'queued') {
    return status;
  }
  return 'queued';
}

function progressForStatus(status: JobStatus): number {
  if (status === 'queued') {
    return 0;
  }
  if (status === 'processing') {
    return 50;
  }
  return 100;
}

function timeoutMessage(message?: string): string {
  if (message?.toLowerCase().includes('timed out')) {
    return 'Optimization taking longer than expected. Please try again.';
  }
  return message ?? 'Optimization failed.';
}

function normalizeAppError(error: unknown): AppError {
  if (error && typeof error === 'object' && 'category' in error && 'code' in error && 'message' in error) {
    return error as AppError;
  }
  return {
    category: 'unknown',
    code: 'unknown_error',
    message: 'Something went wrong',
    retryable: false,
    cause: error
  };
}
