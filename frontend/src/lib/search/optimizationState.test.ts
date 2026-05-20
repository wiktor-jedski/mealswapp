import { describe, expect, it } from 'bun:test';
import type { DietOptimizationRequest, OptimizationJob, OptimizationSubmitResponse } from '../api/types';
import { createOptimizationController } from './optimizationState';

describe('Optimization controller', () => {
  it('handles 202 submission and starts polling queued jobs', async () => {
    const timers: Array<() => void> = [];
    const api = fakeOptimizationApi({
      submit: { jobId: 'job-1', pollUrl: '/api/v1/optimization/jobs/job-1', status: 'queued' },
      jobs: []
    });
    const controller = createOptimizationController({
      api,
      setIntervalFn: (callback) => {
        timers.push(callback);
        return callback;
      },
      clearIntervalFn: () => undefined
    });

    await controller.submit(validRequest());

    expect(api.submitted?.targetMacros.protein).toBe(90);
    expect(controller.getState().status).toBe('queued');
    expect(controller.getState().jobId).toBe('job-1');
    expect(timers.length).toBe(1);
  });

  it('updates queued and processing states while polling', async () => {
    const api = fakeOptimizationApi({
      submit: { jobId: 'job-1', pollUrl: '/api/v1/optimization/jobs/job-1', status: 'queued' },
      jobs: [job('job-1', 'processing', { progress: 60 })]
    });
    const controller = createOptimizationController({ api, setIntervalFn: noopInterval, clearIntervalFn: noopClear });

    await controller.submit(validRequest());
    await controller.poll();

    expect(controller.getState().status).toBe('processing');
    expect(controller.getState().progress).toBe(60);
  });

  it('stops polling and exposes completed alternatives', async () => {
    let cleared = false;
    const api = fakeOptimizationApi({
      submit: { jobId: 'job-1', pollUrl: '/api/v1/optimization/jobs/job-1', status: 'queued' },
      jobs: [job('job-1', 'completed', {
        progress: 100,
        result: [{ meals: [{ itemId: 'tofu', quantity: 200 }], macros: { protein: 92, carbs: 155, fat: 50 }, calories: 640, similarityScore: 0.68 }]
      })]
    });
    const controller = createOptimizationController({
      api,
      setIntervalFn: noopInterval,
      clearIntervalFn: () => {
        cleared = true;
      }
    });

    await controller.submit(validRequest());
    await controller.poll();

    expect(controller.getState().status).toBe('completed');
    expect(controller.getState().alternatives[0].meals[0].itemId).toBe('tofu');
    expect(cleared).toBe(true);
  });

  it('maps failed timeout jobs and preserves partial alternatives', async () => {
    const api = fakeOptimizationApi({
      submit: { jobId: 'job-1', pollUrl: '/api/v1/optimization/jobs/job-1', status: 'queued' },
      jobs: [job('job-1', 'failed', {
        error: 'Optimization timed out',
        result: [{ meals: [{ itemId: 'lentils', quantity: 150 }], macros: { protein: 80, carbs: 150, fat: 48 }, calories: 590, similarityScore: 0.5 }]
      })]
    });
    const controller = createOptimizationController({ api, setIntervalFn: noopInterval, clearIntervalFn: noopClear });

    await controller.submit(validRequest());
    await controller.poll();

    expect(controller.getState().status).toBe('failed');
    expect(controller.getState().error?.category).toBe('timeout');
    expect(controller.getState().message).toBe('Optimization taking longer than expected. Please try again.');
    expect(controller.getState().partialAlternatives[0].meals[0].itemId).toBe('lentils');
  });

  it('supports local cancellation while a job is in progress', async () => {
    let cleared = false;
    const controller = createOptimizationController({
      api: fakeOptimizationApi({
        submit: { jobId: 'job-1', pollUrl: '/api/v1/optimization/jobs/job-1', status: 'queued' },
        jobs: []
      }),
      setIntervalFn: noopInterval,
      clearIntervalFn: () => {
        cleared = true;
      }
    });

    await controller.submit(validRequest());
    controller.cancel();

    expect(controller.getState().status).toBe('cancelled');
    expect(controller.getState().message).toBe('Optimization cancelled.');
    expect(cleared).toBe(true);
  });
});

function validRequest(): DietOptimizationRequest {
  return {
    originalMeals: [{ id: 'meal-1', name: 'Oats', quantity: 100 }],
    targetMacros: { protein: 90, carbs: 160, fat: 55 },
    excludedIds: [],
    tolerancePercent: 10
  };
}

function job(jobId: string, status: OptimizationJob['status'], overrides: Partial<OptimizationJob> = {}): OptimizationJob {
  return {
    jobId,
    userId: 'user-1',
    request: validRequest(),
    status,
    createdAt: '2026-05-20T12:00:00Z',
    ...overrides
  };
}

function fakeOptimizationApi(options: { submit: OptimizationSubmitResponse; jobs: OptimizationJob[] }) {
  const api = {
    submitted: undefined as DietOptimizationRequest | undefined,
    async submitOptimizationJob(request: DietOptimizationRequest): Promise<OptimizationSubmitResponse> {
      api.submitted = request;
      return options.submit;
    },
    async getOptimizationJob(): Promise<OptimizationJob> {
      const next = options.jobs.shift();
      if (!next) {
        throw new Error('missing job fixture');
      }
      return next;
    }
  };
  return api;
}

function noopInterval(): unknown {
  return {};
}

function noopClear(): void {
  return undefined;
}
