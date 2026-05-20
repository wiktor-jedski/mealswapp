import { describe, expect, it } from 'bun:test';
import { createOfflineController } from './offlineState';

describe('Offline controller', () => {
  it('tracks browser online and offline events', () => {
    const listeners = new Map<string, () => void>();
    const target = {
      onLine: true,
      addEventListener: (type: 'online' | 'offline', listener: () => void) => listeners.set(type, listener),
      removeEventListener: (type: 'online' | 'offline') => listeners.delete(type)
    };
    const controller = createOfflineController({ target, now: fixedNow });
    const detach = controller.attach();

    listeners.get('offline')?.();
    expect(controller.getState().isOnline).toBe(false);
    expect(controller.getState().status).toBe('offline');

    listeners.get('online')?.();
    expect(controller.getState().isOnline).toBe(true);

    detach();
    expect(listeners.size).toBe(0);
  });

  it('queues one eligible retry and runs it when connection returns', async () => {
    let retries = 0;
    const controller = createOfflineController({ target: { onLine: false }, now: fixedNow });

    controller.queueRetry(() => {
      retries += 1;
    });
    expect(controller.getState().queuedRetries).toBe(1);
    expect(controller.getState().message).toContain('retry when the connection returns');

    controller.setOnline(true);
    await Promise.resolve();
    await Promise.resolve();

    expect(retries).toBe(1);
    expect(controller.getState().status).toBe('online');
    expect(controller.getState().queuedRetries).toBe(0);
  });

  it('blocks offline mutations without dropping current state', () => {
    const controller = createOfflineController({ target: { onLine: false }, now: fixedNow });

    controller.blockMutation('Diet optimization');
    controller.blockMutation('Diet optimization');

    expect(controller.getState().blockedMutations).toEqual(['Diet optimization']);
    expect(controller.getState().message).toContain('current search state is preserved');
  });
});

function fixedNow(): Date {
  return new Date('2026-05-20T00:00:00.000Z');
}
