<script lang="ts">
	import { onDestroy, onMount } from "svelte";
	import { accountDataApi, type AccountDataApi } from "../api/account-data-client";
	import type { CustomItem } from "../api/generated";

	// Implements DESIGN-008 DataExporter/ProfileController private-data controls in DESIGN-009 UserAdminPanel.

	interface Props { api?: AccountDataApi }
	let { api = accountDataApi }: Props = $props();
	let items = $state<CustomItem[]>([]);
	let loading = $state(true);
	let error = $state("");
	let message = $state("");
	let pendingDelete = $state<CustomItem | undefined>();
	let controller: AbortController | undefined;

	onMount(() => { void refresh(); });
	onDestroy(() => controller?.abort());

	async function refresh(successMessage = ""): Promise<void> {
		controller?.abort(); controller = new AbortController(); loading = true; error = "";
		try {
			const bundle = await api.loadExport(controller.signal);
			items = bundle.customItems as CustomItem[];
			message = successMessage;
		} catch (cause) {
			if (!controller.signal.aborted) error = cause instanceof Error ? cause.message : "Account data could not be refreshed. Try again.";
		} finally { if (!controller.signal.aborted) loading = false; }
	}

	async function confirmDelete(): Promise<void> {
		const item = pendingDelete; if (!item) return;
		pendingDelete = undefined; loading = true; error = ""; message = "";
		controller?.abort(); controller = new AbortController();
		try { await api.deleteCustomItem(item.id, controller.signal); await refresh("Private item deleted and authoritative export refreshed."); }
		catch (cause) { if (!controller.signal.aborted) { error = cause instanceof Error ? cause.message : "The private item could not be deleted. Try again."; loading = false; } }
	}
</script>

<!-- Implements DESIGN-008 owner-free account export and explicit private-item deletion flow. -->
<section class="grid gap-3 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4" aria-labelledby="admin-private-data-title" data-admin-private-data>
	<div class="flex flex-wrap items-center justify-between gap-2">
		<div><h2 id="admin-private-data-title" class="text-lg font-semibold">Current admin private data</h2><p class="text-sm text-[var(--color-muted)]">Review the generated account export before deleting a private custom item.</p></div>
		<button type="button" class="rounded border border-[var(--color-border)] px-3 py-2 text-sm focus:ring-2 focus:ring-[var(--color-primary)]" onclick={() => void refresh()} disabled={loading}>Refresh export</button>
	</div>
	{#if loading}<p role="status">Loading authoritative account export…</p>{/if}
	{#if error}<p role="alert" class="text-[var(--color-error)]">{error}</p>{/if}
	{#if message}<p role="status">{message}</p>{/if}
	{#if !loading && !error && items.length === 0}<p data-admin-private-data-empty>No private custom items are present in the account export.</p>{/if}
	{#if items.length > 0}
		<ul class="grid gap-2" aria-label="Private custom items from account export">
			{#each items as item (item.id)}
				<li class="flex flex-wrap items-center justify-between gap-2 rounded border border-[var(--color-border)] p-3"><span>{item.name}</span><button type="button" class="rounded border border-[var(--color-error)] px-3 py-2 text-sm focus:ring-2 focus:ring-[var(--color-primary)]" onclick={() => pendingDelete = item}>Delete private item</button></li>
			{/each}
		</ul>
	{/if}
	{#if pendingDelete}
		<div class="flex flex-wrap items-center gap-2 rounded border border-[var(--color-error)] p-3" role="alertdialog" aria-label={`Confirm deletion of ${pendingDelete.name}`}>
			<p>Delete {pendingDelete.name}? This refreshes the authoritative export after the server confirms deletion.</p>
			<button type="button" class="rounded bg-[var(--color-error)] px-3 py-2 text-[var(--color-on-error)]" onclick={() => void confirmDelete()}>Confirm private item deletion</button>
			<button type="button" class="rounded border px-3 py-2" onclick={() => pendingDelete = undefined}>Cancel</button>
		</div>
	{/if}
</section>
