<script lang="ts">
	import { onDestroy, onMount, tick } from "svelte";
	import { adminApi, type AdminApi, type ClassificationKind } from "../api/admin-client";
	import type { AdminClassification, AdminItem, AdminUser } from "../api/generated";
	import { deletionRetryEligible, newAdminItemKey, parseAdminItemForm, type AdminItemForm } from "../admin-workflows";

	// Implements DESIGN-009 ItemCurator, TagManager, and UserAdminPanel authoritative administration workflows.

	interface Props { api?: AdminApi }
	let { api = adminApi }: Props = $props();

	const emptyForm = (): AdminItemForm => ({ name: "", physicalState: "solid", prepTimeMinutes: "", averageUnitWeightGrams: "", averageServingVolumeMilliliters: "", protein: "", carbohydrates: "", fat: "", density: "", densitySourceProvider: "", densitySourceFoodId: "", densitySourceKind: "", micros: "{}", foodCategoryIds: [], culinaryRoleIds: [], imageUrl: "" });
	let form = $state<AdminItemForm>(emptyForm());
	let itemId = $state("");
	let currentItem = $state<AdminItem | undefined>();
	let itemBusy = $state(false);
	let itemMessage = $state("");
	let itemError = $state("");
	let createKey = $state("");
	let createBody = $state("");
	let classifications = $state<Record<ClassificationKind, AdminClassification[]>>({ food_category: [], culinary_role: [] });
	let classificationKind = $state<ClassificationKind>("food_category");
	let classificationName = $state("");
	let classificationId = $state("");
	let classificationParentId = $state<string | undefined>();
	let classificationBusy = $state(false);
	let classificationMessage = $state("");
	let classificationError = $state("");
	let userQuery = $state("");
	let users = $state<AdminUser[]>([]);
	let userBusy = $state(false);
	let userMessage = $state("");
	let userError = $state("");
	type Confirmation = Readonly<{ action: "item" | "classification" | "retry"; id: string; label: string; userId?: string }>;
	let confirmation = $state<Confirmation | undefined>();
	let adminRoot: HTMLElement;
	let confirmationDialog = $state<HTMLDialogElement | undefined>();
	let confirmationOpener: HTMLElement | undefined;
	let confirmationContext: HTMLElement | undefined;
	let itemGeneration = 0;
	let classificationGeneration = 0;
	let userGeneration = 0;
	let itemController: AbortController | undefined;
	let classificationController: AbortController | undefined;
	let userController: AbortController | undefined;

	onMount(() => { void refreshClassifications(); });
	onDestroy(() => { itemController?.abort(); classificationController?.abort(); userController?.abort(); });

	function beginItemOperation(): { generation: number; controller: AbortController } {
		itemController?.abort(); const controller = new AbortController(); itemController = controller; return { generation: ++itemGeneration, controller };
	}
	function beginClassificationOperation(): { generation: number; controller: AbortController } {
		classificationController?.abort(); const controller = new AbortController(); classificationController = controller; return { generation: ++classificationGeneration, controller };
	}
	function beginUserOperation(): { generation: number; controller: AbortController } {
		userController?.abort(); const controller = new AbortController(); userController = controller; return { generation: ++userGeneration, controller };
	}
	function currentItemOperation(generation: number, controller: AbortController): boolean { return generation === itemGeneration && !controller.signal.aborted; }
	function currentClassificationOperation(generation: number, controller: AbortController): boolean { return generation === classificationGeneration && !controller.signal.aborted; }
	function currentUserOperation(generation: number, controller: AbortController): boolean { return generation === userGeneration && !controller.signal.aborted; }
	function aborted(error: unknown): boolean { return error instanceof Error && error.name === "AbortError"; }
	async function classificationProjection(signal: AbortSignal): Promise<Record<ClassificationKind, AdminClassification[]>> {
		const [foodCategories, culinaryRoles] = await Promise.all([api.listClassifications("food_category", signal), api.listClassifications("culinary_role", signal)]);
		return { food_category: foodCategories, culinary_role: culinaryRoles };
	}

	async function refreshClassifications(): Promise<void> {
		const { generation, controller } = beginClassificationOperation(); classificationBusy = true;
		try {
			const projection = await classificationProjection(controller.signal);
			if (currentClassificationOperation(generation, controller)) classifications = projection;
		} catch (error) { if (currentClassificationOperation(generation, controller) && !aborted(error)) classificationError = message(error); }
		finally { if (generation === classificationGeneration) classificationBusy = false; }
	}

	async function loadItem(): Promise<void> {
		const id = itemId.trim(); if (!id) { itemError = "Enter an item ID."; return; }
		const { generation, controller } = beginItemOperation();
		itemBusy = true; itemError = ""; itemMessage = "";
		try { const item = await api.getItem(id, controller.signal); if (currentItemOperation(generation, controller)) { applyItem(item); itemMessage = "Authoritative item loaded."; } }
		catch (error) { if (currentItemOperation(generation, controller) && !aborted(error)) { currentItem = undefined; itemError = message(error); } }
		finally { if (generation === itemGeneration) itemBusy = false; }
	}

	function applyItem(item: AdminItem): void {
		currentItem = item; itemId = item.id;
		form = {
			name: item.name,
			physicalState: item.physicalState,
			prepTimeMinutes: String(item.prepTimeMinutes),
			averageUnitWeightGrams: item.averageUnitWeightGrams === undefined ? "" : String(item.averageUnitWeightGrams),
			averageServingVolumeMilliliters: item.averageServingVolumeMilliliters === undefined ? "" : String(item.averageServingVolumeMilliliters),
			protein: String(item.macrosPer100.protein), carbohydrates: String(item.macrosPer100.carbohydrates), fat: String(item.macrosPer100.fat),
			density: item.densityGramsPerMilliliter === undefined ? "" : String(item.densityGramsPerMilliliter),
			densitySourceProvider: item.densitySourceProvider ?? "", densitySourceFoodId: item.densitySourceFoodId ?? "", densitySourceKind: item.densitySourceKind ?? "",
			micros: JSON.stringify(item.micros), foodCategoryIds: item.foodCategoryIds ?? item.foodCategories.map(({ id }) => id), culinaryRoleIds: item.culinaryRoleIds ?? item.culinaryRoles.map(({ id }) => id), imageUrl: item.imageUrl ?? ""
		};
	}

	async function saveItem(event: SubmitEvent): Promise<void> {
		event.preventDefault(); itemError = ""; itemMessage = "";
		const parsed = parseAdminItemForm(form);
		if (!parsed.request) { itemError = parsed.error ?? "Check the item fields."; return; }
		const targetId = currentItem?.id; const { generation, controller } = beginItemOperation(); itemBusy = true;
		try {
			const wasEditing = Boolean(targetId);
			let saved: AdminItem;
			if (targetId) saved = await api.replaceItem(targetId, parsed.request, { signal: controller.signal });
			else {
				const body = JSON.stringify(parsed.request);
				if (!createKey || createBody !== body) { createKey = newAdminItemKey(); createBody = body; }
				saved = await api.createItem(parsed.request, createKey, { signal: controller.signal });
			}
			const projection = await api.getItem(saved.id, controller.signal);
			if (currentItemOperation(generation, controller)) { applyItem(projection); createKey = ""; createBody = ""; itemMessage = wasEditing ? "Item saved and refreshed." : "Item created and refreshed."; }
		} catch (error) {
			if (currentItemOperation(generation, controller) && !aborted(error)) { itemError = message(error); if (targetId) await refreshCurrentItem(targetId, generation, controller); }
		} finally { if (generation === itemGeneration) itemBusy = false; }
	}

	async function refreshCurrentItem(id: string, generation: number, controller: AbortController): Promise<void> {
		try { const item = await api.getItem(id, controller.signal); if (currentItemOperation(generation, controller)) applyItem(item); }
		catch (error) { if (currentItemOperation(generation, controller) && !aborted(error)) currentItem = undefined; }
	}

	function resetItemState(): void { currentItem = undefined; itemId = ""; form = emptyForm(); itemError = ""; createKey = ""; createBody = ""; }
	function newItem(): void { itemController?.abort(); ++itemGeneration; itemBusy = false; resetItemState(); itemMessage = ""; }

	async function deleteItem(target: Confirmation): Promise<void> {
		if (target.action !== "item" || currentItem?.id !== target.id) { itemError = "The confirmed item is no longer current. Reload it before deleting."; return; }
		const { generation, controller } = beginItemOperation(); itemBusy = true; itemError = "";
		try { await api.deleteItem(target.id, { signal: controller.signal }); if (currentItemOperation(generation, controller)) { resetItemState(); itemMessage = "Item deleted after server confirmation."; } }
		catch (error) { if (currentItemOperation(generation, controller) && !aborted(error)) { itemError = message(error); await refreshCurrentItem(target.id, generation, controller); } }
		finally { if (generation === itemGeneration) itemBusy = false; }
	}

	async function saveClassification(event: SubmitEvent): Promise<void> {
		event.preventDefault(); const name = classificationName.trim();
		if (!name || name.length > 120) { classificationError = "Enter a classification name of at most 120 characters."; return; }
		const id = classificationId; const kind = classificationKind; const { generation, controller } = beginClassificationOperation(); classificationBusy = true; classificationError = ""; classificationMessage = "";
		try {
			if (id) await api.replaceClassification(id, { name, parentId: classificationParentId ?? null }, { signal: controller.signal });
			else await api.createClassification(kind, { name }, { signal: controller.signal });
			const projection = await classificationProjection(controller.signal);
			if (currentClassificationOperation(generation, controller)) { classifications = projection; classificationName = ""; classificationId = ""; classificationParentId = undefined; classificationMessage = "Classification saved and refreshed."; }
		} catch (error) {
			if (currentClassificationOperation(generation, controller) && !aborted(error)) {
				classificationError = message(error);
				try { const projection = await classificationProjection(controller.signal); if (currentClassificationOperation(generation, controller)) classifications = projection; } catch { /* Preserve the mutation error and latest known state. */ }
			}
		} finally { if (generation === classificationGeneration) classificationBusy = false; }
	}

	function editClassification(value: AdminClassification): void { classificationKind = value.kind; classificationId = value.id; classificationName = value.name; classificationParentId = value.parentId; classificationError = ""; }

	async function deleteClassification(target: Confirmation): Promise<void> {
		if (target.action !== "classification" || !Object.values(classifications).flat().some(({ id }) => id === target.id)) { classificationError = "The confirmed classification is no longer current. Reload before deleting."; return; }
		const { generation, controller } = beginClassificationOperation(); classificationBusy = true; classificationError = ""; classificationMessage = "";
		try {
			await api.deleteClassification(target.id, { signal: controller.signal });
			const projection = await classificationProjection(controller.signal);
			if (currentClassificationOperation(generation, controller)) { classifications = projection; classificationMessage = "Classification deleted and refreshed."; }
		} catch (error) {
			if (currentClassificationOperation(generation, controller) && !aborted(error)) {
				classificationError = message(error);
				try { const projection = await classificationProjection(controller.signal); if (currentClassificationOperation(generation, controller)) classifications = projection; } catch { /* Preserve the mutation error and latest known state. */ }
			}
		} finally { if (generation === classificationGeneration) classificationBusy = false; }
	}

	async function lookupUsers(): Promise<void> {
		const query = userQuery.trim(); if (!query) { userError = "Enter an exact email or user ID."; return; }
		const { generation, controller } = beginUserOperation(); userBusy = true; userError = ""; userMessage = "";
		try { const projection = (await api.lookupUsers(query.includes("@") ? { email: query } : { userId: query }, controller.signal)).users; if (currentUserOperation(generation, controller)) { users = projection; userMessage = users.length ? "Authoritative user state loaded." : "No matching user."; } }
		catch (error) { if (currentUserOperation(generation, controller) && !aborted(error)) { users = []; userError = message(error); } }
		finally { if (generation === userGeneration) userBusy = false; }
	}

	async function retryDeletion(target: Confirmation): Promise<void> {
		const { userId, id } = target;
		if (target.action !== "retry" || !userId || !users.some((user) => user.id === userId && user.deletion?.requestId === id)) { userError = "The confirmed deletion request is no longer current. Look up the user again."; return; }
		const { generation, controller } = beginUserOperation(); userBusy = true; userError = ""; userMessage = "";
		try {
			await api.retryDeletion(userId, id, { signal: controller.signal });
			const projection = (await api.lookupUsers({ userId }, controller.signal)).users;
			if (currentUserOperation(generation, controller)) { users = projection; userMessage = "Deletion retry accepted and authoritative state refreshed."; }
		} catch (error) {
			if (currentUserOperation(generation, controller) && !aborted(error)) {
				userError = message(error);
				try { const projection = (await api.lookupUsers({ userId }, controller.signal)).users; if (currentUserOperation(generation, controller)) users = projection; } catch { /* Preserve the mutation error and never claim success. */ }
			}
		} finally { if (generation === userGeneration) userBusy = false; }
	}

	async function confirmAction(): Promise<void> {
		const target = confirmation; if (!target) return;
		const opener = confirmationOpener; const context = confirmationContext;
		confirmationOpener = undefined; confirmationContext = undefined;
		closeConfirmationDialog();
		confirmation = undefined;
		await tick();
		restoreConfirmationFocus(opener, context);
		if (target.action === "item") await deleteItem(target);
		else if (target.action === "classification") await deleteClassification(target);
		else await retryDeletion(target);
		await tick();
		restoreConfirmationFocus(opener, context);
	}

	function confirm(target: Confirmation, opener: HTMLElement): void {
		if (target.action === "item") { itemController?.abort(); ++itemGeneration; itemBusy = false; }
		else if (target.action === "classification") { classificationController?.abort(); ++classificationGeneration; classificationBusy = false; }
		else { userController?.abort(); ++userGeneration; userBusy = false; }
		confirmationOpener = opener;
		confirmationContext = opener.closest("section") ?? undefined;
		confirmation = Object.freeze({ ...target });
	}

	async function cancelConfirmation(): Promise<void> {
		const opener = confirmationOpener; const context = confirmationContext;
		confirmationOpener = undefined; confirmationContext = undefined;
		closeConfirmationDialog();
		confirmation = undefined;
		await tick();
		restoreConfirmationFocus(opener, context);
	}

	/** Returns focus after the DESIGN-009 destructive boundary, using an enabled control or the component root if its opener disappeared. */
	function restoreConfirmationFocus(opener: HTMLElement | undefined, context: HTMLElement | undefined): void {
		const focusable = 'button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), a[href], [tabindex]:not([tabindex="-1"])';
		const candidates = [opener, ...(context?.querySelectorAll<HTMLElement>(focusable) ?? []), adminRoot];
		for (const candidate of candidates) {
			if (!candidate?.isConnected || candidate.closest("[inert]") || candidate.getAttribute("aria-disabled") === "true" || ("disabled" in candidate && candidate.disabled)) continue;
			candidate.focus();
			if (document.activeElement === candidate) return;
		}
	}

	function message(error: unknown): string { return error instanceof Error ? error.message : "The administration action did not complete."; }
	/** Uses the native DESIGN-009 modal lifecycle so the complete page background remains keyboard-inert. */
	function openModal(node: HTMLDialogElement): { destroy: () => void } {
		const controls = (): HTMLElement[] => [...node.querySelectorAll<HTMLElement>('button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])')];
		const cancel = (event: Event): void => { event.preventDefault(); void cancelConfirmation(); };
		const containFocusIn = (): void => { const available = controls(); if (!available.includes(document.activeElement as HTMLElement)) available[0]?.focus(); };
		const containFocus = (event: KeyboardEvent): void => {
			if (event.key !== "Tab") return;
			const available = controls();
			const target = event.shiftKey ? available.at(-1) : available[0];
			if (!target || (event.shiftKey ? document.activeElement !== available[0] : document.activeElement !== available.at(-1))) return;
			event.preventDefault(); target.focus();
		};
		node.addEventListener("cancel", cancel);
		node.addEventListener("focusin", containFocusIn);
		node.addEventListener("keydown", containFocus);
		node.showModal();
		return { destroy: () => { node.removeEventListener("cancel", cancel); node.removeEventListener("focusin", containFocusIn); node.removeEventListener("keydown", containFocus); if (node.open) node.close(); } };
	}
	function closeConfirmationDialog(): void { if (confirmationDialog?.open) confirmationDialog.close(); }
	/** Moves keyboard focus into a newly opened confirmation boundary. */
	function focusOnMount(node: HTMLElement): void { queueMicrotask(() => node.focus()); }
</script>

<div class="grid gap-6" data-admin-data-management bind:this={adminRoot} tabindex="-1">
	<div class="contents" data-admin-background inert={confirmation ? true : undefined}>
	<section class="grid gap-4 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4" aria-labelledby="manual-items-title">
		<div class="flex flex-wrap items-start justify-between gap-3"><div><h2 id="manual-items-title" class="text-lg font-semibold">Manual global items</h2><p class="text-sm text-[var(--color-muted)]">Create an ownerless item or load one by ID to edit it.</p></div><button type="button" class="rounded border px-3 py-2 focus:ring-2 focus:ring-[var(--color-primary)]" onclick={newItem} disabled={itemBusy}>New item</button></div>
		<form class="flex flex-col gap-2 sm:flex-row" onsubmit={(event) => { event.preventDefault(); void loadItem(); }} aria-label="Load global item"><label class="grid flex-1 gap-1 text-sm">Item ID<input class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={itemId} /></label><button type="submit" class="self-end rounded border px-3 py-2 focus:ring-2 focus:ring-[var(--color-primary)]" disabled={itemBusy}>Load</button></form>
		<form class="grid gap-3 sm:grid-cols-2" onsubmit={saveItem} aria-label="Manual global item form">
			<label class="grid gap-1 text-sm sm:col-span-2">Name<input required maxlength="200" class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={form.name} /></label>
			<label class="grid gap-1 text-sm">Physical state<select class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={form.physicalState}><option value="solid">Solid</option><option value="liquid">Liquid</option></select></label>
			<label class="grid gap-1 text-sm">Preparation time (minutes)<input inputmode="numeric" class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={form.prepTimeMinutes} /></label>
			<label class="grid gap-1 text-sm">Average unit weight (g)<input inputmode="decimal" class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={form.averageUnitWeightGrams} /></label>
			{#if form.physicalState === "liquid"}
				<label class="grid gap-1 text-sm">Average serving volume (ml)<input inputmode="decimal" class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={form.averageServingVolumeMilliliters} /></label>
				<label class="grid gap-1 text-sm">Density (g/ml)<input inputmode="decimal" class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={form.density} /></label>
				<label class="grid gap-1 text-sm">Density source<select class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={form.densitySourceKind}><option value="">Manual (default)</option><option value="manual">Manual</option><option value="estimated">Estimated</option><option value="imported">Imported</option></select></label>
				<label class="grid gap-1 text-sm">Density source provider<input maxlength="200" class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={form.densitySourceProvider} /></label>
				<label class="grid gap-1 text-sm sm:col-span-2">Density source food ID<input maxlength="200" class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={form.densitySourceFoodId} /></label>
			{/if}
			<label class="grid gap-1 text-sm">Protein per 100<input inputmode="decimal" class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={form.protein} /></label>
			<label class="grid gap-1 text-sm">Carbohydrates per 100<input inputmode="decimal" class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={form.carbohydrates} /></label>
			<label class="grid gap-1 text-sm">Fat per 100<input inputmode="decimal" class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={form.fat} /></label>
			<label class="grid gap-1 text-sm sm:col-span-2">Micronutrients (JSON)<textarea rows="3" class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2 font-data" bind:value={form.micros}></textarea></label>
			<label class="grid gap-1 text-sm sm:col-span-2">Image URL<input maxlength="2048" class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={form.imageUrl} /></label>
			<label class="grid gap-1 text-sm">Food Categories<select multiple size="4" class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={form.foodCategoryIds}>{#each classifications.food_category as value (value.id)}<option value={value.id}>{value.name}</option>{/each}</select></label>
			<label class="grid gap-1 text-sm">Culinary Roles<select multiple size="4" class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={form.culinaryRoleIds}>{#each classifications.culinary_role as value (value.id)}<option value={value.id}>{value.name}</option>{/each}</select></label>
			<div class="flex flex-wrap gap-2 sm:col-span-2"><button type="submit" class="rounded bg-[var(--color-primary)] px-4 py-2 font-semibold text-[var(--color-on-primary)] focus:ring-2 focus:ring-[var(--color-primary)]" disabled={itemBusy}>{currentItem ? "Save item" : "Create item"}</button>{#if currentItem}<button type="button" class="rounded border border-[var(--color-error)] px-4 py-2 focus:ring-2 focus:ring-[var(--color-primary)]" disabled={itemBusy} onclick={(event) => confirm({ action: "item", id: currentItem!.id, label: currentItem!.name }, event.currentTarget)}>Delete item</button>{/if}</div>
		</form>
		{#if itemError}<p role="alert" class="text-sm text-[var(--color-error)]" data-admin-item-error>{itemError}</p>{:else if itemMessage}<p role="status" class="text-sm text-[var(--color-muted)]">{itemMessage}</p>{/if}
	</section>

	<section class="grid gap-4 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4" aria-labelledby="classifications-title">
		<h2 id="classifications-title" class="text-lg font-semibold">Food Categories and Culinary Roles</h2>
		<form class="grid gap-3 sm:grid-cols-[12rem_1fr_auto]" onsubmit={saveClassification} aria-label="Classification form"><label class="grid gap-1 text-sm">Kind<select class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={classificationKind} disabled={Boolean(classificationId)}><option value="food_category">Food Category</option><option value="culinary_role">Culinary Role</option></select></label><label class="grid gap-1 text-sm">Name<input maxlength="120" class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={classificationName} /></label><button type="submit" class="self-end rounded bg-[var(--color-primary)] px-3 py-2 font-semibold text-[var(--color-on-primary)] focus:ring-2 focus:ring-[var(--color-primary)]" disabled={classificationBusy}>{classificationId ? "Save rename" : "Create"}</button></form>
		{#if classificationId}<button type="button" class="w-fit text-sm underline" onclick={() => { classificationId = ""; classificationName = ""; classificationParentId = undefined; }}>Cancel rename</button>{/if}
		<div class="grid gap-4 md:grid-cols-2" data-admin-classification-grid>{#each ["food_category", "culinary_role"] as kind}<div class="grid content-start gap-2"><h3 class="font-semibold">{kind === "food_category" ? "Food Categories" : "Culinary Roles"}</h3><ul class="grid gap-2">{#each classifications[kind as ClassificationKind] as value (value.id)}<li class="flex items-center justify-between gap-2 rounded border border-[var(--color-border)] p-2"><span>{value.name}</span><span class="flex gap-2"><button type="button" class="rounded border px-2 py-1 focus:ring-2 focus:ring-[var(--color-primary)]" disabled={classificationBusy} onclick={() => editClassification(value)}>Rename</button><button type="button" class="rounded border border-[var(--color-error)] px-2 py-1 focus:ring-2 focus:ring-[var(--color-primary)]" disabled={classificationBusy} onclick={(event) => confirm({ action: "classification", id: value.id, label: value.name }, event.currentTarget)}>Delete</button></span></li>{/each}</ul></div>{/each}</div>
		{#if classificationError}<p role="alert" class="text-sm text-[var(--color-error)]" data-admin-classification-error>{classificationError}</p>{:else if classificationMessage}<p role="status" class="text-sm text-[var(--color-muted)]">{classificationMessage}</p>{/if}
	</section>

	<section class="grid gap-4 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4" aria-labelledby="user-admin-title">
		<div><h2 id="user-admin-title" class="text-lg font-semibold">Restricted user lookup</h2><p class="text-sm text-[var(--color-muted)]">Exact lookup returns only the approved account and deletion summary.</p></div>
		<form class="flex flex-col gap-2 sm:flex-row" onsubmit={(event) => { event.preventDefault(); void lookupUsers(); }} aria-label="User lookup"><label class="grid flex-1 gap-1 text-sm">Email or user ID<input class="rounded border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-2" bind:value={userQuery} /></label><button type="submit" class="self-end rounded bg-[var(--color-primary)] px-4 py-2 font-semibold text-[var(--color-on-primary)] focus:ring-2 focus:ring-[var(--color-primary)]" disabled={userBusy}>Look up</button></form>
		{#each users as user (user.id)}<article class="grid gap-2 rounded border border-[var(--color-border)] p-3" data-admin-user><h3 class="font-semibold">{user.email}</h3><dl class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-sm"><dt>ID</dt><dd class="break-all font-data">{user.id}</dd><dt>Verified</dt><dd>{user.emailVerified ? "Yes" : "No"}</dd><dt>Created</dt><dd>{user.createdAt}</dd>{#if user.deletion}<dt>Deletion</dt><dd>{user.deletion.status} · {user.deletion.failureCategory ?? "none"} · retries {user.deletion.retryCount}</dd>{/if}</dl>{#if deletionRetryEligible(user.deletion)}<button type="button" class="w-fit rounded border border-[var(--color-error)] px-3 py-2 focus:ring-2 focus:ring-[var(--color-primary)]" disabled={userBusy} onclick={(event) => confirm({ action: "retry", id: user.deletion!.requestId, userId: user.id, label: user.email }, event.currentTarget)}>Retry legal deletion</button>{/if}</article>{/each}
		{#if userError}<p role="alert" class="text-sm text-[var(--color-error)]" data-admin-user-error>{userError}</p>{:else if userMessage}<p role="status" class="text-sm text-[var(--color-muted)]">{userMessage}</p>{/if}
	</section>
	</div>

	{#if confirmation}<dialog bind:this={confirmationDialog} use:openModal class="sticky bottom-3 m-0 grid w-full max-w-none gap-3 rounded border-2 border-[var(--color-error)] bg-[var(--color-surface)] p-4 text-[var(--color-text)] shadow-lg" aria-labelledby="admin-confirm-title" aria-modal="true" data-admin-confirmation><h2 id="admin-confirm-title" class="font-semibold">Confirm destructive action</h2><p>Confirm {confirmation.action === "retry" ? "deletion retry for" : `deletion of`} <strong>{confirmation.label}</strong>. The server will remain authoritative.</p><div class="flex gap-2"><button type="button" class="rounded bg-[var(--color-error)] px-3 py-2 text-[var(--color-on-muted)] focus:ring-2 focus:ring-[var(--color-primary)]" use:focusOnMount onclick={() => void confirmAction()}>Confirm</button><button type="button" class="rounded border px-3 py-2 focus:ring-2 focus:ring-[var(--color-primary)]" onclick={() => void cancelConfirmation()}>Cancel</button></div></dialog>{/if}
</div>
