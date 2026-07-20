import { writable } from "svelte/store";

// Implements DESIGN-001 SearchView authoritative Daily Diet selection across search modes.

/** One memory-only selected Daily Diet identity shared by collection, search, and optimization. */
export const selectedDailyDietId = writable<string | null>(null);
