import { fetchAutocomplete } from "../api/search-client";
import type { AutocompleteResponse, RankedAutocomplete } from "../api/generated";

// Implements DESIGN-001 AutocompleteDropdown 150ms debounce and server-ranked fetch orchestration.

/**
 * Debounce window applied to query changes before issuing an autocomplete request, matching
 * the SearchView `debounceSearchInput(value, delayMs: 150)` contract in DESIGN-001 step 4.
 */
export const AUTOCOMPLETE_DEBOUNCE_MS = 150;

/**
 * Fetch signature mirroring {@link fetchAutocomplete} so callers (and tests) can inject a stub
 * without binding to the global `fetch`.
 */
export type AutocompleteFetchFn = (query: string, signal: AbortSignal) => Promise<AutocompleteResponse>;

/** Opaque timer handle type shared by the injectable scheduler functions. */
export type TimerHandle = ReturnType<typeof setTimeout>;

/** Injectable `setTimeout` used for fake-timer tests. */
export type SetTimerFn = (handler: () => void, ms: number) => TimerHandle;

/** Injectable `clearTimeout` used for fake-timer tests. */
export type ClearTimerFn = (handle: TimerHandle) => void;

/**
 * Options for constructing an {@link AutocompleteController}.
 */
export interface AutocompleteControllerOptions {
	/** Debounce window in milliseconds; defaults to {@link AUTOCOMPLETE_DEBOUNCE_MS}. */
	delayMs?: number;
	/** Fetch implementation; defaults to {@link fetchAutocomplete}. */
	fetch?: AutocompleteFetchFn;
	/** `setTimeout` injector for fake-timer tests; defaults to the global. */
	setTimeout?: SetTimerFn;
	/** `clearTimeout` injector for fake-timer tests; defaults to the global. */
	clearTimeout?: ClearTimerFn;
	/** Called after a successful fetch with the server-ranked items in received order. */
	onResults?: (items: RankedAutocomplete[]) => void;
	/** Called when a non-abort fetch error occurs; aborts from superseded requests are suppressed. */
	onError?: (message: string) => void;
}

/**
 * Owns the 150ms debounce, in-flight cancellation, and server-ranked result commit for the
 * AutocompleteDropdown. Pure orchestration with no DOM dependency so the debounce timing can be
 * verified with injected fake timers.
 *
 * @remarks Implements DESIGN-001 AutocompleteDropdown debounce timing, selection, and dismissal rules.
 */
export class AutocompleteController {
	private readonly delayMs: number;
	private readonly fetchFn: AutocompleteFetchFn;
	private readonly setTimer: SetTimerFn;
	private readonly clearTimer: ClearTimerFn;
	private readonly onResults?: (items: RankedAutocomplete[]) => void;
	private readonly onError?: (message: string) => void;

	private timer: TimerHandle | undefined;
	private inflight: AbortController | undefined;
	private latestQuery = "";
	private disposed = false;

	constructor(options: AutocompleteControllerOptions = {}) {
		this.delayMs = options.delayMs ?? AUTOCOMPLETE_DEBOUNCE_MS;
		this.fetchFn = options.fetch ?? fetchAutocomplete;
		this.setTimer = options.setTimeout ?? ((handler, ms) => setTimeout(handler, ms));
		this.clearTimer = options.clearTimeout ?? ((handle) => clearTimeout(handle));
		this.onResults = options.onResults;
		this.onError = options.onError;
	}

	/**
	 * Schedules a debounced autocomplete fetch for the latest query. Rapid successive calls
	 * coalesce into a single request for the most recent query, clearing any pending timer.
	 */
	input(query: string): void {
		if (this.disposed) {
			return;
		}
		this.latestQuery = query;
		if (this.timer !== undefined) {
			this.clearTimer(this.timer);
			this.timer = undefined;
		}
		const queryForRequest = query;
		this.timer = this.setTimer(() => {
			void this.runFetch(queryForRequest);
		}, this.delayMs);
	}

	/**
	 * Cancels any pending debounce timer and aborts the in-flight request, if present. After
	 * `dispose`, further {@link input} calls are ignored.
	 */
	dispose(): void {
		this.disposed = true;
		this.cancel();
	}

	/** Cancels pending or in-flight autocomplete work while keeping the controller reusable. */
	cancel(): void {
		if (this.timer !== undefined) {
			this.clearTimer(this.timer);
			this.timer = undefined;
		}
		if (this.inflight) {
			this.inflight.abort(new DOMException("Autocomplete cancelled", "AbortError"));
			this.inflight = undefined;
		}
	}

	/** Returns the most recent query passed to {@link input}, primarily for diagnostics and tests. */
	get currentQuery(): string {
		return this.latestQuery;
	}

	private async runFetch(query: string): Promise<void> {
		this.timer = undefined;
		if (this.disposed) {
			return;
		}
		if (query.trim().length === 0) {
			// Empty query clears suggestions without a network round-trip.
			this.onResults?.([]);
			return;
		}
		// Abort any still-in-flight request superseded by this one.
		if (this.inflight) {
			this.inflight.abort(new DOMException("Superseded by newer query", "AbortError"));
		}
		const controller = new AbortController();
		this.inflight = controller;
		try {
			const response = await this.fetchFn(query, controller.signal);
			// Commit only if this request is still the latest one and the controller is alive.
			if (this.inflight === controller && !this.disposed) {
				this.onResults?.(response.items);
			}
		} catch (error) {
			if (this.inflight !== controller || this.disposed) {
				return;
			}
			if (error instanceof DOMException && error.name === "AbortError") {
				return;
			}
			this.onError?.(error instanceof Error ? error.message : "Autocomplete request failed");
		} finally {
			if (this.inflight === controller) {
				this.inflight = undefined;
			}
		}
	}
}
