<script lang="ts">
	import { onDestroy, onMount } from "svelte";
	import { adminApi, type AdminApi } from "../api/admin-client";
	import type { AdminMicronutrient } from "../api/generated";

	// Implements DESIGN-005 MicronutrientVocabulary accessible authoritative-refresh administration UI.

	interface Props { api?: AdminApi }
	let { api = adminApi }: Props = $props();
	let entries = $state<AdminMicronutrient[]>([]);
	let key = $state("");
	let displayName = $state("");
	let unit = $state<AdminMicronutrient["unit"]>("mg");
	let busy = $state(false);
	let error = $state("");
	let message = $state("");
	let pending = $state<AdminMicronutrient | undefined>();
	let controller: AbortController | undefined;
	let generation = 0;

	onMount(() => { void refresh(); });
	onDestroy(() => controller?.abort());

	function begin(): { signal: AbortSignal; generation: number } {
		controller?.abort();
		controller = new AbortController();
		return { signal: controller.signal, generation: ++generation };
	}

	async function refresh(): Promise<void> {
		const operation = begin(); busy = true; error = "";
		try {
			const current = await api.listMicronutrients(operation.signal);
			if (operation.generation === generation) entries = current;
		} catch (reason) {
			if (operation.generation === generation && !aborted(reason)) error = safeMessage(reason);
		} finally { if (operation.generation === generation) busy = false; }
	}

	async function create(event: SubmitEvent): Promise<void> {
		event.preventDefault();
		if (!/^[A-Z][A-Za-z0-9]{2,119}$/.test(key) || displayName.trim() !== displayName || !displayName) {
			error = "Enter a PascalCase canonical key and a trimmed display name."; return;
		}
		await mutate((signal) => api.createMicronutrient({ key, displayName, unit }, { signal }), "Micronutrient created and authoritative state refreshed.");
		if (!error) { key = ""; displayName = ""; unit = "mg"; }
	}

	async function saveDisplayName(entry: AdminMicronutrient, input: HTMLInputElement): Promise<void> {
		const value = input.value;
		if (!value || value.trim() !== value || value.length > 120) { error = "Display names must be trimmed and at most 120 characters."; return; }
		await mutate((signal) => api.updateMicronutrientDisplayName(entry.key, value, { signal }), "Display name saved and authoritative state refreshed.");
	}

	async function saveUnit(entry: AdminMicronutrient, select: HTMLSelectElement): Promise<void> {
		await mutate((signal) => api.updateMicronutrientUnit(entry.key, select.value as AdminMicronutrient["unit"], { signal }), "Unit saved and authoritative state refreshed.");
	}

	async function changeActive(entry: AdminMicronutrient): Promise<void> {
		pending = undefined;
		await mutate((signal) => api.setMicronutrientActive(entry.key, !entry.active, { signal }), `Micronutrient ${entry.active ? "deactivated" : "reactivated"} and authoritative state refreshed.`);
	}

	async function mutate(action: (signal: AbortSignal) => Promise<AdminMicronutrient>, success: string): Promise<void> {
		const operation = begin(); busy = true; error = ""; message = "";
		try {
			const changed = await action(operation.signal);
			try {
				const current = await api.listMicronutrients(operation.signal);
				if (operation.generation === generation) { entries = current; message = success; }
			} catch (refreshReason) {
				if (operation.generation === generation && !aborted(refreshReason)) {
					entries = entries.map((entry) => entry.key === changed.key ? changed : entry);
					message = `${success} Refresh is unavailable; showing the successful result.`;
				}
			}
		} catch (reason) {
			if (operation.generation === generation && !aborted(reason)) {
				error = safeMessage(reason);
				try { entries = await api.listMicronutrients(operation.signal); } catch { /* Preserve the safe mutation error and latest known state. */ }
			}
		} finally { if (operation.generation === generation) busy = false; }
	}

	function aborted(reason: unknown): boolean { return reason instanceof Error && reason.name === "AbortError"; }
	function safeMessage(reason: unknown): string { return reason instanceof Error ? reason.message : "The vocabulary action did not complete. No change was shown as successful."; }
	function openModal(node: HTMLDialogElement): { destroy: () => void } {
		const cancel = (event: Event): void => { event.preventDefault(); pending = undefined; };
		node.addEventListener("cancel", cancel);
		node.showModal();
		node.querySelector<HTMLButtonElement>("button")?.focus();
		return { destroy: () => { node.removeEventListener("cancel", cancel); if (node.open) node.close(); } };
	}
</script>

<!-- Implements DESIGN-005 MicronutrientVocabulary; this projection contains no private food-item data. -->
<section class="grid gap-4 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4" aria-labelledby="micronutrient-vocabulary-title" data-admin-micronutrients>
	<div class="flex flex-wrap items-start justify-between gap-3">
		<div><h2 id="micronutrient-vocabulary-title" class="text-lg font-bold">Micronutrient vocabulary</h2><p class="text-sm text-[var(--color-muted)]">Canonical keys cannot be renamed or deleted. In-use entries cannot change unit or be deactivated.</p></div>
		<button type="button" class="rounded border px-3 py-2 focus:ring-2 focus:ring-[var(--color-primary)]" disabled={busy} onclick={() => void refresh()}>Refresh</button>
	</div>
	<form class="grid gap-3 sm:grid-cols-4" aria-label="Add micronutrient" onsubmit={create}>
		<label class="grid gap-1 text-sm">Canonical key<input required minlength="3" maxlength="120" class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 focus:ring-2 focus:ring-[var(--color-primary)]" bind:value={key} /></label>
		<label class="grid gap-1 text-sm sm:col-span-2">Display name<input required maxlength="120" class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 focus:ring-2 focus:ring-[var(--color-primary)]" bind:value={displayName} /></label>
		<label class="grid gap-1 text-sm">Unit<select class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-3 py-2 focus:ring-2 focus:ring-[var(--color-primary)]" bind:value={unit}><option value="g">g</option><option value="mg">mg</option><option value="mcg">mcg</option></select></label>
		<button type="submit" class="rounded bg-[var(--color-primary)] px-4 py-2 font-semibold text-[var(--color-on-primary)] focus:ring-2 focus:ring-[var(--color-primary)] sm:col-span-4 sm:w-fit" disabled={busy}>Add micronutrient</button>
	</form>
	<div class="grid gap-2" aria-busy={busy}>
		{#each entries as entry (entry.key)}
			<article class="grid gap-3 rounded border border-[var(--color-border)] p-3 sm:grid-cols-[minmax(8rem,1fr)_minmax(10rem,2fr)_7rem_auto] sm:items-end" data-micronutrient-key={entry.key}>
				<div><span class="font-data text-sm">{entry.key}</span><span class="ml-2 rounded-full px-2 py-1 text-xs {entry.active ? 'bg-[var(--color-secondary)]' : 'border border-[var(--color-border)]'}">{entry.active ? "Active" : "Inactive"}</span></div>
				<label class="grid gap-1 text-sm">Display name<input maxlength="120" value={entry.displayName} class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-2 focus:ring-2 focus:ring-[var(--color-primary)]" onkeydown={(event) => { if (event.key === "Enter") { event.preventDefault(); void saveDisplayName(entry, event.currentTarget); } }} /></label>
				<label class="grid gap-1 text-sm">Unit<select value={entry.unit} class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-2 py-2 focus:ring-2 focus:ring-[var(--color-primary)]" onchange={(event) => void saveUnit(entry, event.currentTarget)} disabled={busy}><option value="g">g</option><option value="mg">mg</option><option value="mcg">mcg</option></select></label>
				<div class="flex flex-wrap gap-2"><button type="button" class="rounded border px-3 py-2 focus:ring-2 focus:ring-[var(--color-primary)]" disabled={busy} onclick={(event) => { const input = event.currentTarget.closest("article")?.querySelector<HTMLInputElement>("input"); if (input) void saveDisplayName(entry, input); }}>Save name</button>{#if entry.active}<button type="button" class="rounded border border-[var(--color-error)] px-3 py-2 focus:ring-2 focus:ring-[var(--color-primary)]" disabled={busy} onclick={() => pending = entry}>Deactivate</button>{:else}<button type="button" class="rounded border px-3 py-2 focus:ring-2 focus:ring-[var(--color-primary)]" disabled={busy} onclick={() => void changeActive(entry)}>Reactivate</button>{/if}</div>
			</article>
		{/each}
	</div>
	{#if pending}<dialog use:openModal role="alertdialog" aria-modal="true" aria-labelledby="micronutrient-confirm-title" class="m-auto grid max-w-lg gap-3 rounded border-2 border-[var(--color-error)] bg-[var(--color-surface)] p-4 text-[var(--color-text)]"><h3 id="micronutrient-confirm-title" class="font-bold">Confirm deactivation</h3><p>Deactivate <strong>{pending.displayName}</strong>? Imports and item writes using this key will be rejected until reactivation.</p><div class="flex gap-2"><button type="button" class="rounded bg-[var(--color-error)] px-3 py-2 text-[var(--color-on-error)] focus:ring-2 focus:ring-[var(--color-primary)]" onclick={() => void changeActive(pending!)}>Confirm deactivation</button><button type="button" class="rounded border px-3 py-2 focus:ring-2 focus:ring-[var(--color-primary)]" onclick={() => pending = undefined}>Cancel</button></div></dialog>{/if}
	{#if error}<p role="alert" class="text-sm text-[var(--color-error)]">{error}</p>{:else if message}<p role="status" class="text-sm text-[var(--color-muted)]">{message}</p>{/if}
</section>
