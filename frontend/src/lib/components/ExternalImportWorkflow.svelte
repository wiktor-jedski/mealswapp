<script lang="ts">
  import { onMount, tick } from "svelte";
  import type {
    AdminClassification,
    CuratedImportRequest,
    CuratedImportResult,
    ExternalCandidate,
    ExternalCandidateWarning,
    ExternalDataWarning
  } from "../api/generated";
  import {
    createImportIdempotencyKey,
    ExternalAdminClientError,
    importCuratedItem,
    loadAdminClassifications,
    searchExternalFoods,
    type ExternalProvider
  } from "../api/external-admin-client";

  // Implements DESIGN-009 ExternalSearchProxy, ItemCurator, and DataImporter administration workflow.

  interface Props {
    onViewLocalItem?: (name: string) => void;
  }

  interface SearchRequest {
    query: string;
    provider: ExternalProvider;
    page: number;
  }

  let { onViewLocalItem = () => undefined }: Props = $props();
  let query = $state("");
  let provider = $state<ExternalProvider>("all");
  let page = $state(1);
  let searchState = $state<"idle" | "loading" | "results" | "empty" | "error">("idle");
  let searchMessage = $state("");
  let candidates = $state<ExternalCandidate[]>([]);
  let providerWarnings = $state<ExternalDataWarning[]>([]);
  let draft = $state<CuratedImportRequest | null>(null);
  let selectedWarnings = $state<ExternalCandidateWarning[]>([]);
  let foodCategories = $state<AdminClassification[]>([]);
  let culinaryRoles = $state<AdminClassification[]>([]);
  let classificationsMessage = $state("");
  let importState = $state<"idle" | "importing" | "nameConflict" | "blockedConflict" | "ambiguous" | "error" | "success">("idle");
  let importMessage = $state("");
  let importResult = $state<CuratedImportResult | null>(null);
  let importKey = $state("");
  let importConfirmNameConflict = $state(false);
  let blockedConflictCode = $state("");
  let pendingSearch = $state<SearchRequest | null>(null);
  let visibleSelectedWarnings = $derived(selectedWarnings.filter((warning) => warning !== "missing_liquid_density" || !hasValidLiquidDensity(draft)));
  let hasRejectedCandidates = $derived(providerWarnings.some((warning) => warning.code === "invalid_external_payload"));
  let hasProviderFailure = $derived(providerWarnings.some((warning) => warning.code !== "invalid_external_payload"));
  let searchController: AbortController | null = null;
  let searchSequence = 0;
  let importOwnershipToken = 0;
  let keepEditingButton = $state<HTMLButtonElement | null>(null);
  let discardSearchButton = $state<HTMLButtonElement | null>(null);
  let searchInput = $state<HTMLInputElement | null>(null);
  let draftNameInput = $state<HTMLInputElement | null>(null);
  let draftBoundaryOpener: HTMLElement | null = null;

  const warningLabels: Record<ExternalCandidateWarning, string> = {
    missing_image: "Image is missing.",
    missing_macros: "Some macro values are missing; review them before import.",
    missing_micronutrients: "Micronutrient data is incomplete.",
    missing_liquid_density: "Liquid density is missing; add it when quantity conversion needs it.",
    uncertain_unit_conversion: "A source unit conversion needs review.",
    suspicious_liquid_macros: "Liquid nutrition values may use an unexpected basis.",
    partial_normalization: "Some optional source measures were ignored."
  };

  const providerWarningLabels: Record<ExternalDataWarning["code"], string> = {
    provider_rate_limited: "One provider is rate limited; available results are shown.",
    provider_unavailable: "One provider is unavailable; available results are shown.",
    timeout: "One provider timed out; available results are shown.",
    retry_exhausted: "One provider could not be reached after retries.",
    invalid_external_payload: "Some provider candidates were rejected because required data was invalid."
  };

  onMount(() => {
    void loadClassifications();
    return () => {
      searchController?.abort();
      invalidateImportOwnership();
    };
  });

  async function loadClassifications(): Promise<void> {
    classificationsMessage = "";
    try {
      [foodCategories, culinaryRoles] = await Promise.all([
        loadAdminClassifications("food_category"),
        loadAdminClassifications("culinary_role")
      ]);
    } catch {
      classificationsMessage = "Classifications are temporarily unavailable. Retry before importing.";
    }
  }

  function requestSearch(targetPage = 1): void {
    if (!query.trim() || importState === "importing") return;
    const request = { query: query.trim(), provider, page: targetPage };
    if (!draft) {
      resetCuration();
      void runSearch(request);
      return;
    }
    draftBoundaryOpener = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    pendingSearch = request;
    void tick().then(() => keepEditingButton?.focus());
  }

  async function runSearch(request: SearchRequest): Promise<void> {
    const sequence = ++searchSequence;
    searchController?.abort();
    const controller = new AbortController();
    searchController = controller;
    searchState = "loading";
    searchMessage = "";
    candidates = [];
    providerWarnings = [];
    try {
      const result = await searchExternalFoods(request.query, request.provider, request.page, controller.signal);
      if (sequence !== searchSequence) return;
      page = result.page;
      candidates = result.candidates;
      providerWarnings = result.warnings;
      searchState = candidates.length === 0 ? "empty" : "results";
    } catch (error) {
      if (sequence !== searchSequence) return;
      searchState = "error";
      searchMessage = error instanceof ExternalAdminClientError
        ? withRetryAfter(error.message, error.retryAfterSeconds)
        : "External food search is temporarily unavailable. Try again.";
    } finally {
      if (sequence === searchSequence) searchController = null;
    }
  }

  function selectCandidate(candidate: ExternalCandidate): void {
    invalidateImportOwnership();
    pendingSearch = null;
    draft = {
      externalRecordToken: candidate.recordToken,
      name: candidate.name,
      physicalState: candidate.physicalState,
      macrosPer100: { ...candidate.macrosPer100 },
      micros: { ...candidate.micronutrients },
      foodCategoryIds: [],
      culinaryRoleIds: [],
      ...(candidate.densityGramsPerMilliliter ? { densityGramsPerMilliliter: candidate.densityGramsPerMilliliter, densitySourceKind: "imported" as const } : {}),
      ...(candidate.imageUrl ? { imageUrl: candidate.imageUrl } : {})
    };
    selectedWarnings = candidate.warnings;
    importKey = createImportIdempotencyKey();
    importConfirmNameConflict = false;
    blockedConflictCode = "";
    importState = "idle";
    importMessage = "";
    importResult = null;
  }

  function keepEditing(): void {
    pendingSearch = null;
    draftBoundaryOpener = null;
    void tick().then(() => draftNameInput?.focus());
  }

  function handleDraftBoundaryKeydown(event: KeyboardEvent): void {
    if (event.key === "Escape") {
      event.preventDefault();
      keepEditing();
      return;
    }
    if (event.key !== "Tab") return;
    if (event.shiftKey && event.target === keepEditingButton) {
      event.preventDefault();
      discardSearchButton?.focus();
    } else if (!event.shiftKey && event.target === discardSearchButton) {
      event.preventDefault();
      keepEditingButton?.focus();
    }
  }

  function discardDraftAndSearch(): void {
    if (!pendingSearch) return;
    const request = pendingSearch;
    const opener = draftBoundaryOpener;
    draftBoundaryOpener = null;
    pendingSearch = null;
    resetCuration();
    void runSearch(request);
    void tick().then(() => restoreDraftBoundaryFocus(opener));
  }

  function restoreDraftBoundaryFocus(opener: HTMLElement | null): void {
    for (const candidate of [opener, searchInput]) {
      if (!candidate?.isConnected || candidate.matches(":disabled, [aria-disabled='true']") || candidate.closest("[inert]")) continue;
      candidate.focus();
      if (document.activeElement === candidate) return;
    }
  }

  // Implements DESIGN-009 ItemCurator draft ownership boundary with native modal and focus containment.
  function openDraftBoundary(node: HTMLDialogElement): { destroy: () => void } {
    const recoverFocus = (event: FocusEvent): void => {
      if (pendingSearch && event.target !== keepEditingButton && event.target !== discardSearchButton) keepEditingButton?.focus();
    };
    const cancel = (event: Event): void => {
      event.preventDefault();
      keepEditing();
    };
    node.addEventListener("keydown", handleDraftBoundaryKeydown);
    node.addEventListener("cancel", cancel);
    document.addEventListener("focusin", recoverFocus);
    node.showModal();
    queueMicrotask(() => keepEditingButton?.focus());
    return {
      destroy: () => {
        node.removeEventListener("keydown", handleDraftBoundaryKeydown);
        node.removeEventListener("cancel", cancel);
        document.removeEventListener("focusin", recoverFocus);
        if (node.open) node.close();
      }
    };
  }

  function toggleClassification(kind: "food_category" | "culinary_role", id: string, checked: boolean): void {
    if (!draft) return;
    const field = kind === "food_category" ? "foodCategoryIds" : "culinaryRoleIds";
    const current = draft[field];
    draft = { ...draft, [field]: checked ? [...current, id] : current.filter((value) => value !== id) };
  }

  function updatePhysicalState(physicalState: "solid" | "liquid"): void {
    if (!draft) return;
    if (physicalState === "liquid") {
      draft = { ...draft, physicalState };
      return;
    }
    draft = { ...draft, physicalState, densityGramsPerMilliliter: undefined, densitySourceKind: undefined, averageServingVolumeMilliliters: undefined };
  }

  function updateDensity(density: number): void {
    if (!draft || draft.physicalState !== "liquid") return;
    const normalizedDensity = Number.isFinite(density) ? density : undefined;
    draft = { ...draft, densityGramsPerMilliliter: normalizedDensity, ...normalizedDensity && normalizedDensity > 0 && !draft.densitySourceKind ? { densitySourceKind: "manual" as const } : {} };
  }

  function updateDensitySourceKind(kind: "imported" | "manual" | "estimated"): void {
    if (!draft) return;
    draft = { ...draft, densitySourceKind: kind };
  }

  async function submitImport(confirmNameConflict = false): Promise<void> {
    if (importState === "importing") return;
    if (!draft || !validDraft(draft)) {
      importState = "error";
      importMessage = "Complete the required name and non-negative macro fields before importing.";
      return;
    }
    const ownershipToken = ++importOwnershipToken;
    const draftSnapshot = snapshotDraft(draft, confirmNameConflict);
    const keySnapshot = importKey;
    importState = "importing";
    importMessage = "";
    importConfirmNameConflict = confirmNameConflict;
    try {
      const result = await importCuratedItem(draftSnapshot, keySnapshot);
      if (ownershipToken !== importOwnershipToken) return;
      importResult = result;
      importState = "success";
      resetCompletedDraft();
    } catch (error) {
      if (ownershipToken !== importOwnershipToken) return;
      if (error instanceof ExternalAdminClientError && error.appError.code === "name_conflict_confirmation_required") {
        importState = "nameConflict";
        importMessage = error.message;
      } else if (error instanceof ExternalAdminClientError && error.status === 409) {
        importState = "blockedConflict";
        blockedConflictCode = error.appError.code;
        importMessage = error.message;
      } else if (error instanceof ExternalAdminClientError && error.appError.code === "external_request_ambiguous") {
        importState = "ambiguous";
        importMessage = error.message;
      } else {
        importState = "error";
        importMessage = error instanceof ExternalAdminClientError ? error.message : "The item could not be imported. Try again.";
      }
    }
  }

  function validDraft(value: CuratedImportRequest): boolean {
    const macros = value.macrosPer100;
    const validBase = value.name.trim().length > 0 && [macros.protein, macros.carbohydrates, macros.fat].every((number) => Number.isFinite(number) && number >= 0);
    if (!validBase) return false;
    if (value.physicalState === "solid") return value.densityGramsPerMilliliter === undefined && value.densitySourceKind === undefined;
    return hasValidLiquidDensity(value);
  }

  function hasValidLiquidDensity(value: CuratedImportRequest | null): boolean {
    if (!value || value.physicalState !== "liquid" || typeof value.densityGramsPerMilliliter !== "number" || !Number.isFinite(value.densityGramsPerMilliliter) || value.densityGramsPerMilliliter <= 0) return false;
    if (value.densitySourceKind === "manual" || value.densitySourceKind === "estimated") return true;
    return value.densitySourceKind === "imported" && Boolean(value.externalRecordToken);
  }

  function startFreshImport(): void {
    importKey = createImportIdempotencyKey();
    importConfirmNameConflict = false;
    blockedConflictCode = "";
    void submitImport(false);
  }

  function snapshotDraft(value: CuratedImportRequest, confirmNameConflict: boolean): CuratedImportRequest {
    return {
      ...value,
      macrosPer100: { ...value.macrosPer100 },
      micros: { ...value.micros },
      foodCategoryIds: [...value.foodCategoryIds],
      culinaryRoleIds: [...value.culinaryRoleIds],
      confirmNameConflict
    };
  }

  function invalidateImportOwnership(): void {
    importOwnershipToken += 1;
  }

  function resetCompletedDraft(): void {
    draft = null;
    selectedWarnings = [];
    importKey = "";
    importConfirmNameConflict = false;
    blockedConflictCode = "";
  }

  function resetCuration(): void {
    invalidateImportOwnership();
    pendingSearch = null;
    draft = null;
    selectedWarnings = [];
    importKey = "";
    importConfirmNameConflict = false;
    blockedConflictCode = "";
    importState = "idle";
    importMessage = "";
    importResult = null;
  }

  function withRetryAfter(message: string, seconds?: number): string {
    return seconds ? `${message} Retry in about ${seconds} seconds.` : message;
  }
</script>

<!-- Implements DESIGN-009 ExternalSearchProxy provider search, curation, conflict, and import-result UI. -->
<section class="grid gap-5 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4" aria-labelledby="external-import-title" data-external-import-workflow>
  <div class="contents" data-draft-search-background inert={pendingSearch ? true : undefined}>
  <header class="grid gap-1">
    <h2 id="external-import-title" class="text-xl font-bold">External food import</h2>
    <p class="text-sm text-[var(--color-muted)]">Search normalized provider records, review every field, then explicitly import one local item.</p>
  </header>

  <form class="grid gap-3 sm:grid-cols-[minmax(0,1fr)_12rem_auto]" onsubmit={(event) => { event.preventDefault(); requestSearch(1); }} data-external-search-form>
    <label class="grid gap-1 text-sm font-medium">
      External food search
      <input bind:this={searchInput} class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" bind:value={query} disabled={importState === "importing"} required maxlength="200" />
    </label>
    <label class="grid gap-1 text-sm font-medium">
      Provider
      <select class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" bind:value={provider} disabled={importState === "importing"}>
        <option value="all">USDA + OpenFoodFacts</option>
        <option value="usda">USDA</option>
        <option value="openfoodfacts">OpenFoodFacts</option>
      </select>
    </label>
    <button type="submit" class="self-end rounded bg-[var(--color-primary)] px-4 py-2 font-semibold text-[var(--color-on-primary)] transition-all duration-200 motion-reduce:transition-none focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] focus:ring-offset-2" disabled={importState === "importing"}>
      {searchState === "loading" ? "Search again" : "Search"}
    </button>
  </form>

  {#if searchState === "loading"}
    <p role="status" aria-live="polite" data-external-loading>Searching external providers…</p>
  {:else if searchState === "empty"}
    {#if hasRejectedCandidates}
      <p role="status" data-external-rejected>Some provider candidates were rejected because required data was invalid.</p>
    {:else if hasProviderFailure}
      <p role="status" data-external-provider-failure>External providers could not return results.</p>
    {:else}
      <p role="status" data-external-empty>No external candidates matched this search.</p>
    {/if}
  {:else if searchState === "error"}
    <div class="grid justify-items-start gap-2" role="alert" data-external-error>
      <p>{searchMessage}</p>
      <button type="button" class="rounded border border-[var(--color-border)] px-3 py-2 transition-all duration-200 motion-reduce:transition-none focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" disabled={importState === "importing"} onclick={() => requestSearch(page)}>Retry search</button>
    </div>
  {/if}

  {#if providerWarnings.length > 0}
    <aside class="rounded border border-[var(--color-accent)] p-3" aria-label="Partial provider results" data-provider-warnings>
      <p class="font-semibold">Partial results</p>
      <ul class="list-disc pl-5 text-sm">
        {#each providerWarnings as warning}
          <li>{warning.provider}: {providerWarningLabels[warning.code]}</li>
        {/each}
      </ul>
    </aside>
  {/if}

  {#if candidates.length > 0}
    <div class="grid gap-3" data-external-results>
      <p class="font-data text-sm text-[var(--color-muted)]">Page {page}</p>
      {#each candidates as candidate}
        <article class="grid gap-2 rounded border border-[var(--color-border)] p-3 sm:grid-cols-[minmax(0,1fr)_auto]">
          <div class="grid gap-1">
            <h3 class="font-bold">{candidate.name}</h3>
            <p class="font-data text-xs uppercase text-[var(--color-muted)]">{candidate.provider}</p>
            {#if candidate.warnings.length > 0}<p class="text-sm text-[var(--color-accent)]">{candidate.warnings.length} review warning(s)</p>{/if}
          </div>
          <button type="button" class="rounded border border-[var(--color-primary)] px-3 py-2 transition-all duration-200 motion-reduce:transition-none focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" disabled={importState === "importing"} onclick={() => selectCandidate(candidate)}>Curate</button>
        </article>
      {/each}
      <nav class="flex gap-2" aria-label="External search pages">
        <button type="button" class="rounded border px-3 py-2 transition-all duration-200 motion-reduce:transition-none focus:outline-none focus:ring-2" disabled={page <= 1 || searchState === "loading" || importState === "importing"} onclick={() => requestSearch(page - 1)}>Previous</button>
        <button type="button" class="rounded border px-3 py-2 transition-all duration-200 motion-reduce:transition-none focus:outline-none focus:ring-2" disabled={searchState === "loading" || importState === "importing"} onclick={() => requestSearch(page + 1)}>Next</button>
      </nav>
    </div>
  {/if}

  {#if draft}
    <form class="grid gap-4 border-t border-[var(--color-border)] pt-5" onsubmit={(event) => { event.preventDefault(); void submitImport(false); }} data-curation-draft>
      <h3 class="text-lg font-bold">Curation draft</h3>
      {#if visibleSelectedWarnings.length > 0}
        <aside class="rounded border border-[var(--color-accent)] p-3" aria-label="Candidate warnings" data-candidate-warnings>
          <ul class="list-disc pl-5 text-sm">{#each visibleSelectedWarnings as warning}<li>{warningLabels[warning]}</li>{/each}</ul>
        </aside>
      {/if}
      <fieldset class="grid gap-4" disabled={importState === "importing"}>
      <div class="grid gap-3 sm:grid-cols-2">
        <label class="grid gap-1 text-sm font-medium">Name<input bind:this={draftNameInput} class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" bind:value={draft.name} required /></label>
        <label class="grid gap-1 text-sm font-medium">Physical state<select class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" value={draft.physicalState} onchange={(event) => updatePhysicalState(event.currentTarget.value as "solid" | "liquid")}><option value="solid">Solid</option><option value="liquid">Liquid</option></select></label>
        <label class="grid gap-1 text-sm font-medium">Protein per 100<input class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" type="number" min="0" step="any" bind:value={draft.macrosPer100.protein} required /></label>
        <label class="grid gap-1 text-sm font-medium">Carbohydrates per 100<input class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" type="number" min="0" step="any" bind:value={draft.macrosPer100.carbohydrates} required /></label>
        <label class="grid gap-1 text-sm font-medium">Fat per 100<input class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" type="number" min="0" step="any" bind:value={draft.macrosPer100.fat} required /></label>
        <label class="grid gap-1 text-sm font-medium">Image URL<input class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" type="url" bind:value={draft.imageUrl} /></label>
        {#if draft.physicalState === "liquid"}
          <label class="grid gap-1 text-sm font-medium">Density (g/ml)<input class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" type="number" min="0.001" step="any" value={draft.densityGramsPerMilliliter ?? ""} oninput={(event) => updateDensity(event.currentTarget.valueAsNumber)} required /></label>
          <label class="grid gap-1 text-sm font-medium">Density provenance<select class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" value={draft.densitySourceKind ?? ""} onchange={(event) => updateDensitySourceKind(event.currentTarget.value as "imported" | "manual" | "estimated")} required><option value="" disabled>Select provenance</option>{#if draft.densitySourceKind === "imported"}<option value="imported">External provider supplied</option>{/if}<option value="manual">Administrator supplied</option><option value="estimated">Administrator estimate</option></select></label>
          {#if hasValidLiquidDensity(draft)}<p class="text-sm text-[var(--color-muted)]" data-density-curation-state>Liquid density and provenance supplied.</p>{/if}
        {/if}
      </div>

      <fieldset class="grid gap-2"><legend class="font-semibold">Food categories</legend><div class="flex flex-wrap gap-3">{#each foodCategories as classification}<label class="flex items-center gap-2 text-sm"><input type="checkbox" class="border border-[var(--color-border)] bg-[var(--color-surface)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" checked={draft.foodCategoryIds.includes(classification.id)} onchange={(event) => toggleClassification("food_category", classification.id, event.currentTarget.checked)} />{classification.name}</label>{/each}</div></fieldset>
      <fieldset class="grid gap-2"><legend class="font-semibold">Culinary roles</legend><div class="flex flex-wrap gap-3">{#each culinaryRoles as classification}<label class="flex items-center gap-2 text-sm"><input type="checkbox" class="border border-[var(--color-border)] bg-[var(--color-surface)] focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)]" checked={draft.culinaryRoleIds.includes(classification.id)} onchange={(event) => toggleClassification("culinary_role", classification.id, event.currentTarget.checked)} />{classification.name}</label>{/each}</div></fieldset>
      {#if classificationsMessage}<p role="alert">{classificationsMessage}</p><button type="button" class="justify-self-start rounded border px-3 py-2 transition-all duration-200 motion-reduce:transition-none" onclick={() => void loadClassifications()}>Retry classifications</button>{/if}
      </fieldset>

      {#if importState === "nameConflict"}
        <div class="grid justify-items-start gap-2 rounded border border-[var(--color-accent)] p-3" role="alertdialog" aria-labelledby="import-conflict-title" data-import-conflict>
          <p id="import-conflict-title" class="font-semibold">Confirm matching item</p><p>{importMessage}</p>
          <div class="flex gap-2"><button type="button" class="rounded bg-[var(--color-primary)] px-3 py-2 text-[var(--color-on-primary)] transition-all duration-200 motion-reduce:transition-none" onclick={() => void submitImport(true)}>Confirm merge</button><button type="button" class="rounded border px-3 py-2 transition-all duration-200 motion-reduce:transition-none" onclick={() => { importState = "idle"; importMessage = ""; }}>Keep editing</button></div>
        </div>
      {:else if importState === "blockedConflict"}
        <div class="grid justify-items-start gap-2" role="alert" data-import-blocked-conflict>
          <p>{importMessage}</p>
          <div class="flex gap-2">
            {#if blockedConflictCode === "idempotency_key_conflict"}<button type="button" class="rounded border px-3 py-2 transition-all duration-200 motion-reduce:transition-none" onclick={startFreshImport}>Start a fresh import attempt</button>{/if}
            {#if blockedConflictCode === "provider_identity_conflict"}<button type="button" class="rounded border px-3 py-2 transition-all duration-200 motion-reduce:transition-none" onclick={() => requestSearch(page)}>Refresh external results</button>{/if}
            <button type="button" class="rounded border px-3 py-2 transition-all duration-200 motion-reduce:transition-none" onclick={() => { importState = "idle"; importMessage = ""; }}>Keep editing</button>
          </div>
        </div>
      {:else if importState === "ambiguous" || importState === "error"}
        <div class="grid justify-items-start gap-2" role="alert" data-import-error><p>{importMessage}</p><button type="button" class="rounded border px-3 py-2 transition-all duration-200 motion-reduce:transition-none" onclick={() => void submitImport(importConfirmNameConflict)}>Retry import safely</button></div>
      {/if}

      <button type="submit" class="justify-self-start rounded bg-[var(--color-primary)] px-4 py-2 font-semibold text-[var(--color-on-primary)] transition-all duration-200 motion-reduce:transition-none focus:outline-none focus:ring-2" disabled={importState === "importing"}>{importState === "importing" ? "Importing…" : "Import curated item"}</button>
    </form>
  {/if}

  {#if importState === "success" && importResult}
    <section class="grid justify-items-start gap-2 rounded border border-[var(--color-primary)] p-4" role="status" data-import-result>
      <h3 class="font-bold">Import complete</h3>
      <p>{importResult.name} is now available in the local catalog{importResult.merged ? " as a merged item" : ""}{importResult.replayed ? " (confirmed retry)" : ""}.</p>
      <button type="button" class="rounded border border-[var(--color-primary)] px-3 py-2 transition-all duration-200 motion-reduce:transition-none focus:outline-none focus:ring-2" onclick={() => onViewLocalItem(importResult!.name)}>View in local search</button>
    </section>
  {/if}
  </div>

  {#if pendingSearch}
    <dialog class="m-auto grid justify-items-start gap-3 rounded border border-[var(--color-accent)] bg-[var(--color-surface)] p-4 text-[var(--color-text)] backdrop:bg-black/40" role="alertdialog" aria-modal="true" aria-labelledby="draft-search-title" aria-describedby="draft-search-description" use:openDraftBoundary data-draft-search-boundary>
      <h3 id="draft-search-title" class="font-bold">Unsaved curation draft</h3>
      <p id="draft-search-description">Starting a new provider search will discard this draft and its import attempt.</p>
      <div class="flex flex-wrap gap-2">
        <button bind:this={keepEditingButton} type="button" class="rounded border px-3 py-2 transition-all duration-200 motion-reduce:transition-none focus:outline-none focus:ring-2" onclick={keepEditing}>Keep editing</button>
        <button bind:this={discardSearchButton} type="button" class="rounded bg-[var(--color-primary)] px-3 py-2 text-[var(--color-on-primary)] transition-all duration-200 motion-reduce:transition-none focus:outline-none focus:ring-2" onclick={discardDraftAndSearch}>Discard draft and search</button>
      </div>
    </dialog>
  {/if}
</section>
