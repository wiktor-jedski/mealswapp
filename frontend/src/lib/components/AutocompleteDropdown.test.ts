import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";

import type { AutocompleteResponse, RankedAutocomplete } from "../api/generated";
import {
	AUTOCOMPLETE_DEBOUNCE_MS,
	AutocompleteController,
	type ClearTimerFn,
	type SetTimerFn,
	type TimerHandle
} from "./autocomplete-controller";

// Implements DESIGN-001 AutocompleteDropdown debounce timing and component structure verification.
//
// Bun's test runner does not provide Jest-style `useFakeTimers` for `setTimeout`, so the debounce
// timing is verified through an injectable fake scheduler: the controller accepts `setTimeout` and
// `clearTimeout` functions, and a `FakeClock` records scheduled delays and fires the pending timer
// only on demand. This deterministically proves no request fires before 150ms and exactly one
// request fires for the final keystroke. Component structure (ARIA combobox/listbox, keyboard
// handlers, ranked order, document-flow container, traceability) is verified via static-source
// assertions on the Svelte source, mirroring the static component test approach.
// `vite build` compiles the component, validating the Svelte source at build time.

const source = readFileSync(join(import.meta.dir, "AutocompleteDropdown.svelte"), "utf8");

/**
 * Injectable scheduler that records scheduled delays and cleared handles and fires the pending
 * timer only when {@link FakeClock.fire} is invoked, simulating advancing a fake clock past the
 * debounce window.
 */
class FakeClock {
	private handles = new Map<number, () => void>();
	private nextId = 1;
	private activeId: number | undefined;
	readonly scheduledDelays: number[] = [];
	readonly clearedIds: number[] = [];

	setTimeout: SetTimerFn = (handler, ms) => {
		const id = this.nextId++;
		this.handles.set(id, handler);
		this.scheduledDelays.push(ms);
		this.activeId = id;
		return id as unknown as TimerHandle;
	};

	clearTimeout: ClearTimerFn = (handle) => {
		const id = handle as unknown as number;
		if (this.handles.delete(id)) {
			this.clearedIds.push(id);
		}
		if (this.activeId === id) {
			this.activeId = undefined;
		}
	};

	get pendingCount(): number {
		return this.handles.size;
	}

	get lastDelay(): number {
		return this.scheduledDelays[this.scheduledDelays.length - 1] ?? -1;
	}

	/** Fires the most recently scheduled pending timer and awaits its async result. */
	async fire(): Promise<void> {
		const id = this.activeId;
		if (id === undefined) {
			return;
		}
		const handler = this.handles.get(id);
		this.handles.delete(id);
		this.activeId = undefined;
		await handler?.();
	}
}

function makeItem(itemId: string, label: string, rank: number): RankedAutocomplete {
	return { itemId, label, exactMatch: rank === 1, levenshteinDistance: rank - 1, length: label.length, rank };
}

function makeFetchRecorder(response: AutocompleteResponse): {
	fetch: (query: string, _signal: AbortSignal) => Promise<AutocompleteResponse>;
	calls: Array<{ query: string }>;
} {
	const calls: Array<{ query: string }> = [];
	const fetch = (query: string, _signal: AbortSignal): Promise<AutocompleteResponse> => {
		calls.push({ query });
		return Promise.resolve(response);
	};
	return { fetch, calls };
}

// Implements DESIGN-001 AutocompleteDropdown 150ms debounce window verification.
test("no autocomplete request fires before the 150ms debounce window elapses", async () => {
	const clock = new FakeClock();
	const { fetch, calls } = makeFetchRecorder({ items: [] });
	const controller = new AutocompleteController({
		setTimeout: clock.setTimeout,
		clearTimeout: clock.clearTimeout,
		fetch
	});

	controller.input("ap");

	expect(calls.length).toBe(0);
	expect(clock.pendingCount).toBe(1);
	expect(clock.lastDelay).toBe(AUTOCOMPLETE_DEBOUNCE_MS);
	expect(clock.lastDelay).toBe(150);

	// Before the timer fires (i.e. before 150ms elapses), no request has been issued.
	expect(calls.length).toBe(0);

	controller.dispose();
	expect(clock.pendingCount).toBe(0);
});

// Implements DESIGN-001 AutocompleteDropdown final-keystroke coalescing verification.
test("rapid successive keystrokes produce exactly one request after the final keystroke", async () => {
	const clock = new FakeClock();
	const { fetch, calls } = makeFetchRecorder({ items: [] });
	const controller = new AutocompleteController({
		setTimeout: clock.setTimeout,
		clearTimeout: clock.clearTimeout,
		fetch
	});

	controller.input("a");
	controller.input("ap");
	controller.input("app");
	controller.input("appl");

	// All but the latest timer are cleared; only one pending debounce remains.
	expect(calls.length).toBe(0);
	expect(clock.pendingCount).toBe(1);
	expect(clock.clearedIds.length).toBe(3);

	// Advance the fake clock past 150ms: the single pending timer fires once.
	await clock.fire();

	expect(calls.length).toBe(1);
	expect(calls[0]?.query).toBe("appl");

	controller.dispose();
});

// Implements DESIGN-001 AutocompleteDropdown server-ranked display order preservation verification.
test("onResults receives items in server rank order without client re-sorting", async () => {
	const clock = new FakeClock();
	const ranked: RankedAutocomplete[] = [
		makeItem("apple", "Apple", 1),
		makeItem("applesauce", "Applesauce", 2),
		makeItem("snapple", "Snapple", 3)
	];
	const { fetch } = makeFetchRecorder({ items: ranked });
	const received: RankedAutocomplete[][] = [];
	const controller = new AutocompleteController({
		setTimeout: clock.setTimeout,
		clearTimeout: clock.clearTimeout,
		fetch,
		onResults: (items) => received.push(items)
	});

	controller.input("app");
	await clock.fire();

	expect(received.length).toBe(1);
	expect(received[0]?.map((item) => item.itemId)).toEqual(["apple", "applesauce", "snapple"]);
	expect(received[0]?.map((item) => item.rank)).toEqual([1, 2, 3]);

	controller.dispose();
});

// Implements DESIGN-001 AutocompleteDropdown empty-query suppression verification.
test("an empty query clears suggestions without issuing a network request", async () => {
	const clock = new FakeClock();
	const { fetch, calls } = makeFetchRecorder({ items: [makeItem("apple", "Apple", 1)] });
	const received: RankedAutocomplete[][] = [];
	const controller = new AutocompleteController({
		setTimeout: clock.setTimeout,
		clearTimeout: clock.clearTimeout,
		fetch,
		onResults: (items) => received.push(items)
	});

	controller.input("app");
	await clock.fire();
	expect(calls.length).toBe(1);

	controller.input("");
	await clock.fire();

	expect(calls.length).toBe(1);
	expect(received[1]).toEqual([]);

	controller.dispose();
});

// Implements DESIGN-001 AutocompleteDropdown dismissal via dispose verification.
test("dispose cancels the pending debounce timer so no request fires", async () => {
	const clock = new FakeClock();
	const { fetch, calls } = makeFetchRecorder({ items: [] });
	const controller = new AutocompleteController({
		setTimeout: clock.setTimeout,
		clearTimeout: clock.clearTimeout,
		fetch
	});

	controller.input("app");
	expect(clock.pendingCount).toBe(1);

	controller.dispose();
	expect(clock.pendingCount).toBe(0);

	await clock.fire();
	expect(calls.length).toBe(0);
});

// Implements DESIGN-001 AutocompleteDropdown reusable cancellation verification.
test("cancel clears pending autocomplete work without disposing the controller", async () => {
	const clock = new FakeClock();
	const { fetch, calls } = makeFetchRecorder({ items: [] });
	const controller = new AutocompleteController({
		setTimeout: clock.setTimeout,
		clearTimeout: clock.clearTimeout,
		fetch
	});

	controller.input("app");
	expect(clock.pendingCount).toBe(1);

	controller.cancel();
	expect(clock.pendingCount).toBe(0);

	controller.input("apple");
	await clock.fire();
	expect(calls.length).toBe(1);
	expect(calls[0]?.query).toBe("apple");

	controller.dispose();
});

// Implements DESIGN-001 AutocompleteDropdown reusable cancellation abort verification.
test("cancel aborts an in-flight autocomplete request", async () => {
	const clock = new FakeClock();
	let resolveFirst!: (value: AutocompleteResponse) => void;
	const firstPromise = new Promise<AutocompleteResponse>((resolve) => {
		resolveFirst = resolve;
	});
	const calls: Array<{ query: string; signal: AbortSignal }> = [];
	const fetch = (query: string, signal: AbortSignal): Promise<AutocompleteResponse> => {
		calls.push({ query, signal });
		return firstPromise;
	};
	const received: RankedAutocomplete[][] = [];
	const controller = new AutocompleteController({
		setTimeout: clock.setTimeout,
		clearTimeout: clock.clearTimeout,
		fetch,
		onResults: (items) => received.push(items)
	});

	controller.input("app");
	await clock.fire();
	expect(calls.length).toBe(1);

	controller.cancel();
	expect(calls[0]?.signal.aborted).toBe(true);

	resolveFirst({ items: [makeItem("apple", "Apple", 1)] });
	await Promise.resolve();
	expect(received.length).toBe(0);

	controller.dispose();
});

// Implements DESIGN-001 AutocompleteDropdown in-flight supersession verification.
test("a new keystroke aborts an in-flight request and commits only the latest results", async () => {
	const clock = new FakeClock();
	let resolveFirst!: (value: AutocompleteResponse) => void;
	const firstPromise = new Promise<AutocompleteResponse>((resolve) => {
		resolveFirst = resolve;
	});
	const calls: Array<{ query: string; signal: AbortSignal }> = [];
	const fetch = (query: string, signal: AbortSignal): Promise<AutocompleteResponse> => {
		calls.push({ query, signal });
		if (calls.length === 1) {
			return firstPromise;
		}
		return Promise.resolve({ items: [makeItem("appl", "Apple", 1)] });
	};
	const received: RankedAutocomplete[][] = [];
	const controller = new AutocompleteController({
		setTimeout: clock.setTimeout,
		clearTimeout: clock.clearTimeout,
		fetch,
		onResults: (items) => received.push(items)
	});

	controller.input("ap");
	await clock.fire();
	expect(calls.length).toBe(1);
	expect(calls[0]?.signal.aborted).toBe(false);

	controller.input("appl");
	await clock.fire();
	expect(calls.length).toBe(2);
	// The first request was aborted when the second one superseded it.
	expect(calls[0]?.signal.aborted).toBe(true);
	expect(calls[1]?.signal.aborted).toBe(false);

	// Resolving the superseded first request must not commit stale results.
	resolveFirst({ items: [makeItem("ap", "Apricot", 1)] });
	await Promise.resolve();

	expect(received.length).toBe(1);
	expect(received[0]?.[0]?.itemId).toBe("appl");

	controller.dispose();
});

// Implements DESIGN-001 AutocompleteDropdown dispose-while-in-flight abort verification.
test("dispose aborts an in-flight request so its results never commit", async () => {
	const clock = new FakeClock();
	let resolveFirst!: (value: AutocompleteResponse) => void;
	const firstPromise = new Promise<AutocompleteResponse>((resolve) => {
		resolveFirst = resolve;
	});
	const calls: Array<{ query: string; signal: AbortSignal }> = [];
	const fetch = (query: string, signal: AbortSignal): Promise<AutocompleteResponse> => {
		calls.push({ query, signal });
		return firstPromise;
	};
	const received: RankedAutocomplete[][] = [];
	const controller = new AutocompleteController({
		setTimeout: clock.setTimeout,
		clearTimeout: clock.clearTimeout,
		fetch,
		onResults: (items) => received.push(items)
	});

	controller.input("ap");
	await clock.fire();
	expect(calls.length).toBe(1);
	expect(calls[0]?.signal.aborted).toBe(false);

	controller.dispose();
	// Disposing while the request is in-flight aborts the in-flight AbortController.
	expect(calls[0]?.signal.aborted).toBe(true);

	// Resolving the aborted request must not commit results to the disposed controller.
	resolveFirst({ items: [makeItem("ap", "Apricot", 1)] });
	await Promise.resolve();

	expect(received.length).toBe(0);
});

// Implements DESIGN-001 AutocompleteController currentQuery diagnostic getter verification.
test("currentQuery returns the most recent query passed to input", () => {
	const clock = new FakeClock();
	const { fetch } = makeFetchRecorder({ items: [] });
	const controller = new AutocompleteController({
		setTimeout: clock.setTimeout,
		clearTimeout: clock.clearTimeout,
		fetch
	});

	expect(controller.currentQuery).toBe("");

	controller.input("app");
	expect(controller.currentQuery).toBe("app");

	controller.input("apple");
	expect(controller.currentQuery).toBe("apple");

	controller.dispose();
});

// Implements DESIGN-001 AutocompleteController default scheduler fallback verification.
test("default setTimeout and clearTimeout are used when not injected", async () => {
	const received: RankedAutocomplete[][] = [];
	const { fetch, calls } = makeFetchRecorder({ items: [] });
	const controller = new AutocompleteController({
		delayMs: 50,
		fetch,
		onResults: (items) => received.push(items)
	});

	// Scheduling uses the default setTimeout arrow; disposing before the timer fires
	// exercises the default clearTimeout arrow.
	controller.input("");
	controller.dispose();

	// Wait past the 50ms debounce window to confirm the cleared timer never fires.
	await new Promise((resolve) => setTimeout(resolve, 60));

	expect(calls.length).toBe(0);
	expect(received).toEqual([]);
});

// Implements DESIGN-001 AutocompleteDropdown traceability comment verification.
test("component cites the DESIGN source", () => {
	expect(source).toContain("<!-- Implements DESIGN-001 AutocompleteDropdown -->");
});

// Implements DESIGN-001 AutocompleteDropdown ARIA combobox/listbox state verification.
test("declares combobox input, listbox container, option roles, aria-expanded, and aria-selected", () => {
	expect(source).toContain('role="combobox"');
	expect(source).toContain("aria-expanded={isOpen}");
	expect(source).toContain('aria-controls={listboxId}');
	expect(source).toContain('role="listbox"');
	expect(source).toContain('role="option"');
	expect(source).toContain("aria-selected={index === activeIndex}");
	expect(source).toContain("aria-activedescendant");
});

// Implements DESIGN-001 AutocompleteDropdown typed-query submit verification.
test("declares typed-query submission without requiring an active suggestion", () => {
	expect(source).toContain("onSubmit?: (query: string) => void");
	expect(source).toContain("onSubmit(query)");
	expect(source).toContain("query.trim().length > 0");
	expect(source).toContain("activeIndex = -1");
});

// Implements DESIGN-001 SearchView mode-specific search guidance verification.
test("declares a placeholder prop for mode-specific search guidance", () => {
	expect(source).toContain('placeholder = "Search foods, meals, or ingredients…"');
	expect(source).toContain("{placeholder}");
	expect(source).toContain("class=\"truncate rounded");
});

// Implements DESIGN-001 SearchView initial and mode-change search focus verification.
test("declares a focus key that focuses the combobox on initial load and mode changes", () => {
	expect(source).toContain("focusKey?: string | number");
	expect(source).toContain("void focusSearchInput(focusKey, inputEl)");
	expect(source).toContain("await tick()");
	expect(source).toContain("element.focus()");
});

// Implements DESIGN-001 SearchView submitted-search spinner verification.
test("declares an inline spinner for submitted result-search loading", () => {
	expect(source).toContain("searching = false");
	expect(source).toContain("{#if searching}");
	expect(source).toContain("data-search-spinner");
	expect(source).toContain('aria-label="Searching"');
	expect(source).toContain("motion-safe:animate-spin");
	expect(source).toContain("pr-10");
});

// Implements DESIGN-001 AutocompleteDropdown server-ranked display order in markup verification.
test("renders items via #each in received order without client-side sorting", () => {
	expect(source).toContain("{#each items as item, index (item.itemId)}");
	expect(source).toContain("{item.label}");
	expect(source).not.toContain(".sort(");
	expect(source).not.toContain(".toSorted(");
});

// Implements DESIGN-001 AutocompleteDropdown floating overlay verification.
test("listbox floats over page content without pushing the results layout down", () => {
	expect(source).toContain('class="relative grid gap-1"');
	expect(source).toContain("absolute left-0 top-full");
	expect(source).toContain("z-20");
	expect(source).toContain("w-full");
	expect(source).toContain("shadow-lg");
	expect(source).not.toContain("float-right");
	expect(source).not.toContain("float-left");
});

// Implements DESIGN-001 AutocompleteDropdown keyboard navigation handlers verification.
test("Tab, ArrowUp, ArrowDown, Enter, and Escape keydown handlers move, select, and dismiss", () => {
	expect(source).toContain("onkeydown={onInputKeydown}");
	expect(source).toContain('case "Tab"');
	expect(source).toContain('case "ArrowDown"');
	expect(source).toContain('case "ArrowUp"');
	expect(source).toContain("event.shiftKey ? -1 : 1");
	expect(source).toContain('case "Enter"');
	expect(source).toContain("selectActive()");
	expect(source).toContain('case "Escape"');
	expect(source).toContain("dismiss()");
	expect(source).toContain("moveActive");
	expect(source).toContain("moveActive(1, false)");
	expect(source).toContain("moveActive(-1, false)");
	expect(source).toContain("option?.focus()");
});

// Implements DESIGN-001 AutocompleteDropdown selection callback wiring verification.
test("Enter and option click both invoke the onSelect prop with the active item", () => {
	expect(source).toContain("onSelect: (item: RankedAutocomplete) => void");
	expect(source).toContain("suppressedSelectedQuery = item.label");
	expect(source).toContain("controller.cancel()");
	expect(source).toContain("onSelect(item)");
	expect(source).toContain("onOptionClick");
});

// Implements DESIGN-001 AutocompleteDropdown 150ms debounce wiring in the component verification.
test("component uses the AutocompleteController with the 150ms debounce constant", () => {
	expect(source).toContain("import { AutocompleteController, AUTOCOMPLETE_DEBOUNCE_MS }");
	expect(source).toContain("delayMs: AUTOCOMPLETE_DEBOUNCE_MS");
	expect(source).toContain("controller.input(query)");
});
