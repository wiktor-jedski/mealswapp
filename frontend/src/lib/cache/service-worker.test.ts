import { afterEach, expect, test } from "bun:test";
import { registerServiceWorker } from "./service-worker";

const originalNavigator = globalThis.navigator;

afterEach(() => {
  Object.defineProperty(globalThis, "navigator", {
    configurable: true,
    value: originalNavigator
  });
});

// Implements DESIGN-011 ServiceWorkerCache disabled registration verification.
test("registerServiceWorker is disabled by default for bootstrap safety", async () => {
  await expect(registerServiceWorker({ enabled: false })).resolves.toBeNull();
});

// Implements DESIGN-011 ServiceWorkerCache unsupported browser verification.
test("registerServiceWorker returns null when browser support is missing", async () => {
  Object.defineProperty(globalThis, "navigator", {
    configurable: true,
    value: {}
  });

  await expect(registerServiceWorker({ enabled: true })).resolves.toBeNull();
});

// Implements DESIGN-011 ServiceWorkerCache registration verification.
test("registerServiceWorker delegates to navigator service worker", async () => {
  const registration = { scope: "/" } as ServiceWorkerRegistration;
  Object.defineProperty(globalThis, "navigator", {
    configurable: true,
    value: {
      serviceWorker: {
        register: async (path: string) => {
          expect(path).toBe("/service-worker.js");
          return registration;
        }
      }
    }
  });

  await expect(registerServiceWorker({ enabled: true })).resolves.toBe(registration);
});
