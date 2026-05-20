<script lang="ts">
  import { createApiClient } from '../api/client';
  import type { AdminFoodItem, AdminTag, AdminUser, NormalizedExternalCandidate, TagFilterKind } from '../api/types';
  import { createAdminController, createDefaultAdminState, type AdminState, type AdminTab } from '../admin/adminState';

  let state: AdminState = $state(createDefaultAdminState());
  const controller = createAdminController(createApiClient());
  controller.subscribe((next) => {
    state = next;
  });

  const tabs: { id: AdminTab; label: string }[] = [
    { id: 'external', label: 'External' },
    { id: 'items', label: 'Items' },
    { id: 'tags', label: 'Tags' },
    { id: 'users', label: 'Users' },
    { id: 'audit', label: 'Audit' }
  ];

  function itemID(item: AdminFoodItem): string {
    return item.id ?? item.ID ?? '';
  }
  function itemName(item: AdminFoodItem): string {
    return item.name ?? item.Name ?? 'Untitled item';
  }
  function itemState(item: AdminFoodItem): string {
    return item.source?.curationState ?? item.Source?.CurationState ?? 'draft';
  }
  function tagID(tag: AdminTag): string {
    return tag.id ?? tag.ID ?? '';
  }
  function tagName(tag: AdminTag): string {
    return tag.name ?? tag.Name ?? 'Untitled tag';
  }
  function userID(user: AdminUser): string {
    return user.id ?? user.ID ?? '';
  }
  function userEmail(user: AdminUser): string {
    return user.email ?? user.Email ?? '';
  }
  function selectCandidate(candidate: NormalizedExternalCandidate) {
    controller.selectCandidate(candidate);
  }
</script>

<main class="min-h-screen bg-background text-text-primary">
  <div class="mx-auto max-w-app px-4 py-6">
    <header class="flex flex-wrap items-center justify-between gap-3 border-b border-secondary pb-4">
      <div>
        <h1 class="text-2xl font-bold">Admin</h1>
        <p class="font-mono text-sm text-text-muted">Curation, user operations, and audit views</p>
      </div>
      <a class="rounded border border-secondary px-3 py-2 text-sm" href="/">Search</a>
    </header>

    <nav class="mt-4 flex flex-wrap gap-2" aria-label="Admin sections">
      {#each tabs as tab}
        <button
          class="rounded border px-3 py-2 text-sm {state.activeTab === tab.id ? 'border-primary bg-primary text-white' : 'border-secondary bg-surface'}"
          type="button"
          aria-current={state.activeTab === tab.id ? 'page' : undefined}
          onclick={() => controller.setTab(tab.id)}
        >
          {tab.label}
        </button>
      {/each}
    </nav>

    {#if state.error}
      <p class="mt-4 rounded border border-error bg-surface p-3 text-sm text-error" role="alert">{state.error.message}</p>
    {/if}

    {#if state.activeTab === 'external'}
      <section class="mt-5 grid gap-4 lg:grid-cols-[minmax(0,1fr)_320px]" aria-labelledby="external-heading">
        <div>
          <h2 id="external-heading" class="text-lg font-semibold">External Search</h2>
          <div class="mt-3 flex flex-wrap gap-2">
            <label class="sr-only" for="admin-external-query">External food search</label>
            <input
              id="admin-external-query"
              class="min-w-0 flex-1 rounded border border-secondary bg-surface px-3 py-2"
              placeholder="Search provider foods"
              value={state.externalQuery}
              oninput={(event) => controller.setExternalQuery(event.currentTarget.value)}
            />
            <label class="sr-only" for="admin-external-provider">External provider</label>
            <select
              id="admin-external-provider"
              class="rounded border border-secondary bg-surface px-3 py-2"
              value={state.externalProvider}
              onchange={(event) => controller.setExternalProvider(event.currentTarget.value as typeof state.externalProvider)}
            >
              <option value="all">All</option>
              <option value="usda">USDA</option>
              <option value="openfoodfacts">OpenFoodFacts</option>
            </select>
            <button class="rounded bg-primary px-4 py-2 text-white" type="button" onclick={() => void controller.searchExternal()}>
              Search
            </button>
          </div>
          <div class="mt-4 grid gap-2">
            {#each state.external?.candidates ?? [] as candidate}
              <button class="rounded border border-secondary bg-surface p-3 text-left" type="button" aria-label={`Select ${candidate.name} for import preview`} onclick={() => selectCandidate(candidate)}>
                <span class="block font-medium">{candidate.name}</span>
                <span class="font-mono text-xs text-text-muted">{candidate.provider} · P {candidate.macrosPer100.protein} C {candidate.macrosPer100.carbs} F {candidate.macrosPer100.fat}</span>
              </button>
            {/each}
          </div>
        </div>
        <aside class="rounded border border-secondary bg-surface p-4" aria-labelledby="import-preview-heading">
          <h3 id="import-preview-heading" class="font-semibold">Import Preview</h3>
          {#if state.selectedCandidate}
            <p class="mt-2 text-sm">{state.selectedCandidate.name}</p>
            <p class="font-mono text-xs text-text-muted">{state.selectedCandidate.externalId}</p>
            <button class="mt-4 w-full rounded bg-primary px-3 py-2 text-white" type="button" onclick={() => void controller.importSelectedCandidate()}>
              Import
            </button>
          {:else}
            <p class="mt-2 text-sm text-text-muted">Select a provider result.</p>
          {/if}
        </aside>
      </section>
    {:else if state.activeTab === 'items'}
      <section class="mt-5" aria-labelledby="items-heading">
        <div class="flex items-center justify-between gap-3">
          <h2 id="items-heading" class="text-lg font-semibold">Items</h2>
          <button class="rounded border border-secondary px-3 py-2 text-sm" type="button" onclick={() => void controller.loadItems()}>Refresh</button>
        </div>
        <div class="mt-3 overflow-x-auto">
          <table class="w-full border-collapse text-sm">
            <thead><tr class="border-b border-secondary text-left"><th class="py-2" scope="col">Name</th><th scope="col">State</th><th scope="col">Actions</th></tr></thead>
            <tbody>
              {#each state.items?.items ?? [] as item}
                <tr class="border-b border-secondary">
                  <td class="py-2">{itemName(item)}</td>
                  <td class="font-mono text-xs">{itemState(item)}</td>
                  <td class="flex flex-wrap gap-2 py-2">
                    <button class="rounded border border-secondary px-2 py-1" type="button" aria-label={`Approve ${itemName(item)}`} onclick={() => void controller.transitionItem(itemID(item), 'approve')}>Approve</button>
                    <button class="rounded border border-secondary px-2 py-1" type="button" aria-label={`Reject ${itemName(item)}`} onclick={() => void controller.transitionItem(itemID(item), 'reject')}>Reject</button>
                    <button class="rounded border border-secondary px-2 py-1" type="button" aria-label={`Deactivate ${itemName(item)}`} onclick={() => void controller.transitionItem(itemID(item), 'deactivate')}>Deactivate</button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </section>
    {:else if state.activeTab === 'tags'}
      <section class="mt-5" aria-labelledby="tags-heading">
        <h2 id="tags-heading" class="text-lg font-semibold">Tags</h2>
        <div class="mt-3 flex flex-wrap gap-2">
          <label class="sr-only" for="admin-tag-kind">Tag kind</label>
          <select id="admin-tag-kind" class="rounded border border-secondary bg-surface px-3 py-2" bind:value={state.tagKind}>
            <option value="diet">Diet</option>
            <option value="allergen">Allergen</option>
            <option value="functionality">Functionality</option>
            <option value="curation">Curation</option>
          </select>
          <button class="rounded border border-secondary px-3 py-2" type="button" onclick={() => void controller.loadTags(state.tagKind as TagFilterKind)}>Load</button>
        </div>
        <div class="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
          {#each state.tags as tag}
            <div class="rounded border border-secondary bg-surface p-3">
              <p class="font-medium">{tagName(tag)}</p>
              <p class="font-mono text-xs text-text-muted">{tag.kind ?? tag.Kind} · {tagID(tag)}</p>
            </div>
          {/each}
        </div>
      </section>
    {:else if state.activeTab === 'users'}
      <section class="mt-5" aria-labelledby="users-heading">
        <div class="flex items-center justify-between gap-3">
          <h2 id="users-heading" class="text-lg font-semibold">Users</h2>
          <button class="rounded border border-secondary px-3 py-2 text-sm" type="button" onclick={() => void controller.loadUsers()}>Refresh</button>
        </div>
        <div class="mt-3 grid gap-2">
          {#each state.users?.users ?? [] as user}
            <button class="rounded border border-secondary bg-surface p-3 text-left" type="button" aria-label={`Open user ${userEmail(user)}`} onclick={() => void controller.selectUser(userID(user))}>
              <span class="block font-medium">{userEmail(user)}</span>
              <span class="font-mono text-xs text-text-muted">{user.role ?? user.Role} · {(user.disabled ?? user.Disabled) ? 'disabled' : 'active'}</span>
            </button>
          {/each}
        </div>
        {#if state.selectedUser}
          <div class="mt-4 rounded border border-secondary bg-surface p-4">
            <h3 class="font-semibold">{userEmail(state.selectedUser.user)}</h3>
            <p class="font-mono text-xs text-text-muted">{state.selectedUser.entitlement?.plan ?? state.selectedUser.entitlement?.Plan ?? 'no entitlement'}</p>
            <div class="mt-3 flex flex-wrap gap-2">
              <button class="rounded border border-secondary px-3 py-2" type="button" aria-label={`Disable ${userEmail(state.selectedUser.user)}`} onclick={() => void controller.disableUser(userID(state.selectedUser!.user))}>Disable</button>
              <button class="rounded border border-secondary px-3 py-2" type="button" aria-label={`Reset lockout for ${userEmail(state.selectedUser.user)}`} onclick={() => void controller.resetUserLockout(userID(state.selectedUser!.user))}>Reset Lockout</button>
              <button class="rounded border border-secondary px-3 py-2" type="button" aria-label={`Load audit for ${userEmail(state.selectedUser.user)}`} onclick={() => void controller.loadUserAudit(userID(state.selectedUser!.user))}>Audit</button>
            </div>
          </div>
        {/if}
      </section>
    {:else}
      <section class="mt-5" aria-labelledby="audit-heading">
        <h2 id="audit-heading" class="text-lg font-semibold">Audit</h2>
        <div class="mt-3 grid gap-2">
          {#each state.audit?.entries ?? [] as entry}
            <div class="rounded border border-secondary bg-surface p-3">
              <p class="font-medium">{entry.action ?? entry.Action}</p>
              <p class="font-mono text-xs text-text-muted">{entry.target ?? entry.Target}</p>
            </div>
          {/each}
        </div>
      </section>
    {/if}
  </div>
</main>
