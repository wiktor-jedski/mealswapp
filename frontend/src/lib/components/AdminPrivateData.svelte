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
	let operation = 0;

	onMount(() => { void refresh(); });
	onDestroy(() => { operation++; controller?.abort(); });

	async function refresh(successMessage = ""): Promise<void> {
		const current = beginOperation();
		try {
			const bundle = await api.loadExport(current.controller.signal);
			if (!isCurrent(current)) return;
			items = bundle.customItems as CustomItem[];
			message = successMessage;
		} catch (cause) {
			if (isCurrent(current)) error = cause instanceof Error ? cause.message : "Account data could not be refreshed. Try again.";
		} finally { if (isCurrent(current)) loading = false; }
	}

	async function confirmDelete(): Promise<void> {
		const item = pendingDelete; if (!item) return;
		const current = beginOperation();
		let deletionAccepted = false;
		try {
			await api.deleteCustomItem(item.id, current.controller.signal);
			if (!isCurrent(current)) return;
			deletionAccepted = true;
			const bundle = await api.loadExport(current.controller.signal);
			if (!isCurrent(current)) return;
			items = bundle.customItems as CustomItem[];
			message = "Private item deleted and authoritative export refreshed.";
		} catch (cause) {
			if (!isCurrent(current)) return;
			error = deletionAccepted
				? "The private item was deleted, but current account data could not be verified. Refresh the export before continuing."
				: cause instanceof Error ? cause.message : "The private item could not be deleted. Try again.";
		} finally { if (isCurrent(current)) loading = false; }
	}

	// Implements DESIGN-008 DataExporter fail-closed authoritative refresh boundary.
	function beginOperation(): { controller: AbortController; operation: number } {
		controller?.abort();
		controller = new AbortController();
		operation++;
		loading = true;
		error = "";
		message = "";
		pendingDelete = undefined;
		items = [];
		return { controller, operation };
	}

	function isCurrent(current: { controller: AbortController; operation: number }): boolean {
		return operation === current.operation && !current.controller.signal.aborted;
	}
</script>

<!-- Implements DESIGN-008 owner-free account export and explicit private-item deletion flow. -->
<section class="grid gap-3 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4" aria-labelledby="admin-private-data-title" data-admin-private-data>
	<div class="flex flex-wrap items-center justify-between gap-2">
		<div><h2 id="admin-private-data-title" class="text-lg font-bold">Current admin private data</h2><p class="text-sm text-[var(--color-muted)]">Review the generated account export before deleting a private custom item.</p></div>
		<button type="button" class="rounded border border-[var(--color-border)] px-3 py-2 text-sm transition-all duration-200 motion-reduce:transition-none focus:ring-2 focus:ring-[var(--color-primary)]" onclick={() => void refresh()} disabled={loading}>Refresh export</button>
	</div>
	{#if loading}<p role="status">Loading authoritative account export…</p>{/if}
	{#if error}<p role="alert" class="text-[var(--color-error)]">{error}</p>{/if}
	{#if message}<p role="status">{message}</p>{/if}
	{#if !loading && !error && items.length === 0}<p data-admin-private-data-empty>No private custom items are present in the account export.</p>{/if}
	{#if items.length > 0}
		<ul class="grid gap-2" aria-label="Private custom items from account export">
			{#each items as item (item.id)}
				<li class="flex flex-wrap items-center justify-between gap-2 rounded border border-[var(--color-border)] p-3"><span>{item.name}</span><button type="button" class="rounded border border-[var(--color-error)] px-3 py-2 text-sm transition-all duration-200 motion-reduce:transition-none focus:ring-2 focus:ring-[var(--color-primary)]" onclick={() => pendingDelete = item}>Delete private item</button></li>
			{/each}
		</ul>
	{/if}
	{#if pendingDelete}
		<div class="flex flex-wrap items-center gap-2 rounded border border-[var(--color-error)] p-3" role="alertdialog" aria-label={`Confirm deletion of ${pendingDelete.name}`}>
			<p>Delete {pendingDelete.name}? This refreshes the authoritative export after the server confirms deletion.</p>
			<button type="button" class="rounded bg-[var(--color-error)] px-3 py-2 text-[var(--color-on-error)] transition-all duration-200 motion-reduce:transition-none" onclick={() => void confirmDelete()}>Confirm private item deletion</button>
			<button type="button" class="rounded border px-3 py-2 transition-all duration-200 motion-reduce:transition-none" onclick={() => pendingDelete = undefined}>Cancel</button>
		</div>
	{/if}
</section>
