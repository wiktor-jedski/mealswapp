<script lang="ts">
  import SearchSettings from "./SearchSettings.svelte";
  import { createSettingsStore } from "../stores/settings";
  import SearchControls from "./SearchControls.svelte";
  import { createSearchStateStore } from "../search/search-state";
  import { SearchAPIClient } from "../api/search-client";
  import { AppClientError } from "../api/search-client";
  import type { AppError, SearchRejection, SearchResponse } from "../api/generated";
  import { buildSearchRequest } from "../search/search-state";
  import { get } from "svelte/store";
  import { onDestroy } from "svelte";
  import ResultsGrid from "./ResultsGrid.svelte";
  import ActivitySidebar from "./ActivitySidebar.svelte";
  import { ActivityClient } from "../api/activity-client";
  import { createOnlineStatus } from "../stores/online";
  import OfflineBanner from "./OfflineBanner.svelte";
  import { QueryObserver, useQueryClient } from "@tanstack/svelte-query";

  const searchSettings = createSettingsStore();
  const searchState = createSearchStateStore();
  const searchClient = new SearchAPIClient();
  const activityClient = new ActivityClient();
  const onlineStatus = createOnlineStatus();
  let results: SearchResponse | null = null;
  let searchError: AppError | null = null;
  let searchRejection: SearchRejection | null = null;
  let searching = false;
  let cachedResult = false;
  let staleResult = false;
  const queryClient = useQueryClient();
  const initialRequest = buildSearchRequest(get(searchState));
  // Implements DESIGN-001 SearchView observed pagination with previous-page retention.
  const searchObserver = new QueryObserver(queryClient, {
    ...searchClient.searchQueryOptions(initialRequest),
    enabled: false
  });
  const unsubscribeSearch = searchObserver.subscribe((query) => {
    searching = query.isFetching;
    if (query.data) {
      results = query.data;
      cachedResult = false;
      staleResult = false;
    }
    if (query.error) {
      searchError = query.error instanceof AppClientError
        ? query.error.detail
        : { category: "unknown", code: "unknown", message: "Search failed", retryable: false };
      searchRejection = query.error instanceof AppClientError ? query.error.rejection ?? null : null;
    }
  });
  onDestroy(unsubscribeSearch);

  async function executeSearch() {
    searchError = null;
    searchRejection = null;
    cachedResult = false;
    staleResult = false;
    const request = buildSearchRequest(get(searchState));
    if ($onlineStatus) {
      const options = searchClient.searchQueryOptions(request);
      const currentKey = JSON.stringify(searchObserver.options.queryKey);
      searchObserver.setOptions({ ...options, enabled: true, retry: false });
      if (currentKey === JSON.stringify(options.queryKey) && searchObserver.getCurrentResult().isError) {
        await searchObserver.refetch();
      }
      return;
    }
    searching = true;
    searchObserver.setOptions({ ...searchClient.searchQueryOptions(request), enabled: false });
    try {
      const loaded = await searchClient.searchWithCache(request, false);
      results = loaded.response; cachedResult = loaded.cached; staleResult = loaded.stale;
    }
    catch (error) { searchError = error instanceof AppClientError ? error.detail : { category: "unknown", code: "unknown", message: "Search failed", retryable: false }; searchRejection = error instanceof AppClientError ? error.rejection ?? null : null; }
    finally { searching = false; }
  }

  async function changePage(page: number) {
    searchState.update((state) => ({ ...state, page }));
    await executeSearch();
  }
</script>

<!-- Implements DESIGN-001 SearchView, SidebarComponent, SettingsPanel, and DESIGN-016 LayoutGrid. -->
<main class="min-h-screen">
  <section class="mx-auto grid min-h-screen max-w-6xl grid-cols-1 gap-6 px-4 py-6 sm:grid-cols-12 sm:px-6">
    <ActivitySidebar searchState={searchState} loadActivity={() => activityClient.load()} />

    <div class="flex min-w-0 flex-col gap-5 sm:col-span-9">
      <header class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p class="font-data text-xs uppercase text-[var(--color-muted)]">Phase 05 Search</p>
          <h2 class="mt-1 text-xl font-semibold">Food discovery</h2>
        </div>
      </header>

      <OfflineBanner online={$onlineStatus} cached={cachedResult} stale={staleResult} cacheMiss={searchError?.code === "offline_cache_miss"} />

      <SearchControls state={searchState} loadAutocomplete={async (query) => (await searchClient.autocomplete(query)).items} onSearch={executeSearch} />

      <SearchSettings settings={searchSettings} />
      <ResultsGrid response={results} loading={searching} error={searchError} rejection={searchRejection} onPage={changePage} onRetry={executeSearch} />
    </div>
  </section>
</main>
