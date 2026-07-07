<script lang="ts">
  import { derived } from "svelte/store";
  import { createQuery } from "@tanstack/svelte-query";
  import type { CreateQueryOptions } from "@tanstack/svelte-query";
  import {
    createSearchQueryOptions,
    SearchClientError,
    LOCAL_CACHE_STALE_MS,
    type SearchQueryKey
  } from "../api/search-client";
  import { createLocalQueryCache } from "../cache/local-query-cache";
  import { addSubstitutionInput, searchStore, setPage, searchRequestKey } from "../stores/search";
  import { offlineStatus, setShowingCached, setShowingStale } from "../stores/offline";
  import { preferencesStore } from "../stores/preferences";
  import { displayUnitForBasis } from "../units";
  import ResultsGrid from "./ResultsGrid.svelte";
  import type { FoodObject, SearchRejection, SearchResponse } from "../api/generated";

  // Implements DESIGN-001 SearchView results composition: TanStack Query over generated envelopes, ResultsGrid wiring, and Daily Diet rejection lift.

  /**
   * Optional callback lifting a structured {@link SearchRejection} (derived from a 422
   * `SearchClientError`) to the shell so `DailyDietControls` can render rejection detail.
   */
  let {
    searchEnabled = true,
    onRejection = () => {},
    onSearchInFlightChange = () => {}
  }: {
    searchEnabled?: boolean;
    onRejection?: (r: SearchRejection | null) => void;
    onSearchInFlightChange?: (searching: boolean) => void;
  } = $props();

  /** Local query cache used by the search client for offline reuse and LRU persistence. */
  const localCache = createLocalQueryCache();

  // Implements DESIGN-001 SearchView committed server-side search execution.
  const committedSearchStore = derived(searchStore, (storeState) => ({
    ...storeState,
    query: storeState.submittedQuery
  }));

  /** Derived TanStack Query options store bridging the committed search store to `createQuery`. */
  const optionsStore = createSearchQueryOptions(committedSearchStore, localCache);

  // Bridges the derived options store to a rune so createQuery re-evaluates on committed search changes.
  let currentOptions: CreateQueryOptions<SearchResponse, SearchClientError, SearchResponse, SearchQueryKey> = $derived({
    ...$optionsStore,
    enabled: searchEnabled && $optionsStore.enabled === true
  });

  // Bridges the immediate search store to a rune for template reads (page) and cache-key derivation.
  let state = $derived($searchStore);

  // Bridges the online flag to a rune without subscribing to the writable we mutate below.
  let online = $derived($offlineStatus.online);

  /** TanStack Query result driving the ResultsGrid; reactive via the options rune. */
  const query = createQuery<SearchResponse, SearchClientError, SearchResponse, SearchQueryKey>(() => currentOptions);

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
      const key = searchRequestKey({ ...state, query: state.submittedQuery });
      const hasCachedResult = localCache.has(key);
      const isStaleCachedResult = localCache.isStale(key, LOCAL_CACHE_STALE_MS);
      setShowingCached(hasCachedResult && !isStaleCachedResult);
      setShowingStale(hasCachedResult && isStaleCachedResult);
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

  /** True only when the current search-mode request is explicitly eligible to render results. */
  let hasStartedSearching = $derived(currentOptions.enabled === true);

  /** True only for explicit submitted result-search requests, not autocomplete suggestions. */
  let searchInFlight = $derived(hasStartedSearching && query.isFetching === true);

  /** True only when the active submitted search can render final results or a final error. */
  let shouldRenderResults = $derived(hasStartedSearching && !searchInFlight && (query.data !== undefined || errorMessage !== null));

  // Lifts submitted-search loading state so the shell can render the spinner inside the search input.
  $effect(() => {
    onSearchInFlightChange(searchInFlight);
  });

  /** Adds a full Catalog result to the Substitution Input list while preserving display data. */
  function addCatalogResultToSubstitutions(item: FoodObject): void {
    addSubstitutionInput(
      {
        foodObjectId: item.id,
        quantity: 100,
        unit: displayUnitForBasis(item.macroBasis, $preferencesStore.unitSystem)
      },
      item.name,
      item
    );
  }
</script>

<!-- Implements DESIGN-001 SearchView ResultsGrid wiring from TanStack Query result. -->
{#if shouldRenderResults}
  <ResultsGrid
    results={query.data?.items ?? []}
    similarityMetadata={query.data?.similarityMetadata ?? []}
    similarityScores={query.data?.similarityScores ?? []}
    sourceSummary={query.data?.sourceSummary ?? null}
    showSimilarity={state.mode !== "catalog"}
    onAddToSubstitution={state.mode === "catalog" ? addCatalogResultToSubstitutions : null}
    error={errorMessage}
    loading={searchInFlight}
    totalCount={query.data?.totalCount ?? 0}
    page={state.page}
    onPageChange={setPage}
  />
{/if}
