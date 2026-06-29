import { afterEach, expect, test } from "bun:test";
import { get } from "svelte/store";
import { LocalQueryCache } from "../cache/local-query-cache";
import {
	cleanupOffline,
	createInitialOfflineStatus,
	initOffline,
	isStaleResult,
	offlineStatus,
	resetOfflineStatus,
	setShowingCached,
	setShowingStale
} from "./offline";

// Implements DESIGN-001 OfflineBanner online/offline event subscription and stale-indicator verification.
//
// NOTE: These tests verify Phase 05 online/offline event handling and cached/stale
// indicator state only. They do NOT claim Phase 09 service-worker API/image
// interception coverage, which remains Phase 09 scope per docs/implementation/04_OPEN.md
// ("Phase 09 remains responsible for service-worker API/image interception and broader offline hardening.").

const originalWindow = globalThis.window;
const originalNavigator = globalThis.navigator;

afterEach(() => {
	resetOfflineStatus();
	if (originalWindow === undefined) {
		delete (globalThis as { window?: unknown }).window;
	} else {
		Object.defineProperty(globalThis, "window", {
			configurable: true,
			value: originalWindow
		});
	}
	if (originalNavigator === undefined) {
		delete (globalThis as { navigator?: unknown }).navigator;
	} else {
		Object.defineProperty(globalThis, "navigator", {
			configurable: true,
			value: originalNavigator
		});
	}
});

class FakeWindow {
	addEventListenerCalls: Array<{ type: string; listener: EventListener }> = [];
	removeEventListenerCalls: Array<{ type: string; listener: EventListener }> = [];
	private onlineListeners = new Set<EventListener>();
	private offlineListeners = new Set<EventListener>();

	addEventListener(type: string, listener: EventListener): void {
		this.addEventListenerCalls.push({ type, listener });
		if (type === "online") {
			this.onlineListeners.add(listener);
		} else if (type === "offline") {
			this.offlineListeners.add(listener);
		}
	}

	removeEventListener(type: string, listener: EventListener): void {
		this.removeEventListenerCalls.push({ type, listener });
		if (type === "online") {
			this.onlineListeners.delete(listener);
		} else if (type === "offline") {
			this.offlineListeners.delete(listener);
		}
	}

	dispatchOnline(): void {
		const event = new Event("online");
		for (const listener of this.onlineListeners) {
			listener.call(this, event);
		}
	}

	dispatchOffline(): void {
		const event = new Event("offline");
		for (const listener of this.offlineListeners) {
			listener.call(this, event);
		}
	}
}

function setWindow(fake: FakeWindow): void {
	Object.defineProperty(globalThis, "window", {
		configurable: true,
		value: fake
	});
}

function setNavigator(onLine: boolean): void {
	Object.defineProperty(globalThis, "navigator", {
		configurable: true,
		value: { onLine }
	});
}

// Implements DESIGN-001 OfflineBanner default status verification.
test("createInitialOfflineStatus defaults to online with no cached/stale flags", () => {
	expect(createInitialOfflineStatus()).toEqual({
		online: true,
		showingCached: false,
		showingStale: false
	});
});

// Implements DESIGN-001 OfflineBanner initial store value verification.
test("offlineStatus starts online with no cached/stale flags", () => {
	expect(get(offlineStatus)).toEqual({
		online: true,
		showingCached: false,
		showingStale: false
	});
});

// Implements DESIGN-001 OfflineBanner online/offline event subscription verification.
test("initOffline subscribes to online and offline window events", () => {
	const fake = new FakeWindow();
	setWindow(fake);

	initOffline();

	expect(fake.addEventListenerCalls.map((c) => c.type)).toEqual(["online", "offline"]);
});

// Implements DESIGN-001 OfflineBanner offline event updates store verification.
test("dispatching offline event sets online to false", () => {
	const fake = new FakeWindow();
	setWindow(fake);

	initOffline();
	fake.dispatchOffline();

	expect(get(offlineStatus).online).toBe(false);
});

// Implements DESIGN-001 OfflineBanner cached-result label state verification.
test("setShowingCached flags cached results remain visible with offline label", () => {
	const fake = new FakeWindow();
	setWindow(fake);

	initOffline();
	fake.dispatchOffline();
	setShowingCached(true);

	const status = get(offlineStatus);
	expect(status.online).toBe(false);
	expect(status.showingCached).toBe(true);
});

// Implements DESIGN-001 OfflineBanner stale-result label state verification.
test("setShowingStale flags stale results remain visible with stale label", () => {
	const fake = new FakeWindow();
	setWindow(fake);

	initOffline();
	fake.dispatchOffline();
	setShowingStale(true);

	const status = get(offlineStatus);
	expect(status.online).toBe(false);
	expect(status.showingStale).toBe(true);
});

// Implements DESIGN-001 OfflineBanner uncached offline actionable feedback verification.
test("offline with no cached result produces actionable uncached state", () => {
	const fake = new FakeWindow();
	setWindow(fake);

	initOffline();
	fake.dispatchOffline();

	const status = get(offlineStatus);
	expect(status.online).toBe(false);
	expect(status.showingCached).toBe(false);
	expect(status.showingStale).toBe(false);
});

// Implements DESIGN-001 OfflineBanner reconnection permits fresh request verification.
test("dispatching online event after offline resets showing flags for a fresh request", () => {
	const fake = new FakeWindow();
	setWindow(fake);

	initOffline();
	fake.dispatchOffline();
	setShowingCached(true);
	setShowingStale(true);
	expect(get(offlineStatus).online).toBe(false);

	fake.dispatchOnline();

	const status = get(offlineStatus);
	expect(status.online).toBe(true);
	expect(status.showingCached).toBe(false);
	expect(status.showingStale).toBe(false);
});

// Implements DESIGN-001 OfflineBanner event listener teardown verification.
test("cleanupOffline removes the same online and offline handlers that were registered", () => {
	const fake = new FakeWindow();
	setWindow(fake);

	initOffline();

	const registeredOnline = fake.addEventListenerCalls.find((c) => c.type === "online")?.listener;
	const registeredOffline = fake.addEventListenerCalls.find((c) => c.type === "offline")?.listener;
	expect(registeredOnline).toBeDefined();
	expect(registeredOffline).toBeDefined();

	cleanupOffline();

	const removedOnline = fake.removeEventListenerCalls.find((c) => c.type === "online")?.listener;
	const removedOffline = fake.removeEventListenerCalls.find((c) => c.type === "offline")?.listener;
	expect(removedOnline).toBe(registeredOnline);
	expect(removedOffline).toBe(registeredOffline);
});

// Implements DESIGN-001 OfflineBanner event listener teardown idempotency verification.
test("cleanupOffline is safe to call multiple times", () => {
	const fake = new FakeWindow();
	setWindow(fake);

	initOffline();
	cleanupOffline();
	fake.removeEventListenerCalls.length = 0;

	expect(() => cleanupOffline()).not.toThrow();
	expect(fake.removeEventListenerCalls).toHaveLength(0);
});

// Implements DESIGN-001 OfflineBanner SSR initialization safety verification.
test("initOffline returns a no-op cleanup when window is undefined", () => {
	delete (globalThis as { window?: unknown }).window;

	const cleanup = initOffline();
	expect(typeof cleanup).toBe("function");
	expect(() => cleanup()).not.toThrow();
	expect(get(offlineStatus).online).toBe(true);
});

// Implements DESIGN-001 OfflineBanner SSR teardown safety verification.
test("cleanupOffline is safe when window is undefined", () => {
	delete (globalThis as { window?: unknown }).window;

	expect(() => cleanupOffline()).not.toThrow();
});

// Implements DESIGN-001 OfflineBanner navigator.onLine initial sync verification.
test("initOffline syncs online false from navigator.onLine when starting offline", () => {
	const fake = new FakeWindow();
	setWindow(fake);
	setNavigator(false);

	initOffline();

	expect(get(offlineStatus).online).toBe(false);
});

// Implements DESIGN-001 OfflineBanner navigator.onLine online sync verification.
test("initOffline syncs online true from navigator.onLine when starting online", () => {
	const fake = new FakeWindow();
	setWindow(fake);
	setNavigator(true);

	initOffline();

	expect(get(offlineStatus).online).toBe(true);
});

// Implements DESIGN-001 LocalStorageManager stale state delegation verification.
test("isStaleResult delegates to the LocalQueryCache stale check", () => {
	const cache = new LocalQueryCache({ storage: null, now: () => 1000 });
	const key = "request-key";
	const staleMs = 100;

	expect(isStaleResult(key, cache, staleMs)).toBe(true);
});
