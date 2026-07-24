<script lang="ts">
  import type { AdminAccessState } from "../admin-access";
  import AdminDataManagement from "./AdminDataManagement.svelte";
  import AdminPrivateData from "./AdminPrivateData.svelte";
  import ExternalImportWorkflow from "./ExternalImportWorkflow.svelte";

  // Implements DESIGN-009 UserAdminPanel responsive, feature-local loading/error shell.

  interface Props {
    access: Exclude<AdminAccessState, "denied">;
    onViewLocalItem?: (name: string) => void;
  }

  let { access, onViewLocalItem = () => undefined }: Props = $props();
</script>

<!-- Implements DESIGN-009 UserAdminPanel administration shell; backend authorization remains authoritative for admin APIs. -->
<section class="grid min-w-0 gap-5" aria-labelledby="administration-panel-title" data-administration-panel>
  {#if access === "loading"}
    <div class="grid gap-3 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-5" role="status" aria-live="polite" data-admin-loading>
      <h1 id="administration-panel-title" class="text-2xl font-semibold">Administration Panel</h1>
      <p class="text-sm text-[var(--color-muted)]">Verifying administration access…</p>
    </div>
  {:else if access === "error"}
    <div class="grid gap-3 rounded border border-[var(--color-error)] bg-[var(--color-surface)] p-5" role="alert" data-admin-error>
      <h1 id="administration-panel-title" class="text-2xl font-semibold">Administration Panel</h1>
      <p>Administration access could not be verified. Refresh the page to try again.</p>
    </div>
  {:else}
    <header class="grid gap-2">
      <p class="font-data text-xs uppercase tracking-wide text-[var(--color-muted)]">Restricted workspace</p>
      <h1 id="administration-panel-title" class="text-2xl font-semibold sm:text-3xl">Administration Panel</h1>
      <p class="max-w-3xl text-sm leading-6 text-[var(--color-muted)]">
        Manage curated food data and restricted account operations from this workspace.
      </p>
    </header>

    <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3" data-admin-responsive-grid>
      <section class="grid gap-2 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
        <h2 class="text-lg font-semibold">External data</h2>
        <p class="text-sm text-[var(--color-muted)]">Search and curate provider records.</p>
      </section>
      <section class="grid gap-2 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4">
        <h2 class="text-lg font-semibold">Catalog management</h2>
        <p class="text-sm text-[var(--color-muted)]">Maintain global items and classifications.</p>
      </section>
      <section class="grid gap-2 rounded border border-[var(--color-border)] bg-[var(--color-surface)] p-4 sm:col-span-2 lg:col-span-1">
        <h2 class="text-lg font-semibold">User administration</h2>
        <p class="text-sm text-[var(--color-muted)]">Use restricted, privacy-minimized account actions.</p>
      </section>
    </div>

    <!-- Implements DESIGN-009 ExternalSearchProxy and DataImporter task 255 feature surface. -->
    <ExternalImportWorkflow {onViewLocalItem} />

    <p class="rounded border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-3 text-sm text-[var(--color-muted)]" data-admin-server-auth-notice>
      Visibility in this panel does not grant access. The server authorizes every administration request.
    </p>

    <AdminPrivateData />

    <!-- Implements DESIGN-009 ItemCurator, TagManager, and UserAdminPanel generated-contract administration views. -->
    <AdminDataManagement />
  {/if}
</section>
