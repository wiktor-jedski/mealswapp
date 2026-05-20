import { describe, expect, it } from 'bun:test';
import { ApiClientError } from '../api/client';
import { createRetryManager, RetryCancelledError } from './retryManager';

describe('RetryManager', () => {
  it('retries transient failures with exponential backoff and then succeeds', async () => {
    const timers: Array<() => void> = [];
    const delays: number[] = [];
    let calls = 0;
    const manager = createRetryManager({
      policy: { maxAttempts: 3, baseDelayMs: 100, maxDelayMs: 1000, jitterMs: 0, retryableCategories: ['network', 'timeout', 'server', 'dependency'] },
      setTimeoutFn: (callback, delay) => {
        timers.push(callback);
        delays.push(delay);
        return callback;
      },
      clearTimeoutFn: () => undefined
    });

    const result = manager.run(async () => {
      calls += 1;
      if (calls < 3) {
        throw new ApiClientError({ category: 'network', code: 'network_error', message: 'Network failed', retryable: true });
      }
      return 'ok';
    });

    await flushUntil(() => timers.length > 0);
    timers.shift()?.();
    await flushUntil(() => timers.length > 0);
    timers.shift()?.();

    expect(await result).toBe('ok');
    expect(calls).toBe(3);
    expect(delays).toEqual([100, 200]);
    expect(manager.getState().status).toBe('succeeded');
  });

  it('does not retry validation, auth, entitlement, or business-rule failures', async () => {
    const categories = ['validation', 'auth', 'entitlement'] as const;

    for (const category of categories) {
      let calls = 0;
      const manager = createRetryManager({
        setTimeoutFn: () => {
          throw new Error('timer should not be scheduled');
        }
      });

      await expect(
        manager.run(async () => {
          calls += 1;
          throw new ApiClientError({ category, code: `${category}_error`, message: 'Blocked', retryable: false });
        })
      ).rejects.toThrow('Blocked');
      expect(calls).toBe(1);
      expect(manager.getState().status).toBe('failed');
    }
  });

  it('cancels a pending retry timer', async () => {
    const timers: Array<() => void> = [];
    let cleared = 0;
    const manager = createRetryManager({
      policy: { maxAttempts: 3, baseDelayMs: 100, maxDelayMs: 1000, jitterMs: 0, retryableCategories: ['network'] },
      setTimeoutFn: (callback) => {
        timers.push(callback);
        return callback;
      },
      clearTimeoutFn: () => {
        cleared += 1;
      }
    });

    const result = manager.run(async () => {
      throw new ApiClientError({ category: 'network', code: 'network_error', message: 'Network failed', retryable: true });
    });
    await flushUntil(() => timers.length > 0);

    manager.cancel();
    timers.shift()?.();

    await expect(result).rejects.toBeInstanceOf(RetryCancelledError);
    expect(cleared).toBe(1);
    expect(manager.getState().status).toBe('cancelled');
  });
});

async function flushUntil(predicate: () => boolean): Promise<void> {
  for (let index = 0; index < 10 && !predicate(); index += 1) {
    await Promise.resolve();
  }
}
