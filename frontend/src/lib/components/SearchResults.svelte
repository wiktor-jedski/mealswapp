<script lang="ts">
  import { get } from "svelte/store";
  import { createQuery } from "@tanstack/svelte-query";
  import {
    createSearchQueryOptions,
    SearchClientError,
    LOCAL_CACHE_STALE_MS
  } from "../api/search-client";
  import { createLocalQueryCache } from "../cache/local-query-cache";
  import { searchStore, setPage, searchRequestKey } from "../stores/search";
  import { offlineStatus, setShowingCached } from "../stores/offline";
  import ResultsGrid from "./ResultsGrid.svelte";
  import type { SearchRejection } from "../api/generated";

  // Implements DESIGN-001 SearchView results composition: TanStack Query over generated envelopes, ResultsGrid wiring, and Daily Diet rejection lift.

  /**
   * Optional callback lifting a structured {@link SearchRejection} (derived from a 422
   * `SearchClientError`) to the shell so `DailyDietControls` can render rejection detail.
   */
  let { onRejection = () => {} }: { onRejection?: (r: SearchRejection | null) => void } = $props();

  /** Local query cache used by the search client for offline reuse and LRU persistence. */
  const localCache = createLocalQueryCache();

  /** Derived TanStack Query options store bridging the search store to `createQuery`. */
  const optionsStore = createSearchQueryOptions(searchStore, localCache);

  // Bridges the derived options store to a rune so createQuery re-evaluates on search-store changes.
  let currentOptions = $state(get(optionsStore));
  $effect(() => optionsStore.subscribe((o) => { currentOptions = o; }));

  // Bridges the search store to a rune for template reads (page) and cache-key derivation.
  let state = $state(get(searchStore));
  $effect(() => searchStore.subscribe((s) => { state = s; }));

  // Bridges the online flag to a rune, only updating on change to avoid write-back loops.
  let online = $state(get(offlineStatus).online);
  $effect(() =>
    offlineStatus.subscribe((o) => {
      if (o.online !== online) online = o.online;
    })
  );

  /** TanStack Query result driving the ResultsGrid; reactive via the options rune. */
  const query = createQuery(() => currentOptions);

  /**
   * Structured rejection derived from a 422 `SearchClientError`. The search client maps the
   * envelope `error` to an `AppError` but does not expose `data.rejection.field`, so `field`
   * is omitted here (follow-up: extend search-client to surface the structured rejection).
   */
  let rejection = $derived.by<SearchRejection | null>(() => {
    const err = query.error;
    if (err instanceof SearchClientError && err.status === 422) {
      return { code: err.appError.code, message: err.appError.message };
    }
    return null;
  });

  // Lifts rejection to the parent shell so DailyDietControls renders structured rejection detail.
  $effect(() => {
    onRejection(rejection);
  });

  // Reflects offline cached-result state into the OfflineBanner indicator without re-subscribing to the writable we mutate.
  $effect(() => {
    const data = query.data;
    if (!online && data) {
      const key = searchRequestKey(state);
      setShowingCached(localCache.has(key) && !localCache.isStale(key, LOCAL_CACHE_STALE_MS));
    }
  });

  /** User-facing error message for the ResultsGrid error state; null while the query is healthy. */
  let errorMessage = $derived(
    query.error
      ? query.error instanceof SearchClientError
        ? query.error.appError.message
        : "Search failed."
      : null
  );
</script>

<!-- Implements DESIGN-001 SearchView ResultsGrid wiring from TanStack Query result. -->
<ResultsGrid
  results={query.data?.items ?? []}
  similarityMetadata={query.data?.similarityMetadata ?? []}
  similarityScores={query.data?.similarityScores ?? []}
  loading={query.isFetching}
  error={errorMessage}
  totalCount={query.data?.totalCount ?? 0}
  page={state.page}
  onPageChange={setPage}
/>
