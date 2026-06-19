import { expect, test } from "bun:test";
import { get } from "svelte/store";
import { createOnlineStatus, type OnlineEventTarget } from "./online";

// Implements DESIGN-001 OfflineBanner connectivity listener verification.
test("tracks online events and removes listeners on teardown", () => {
  const listeners = new Map<string, () => void>();
  const removed: string[] = [];
  const target: OnlineEventTarget = { onLine: true, addEventListener: (type, listener) => listeners.set(type, listener), removeEventListener: (type) => removed.push(type) };
  const store = createOnlineStatus(target);
  const unsubscribe = store.subscribe(() => {});
  listeners.get("offline")?.();
  expect(get(store)).toBe(false);
  listeners.get("online")?.();
  expect(get(store)).toBe(true);
  unsubscribe();
  expect(removed.sort()).toEqual(["offline", "online"]);
});

// Implements DESIGN-001 OfflineBanner server-safe default verification.
test("defaults online when browser globals are absent", () => expect(get(createOnlineStatus())).toBe(true));

// Implements DESIGN-001 OfflineBanner browser-global adapter verification.
test("default adapter delegates to browser online events", () => {
  const originalWindow = globalThis.window;
  const originalNavigator = globalThis.navigator;
  const listeners = new Map<string, () => void>();
  Object.defineProperty(globalThis, "window", { configurable: true, value: { addEventListener: (type: string, listener: () => void) => listeners.set(type, listener), removeEventListener: (type: string) => listeners.delete(type) } });
  Object.defineProperty(globalThis, "navigator", { configurable: true, value: { onLine: false } });
  const store = createOnlineStatus();
  const values: boolean[] = [];
  const unsubscribe = store.subscribe((value) => values.push(value));
  listeners.get("online")?.();
  expect(values.at(-1)).toBe(true);
  unsubscribe();
  Object.defineProperty(globalThis, "window", { configurable: true, value: originalWindow });
  Object.defineProperty(globalThis, "navigator", { configurable: true, value: originalNavigator });
});
