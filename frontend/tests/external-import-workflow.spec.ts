import { expect, test, type Page, type Route } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import type {
  AuthSessionEnvelope,
  CuratedImportEnvelope,
  EntitlementStatusEnvelope,
  ExternalCandidate,
  ExternalSearchEnvelope,
  ProfileEnvelope,
  SearchResponseEnvelope
} from "../src/lib/api/generated";

// Implements DESIGN-009 ExternalSearchProxy, ItemCurator, and DataImporter browser verification for tasks 255 and 266.

const candidate: ExternalCandidate = {
  provider: "usda",
  externalId: "usda-apple-255",
  recordToken: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
  name: "Provider Apple Drink",
  physicalState: "liquid",
  macrosPer100: { protein: 0.2, carbohydrates: 12, fat: 0.1 },
  micronutrients: { vitamin_c: 4 },
  warnings: ["missing_liquid_density", "uncertain_unit_conversion", "suspicious_liquid_macros"]
};

const fruitId = "10000000-0000-4000-8000-000000000001";
const drinkId = "10000000-0000-4000-8000-000000000002";
const importId = "10000000-0000-4000-8000-000000000003";
const foodItemId = "10000000-0000-4000-8000-000000000004";

function sessionEnvelope(): AuthSessionEnvelope {
  return { status: "ok", requestId: "task-255-session", data: { userId: "admin-255", role: "admin", hasVerifiedLoginMethod: true, accessExpiresAt: "2026-07-21T22:00:00Z", refreshExpiresAt: "2026-07-28T22:00:00Z" } };
}

function profileEnvelope(): ProfileEnvelope {
  return { status: "ok", requestId: "task-255-profile", data: { userId: "admin-255", displayName: "Admin", unitSystem: "metric", themePreference: "system", requiresUnitRecalculation: false } };
}

function entitlementEnvelope(): EntitlementStatusEnvelope {
  return { status: "ok", requestId: "task-255-entitlement", data: { userId: "admin-255", tier: "paid", status: "active", allowedModes: ["catalog", "substitution", "daily_diet", "daily_diet_alternative"], searchLimitPer24h: null, usageUsed: 0, usageRemaining: null, usageWindowStartedAt: "2026-07-21T00:00:00Z", trialExpiresAt: null, billingRecoveryState: "none" } };
}

async function json(route: Route, status: number, body: unknown, headers?: Record<string, string>): Promise<void> {
  await route.fulfill({ status, contentType: "application/json", headers, body: JSON.stringify(body) });
}

async function stubAdminShell(page: Page): Promise<void> {
  await page.route(/\/api\/v1\/search-history$/, (route) => json(route, 200, { status: "ok", requestId: "history", data: { history: [] } }));
  await page.route(/\/api\/v1\/saved-items\?kind=favorite$/, (route) => json(route, 200, { status: "ok", requestId: "favorites", data: { items: [] } }));
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, (route) => json(route, 200, { status: "ok", requestId: "autocomplete", data: { items: [] } }));
  await page.route(/\/api\/v1\/profile$/, (route) => json(route, 200, profileEnvelope()));
  await page.route(/\/api\/v1\/auth\/refresh$/, (route) => json(route, 200, sessionEnvelope()));
  await page.route(/\/api\/v1\/auth\/csrf-token$/, (route) => json(route, 200, { status: "ok", requestId: "csrf", data: { csrfToken: "csrf-255" } }));
  await page.route(/\/api\/v1\/billing\/entitlement$/, (route) => json(route, 200, entitlementEnvelope()));
  await page.route(/\/api\/v1\/admin\/users(\?.*)?$/, (route) => json(route, 200, { status: "ok", requestId: "users", data: { users: [] } }));
  await page.route(/\/api\/v1\/admin\/classifications\?kind=(food_category|culinary_role)$/, (route) => {
    const kind = new URL(route.request().url()).searchParams.get("kind") as "food_category" | "culinary_role";
    const classification = kind === "food_category"
      ? { id: fruitId, name: "Fruit", kind }
      : { id: drinkId, name: "Drink", kind };
    return json(route, 200, { status: "ok", requestId: `classes-${kind}`, data: { classifications: [classification] } });
  });
}

function externalEnvelope(provider: "usda" | "openfoodfacts" | "all", page: number): ExternalSearchEnvelope {
  const selected = { ...candidate, provider: provider === "openfoodfacts" ? "openfoodfacts" as const : "usda" as const, externalId: `${provider}-${page}` };
  return {
    status: "ok",
    requestId: `external-${provider}-${page}`,
    data: {
      candidates: [selected],
      warnings: provider === "all" ? [{ provider: "openfoodfacts", code: "timeout", message: "timeout" }] : [],
      page
    }
  };
}

function lifecycleEnvelope(query: string): ExternalSearchEnvelope {
  const current = {
    ...candidate,
    externalId: "usda-current-266",
    name: query === "replacement" ? "Replacement candidate" : "Current lifecycle candidate",
    physicalState: "solid" as const,
    warnings: [] as ExternalCandidate["warnings"]
  };
  return {
    ...externalEnvelope("usda", 1),
    data: { candidates: [{ ...candidate, name: "Deferred lifecycle candidate", warnings: [] }, current], warnings: [], page: 1 }
  };
}

function importEnvelope(replayed = false, merged = false): CuratedImportEnvelope {
  return { status: "ok", requestId: "import-255", data: { importId, foodItemId, name: "Curated Apple Drink", physicalState: "liquid", merged, replayed } };
}

function localSearchEnvelope(): SearchResponseEnvelope {
  return { status: "ok", requestId: "local-255", data: { items: [{ id: foodItemId, objectType: "food_item", name: "Curated Apple Drink", physicalState: "liquid", imageUrl: null, classifications: [{ id: fruitId, name: "Fruit", kind: "food_category" }], primaryFoodCategory: { id: fruitId, name: "Fruit", kind: "food_category" }, macros: { protein: 1, carbohydrates: 10, fat: 0 }, macroBasis: "100ml", calories: 44 }], totalCount: 1, page: 1, similarityScores: [1], similarityMetadata: [{ itemId: foodItemId, score: 1, tier: "excellent", imageUrl: "", matchingQuantity: 100 }], warnings: [] } };
}

// Verifies IT-ARCH-009-002, IT-ARCH-009-003, IT-ARCH-012-001,
// IT-ARCH-012-002, ARCH-009, ARCH-012, DESIGN-009 ExternalSearchProxy/DataImporter,
// DESIGN-012 DataNormalizer, and SW-REQ-055/SW-REQ-090.
test("searches every provider, paginates partial results, curates warnings/classifications, confirms conflict, and opens the local result", async ({ page }) => {
  await stubAdminShell(page);
  const searches: string[] = [];
  await page.route(/\/api\/v1\/admin\/external-search\?.*$/, (route) => {
    const url = new URL(route.request().url());
    const provider = url.searchParams.get("provider") as "usda" | "openfoodfacts" | "all";
    const resultPage = Number(url.searchParams.get("page"));
    searches.push(`${provider}:${resultPage}`);
    return json(route, 200, externalEnvelope(provider, resultPage));
  });
  const importKeys: string[] = [];
  const importBodies: unknown[] = [];
  await page.route(/\/api\/v1\/admin\/imports$/, async (route) => {
    importKeys.push(route.request().headers()["idempotency-key"] ?? "");
    const body = route.request().postDataJSON();
    importBodies.push(body);
    const validLiquid = body.physicalState === "liquid" && body.densityGramsPerMilliliter === 1.02 && body.densitySourceKind === "manual" && body.densitySourceProvider === undefined && body.densitySourceFoodId === undefined;
    if (!validLiquid) return json(route, 422, { status: "error", requestId: "invalid", error: { category: "validation", code: "validation_failed", message: "invalid density provenance", retryable: false } });
    if (importKeys.length === 1) return json(route, 409, { status: "error", requestId: "conflict", error: { category: "validation", code: "name_conflict_confirmation_required", message: "RAW duplicate SQL diagnostics", retryable: false } });
    return json(route, 201, importEnvelope(false, true));
  });
  await page.route(/\/api\/v1\/search$/, (route) => json(route, 200, localSearchEnvelope()));

  await page.goto("/admin");
  const workflow = page.locator("[data-external-import-workflow]");
  await expect(workflow).toBeVisible();
  await workflow.getByLabel("External food search").fill("apple drink");

  await workflow.getByRole("button", { name: "Search", exact: true }).click();
  await expect(workflow.locator("[data-external-results]")).toBeVisible();
  expect(searches.at(-1)).toBe("all:1");
  await expect(workflow.locator("[data-provider-warnings]")).toContainText("timed out");
  await workflow.getByRole("button", { name: "Next" }).click();
  expect(searches.at(-1)).toBe("all:2");

  await workflow.getByRole("button", { name: "Curate" }).click();
  const draft = workflow.locator("[data-curation-draft]");
  await expect(draft.locator("[data-candidate-warnings]")).toContainText("Liquid density is missing");
  await expect(draft.locator("[data-candidate-warnings]")).toContainText("unit conversion");
  await expect(draft.locator("[data-candidate-warnings]")).toContainText("unexpected basis");
  await draft.getByLabel("Name").fill("Curated Apple Drink");
  await draft.getByLabel("Protein per 100").fill("1");
  await draft.getByLabel("Density (g/ml)").fill("1.02");
  await expect(draft.getByLabel("Density provenance")).toHaveValue("manual");
  await expect(draft.locator("[data-density-curation-state]")).toContainText("provenance supplied");
  await expect(draft.locator("[data-candidate-warnings]")).not.toContainText("Liquid density is missing");
  await draft.getByLabel("Fruit").check();
  await draft.getByLabel("Drink").check();
  await draft.getByRole("button", { name: "Import curated item" }).click();
  await expect(workflow.locator("[data-import-conflict]")).toContainText("matching local item");
  await expect(workflow).not.toContainText("RAW duplicate");
  await workflow.getByRole("button", { name: "Confirm merge" }).click();
  await expect(workflow.locator("[data-import-result]")).toContainText("Curated Apple Drink");
  expect(importKeys[0]).toBeTruthy();
  expect(importKeys[1]).toBe(importKeys[0]);
  expect(importBodies).toMatchObject([{ confirmNameConflict: false }, { confirmNameConflict: true }]);
  expect(importBodies[0]).toMatchObject({ densityGramsPerMilliliter: 1.02, densitySourceKind: "manual", foodCategoryIds: [fruitId], culinaryRoleIds: [drinkId] });

  const viewLocal = workflow.getByRole("button", { name: "View in local search" });
  await viewLocal.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByLabel("Food search")).toHaveValue("Curated Apple Drink");
  await expect(page.getByText("Curated Apple Drink", { exact: true }).last()).toBeVisible();
  const axe = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
  expect(axe.violations.filter((violation) => violation.impact === "serious" || violation.impact === "critical")).toEqual([]);
});

test("keeps provider, idempotency, and unknown conflicts out of the merge-confirmation path", async ({ page }) => {
  await stubAdminShell(page);
  await page.route(/\/api\/v1\/admin\/external-search\?.*$/, (route) => json(route, 200, externalEnvelope("usda", 1)));
  let conflictCode = "provider_identity_conflict";
  const bodies: Array<Record<string, unknown>> = [];
  const keys: string[] = [];
  await page.route(/\/api\/v1\/admin\/imports$/, (route) => {
    bodies.push(route.request().postDataJSON());
    keys.push(route.request().headers()["idempotency-key"] ?? "");
    return json(route, 409, { status: "error", requestId: "conflict", error: { category: "validation", code: conflictCode, message: "RAW conflict details", retryable: false } });
  });
  await page.goto("/admin");
  const workflow = page.locator("[data-external-import-workflow]");
  await workflow.getByLabel("External food search").fill("apple");
  await workflow.getByRole("button", { name: "Search", exact: true }).click();
  await workflow.getByRole("button", { name: "Curate" }).click();
  const draft = workflow.locator("[data-curation-draft]");
  await draft.getByLabel("Density (g/ml)").fill("1.01");

  for (const [code, copy] of [
    ["provider_identity_conflict", "provider item"],
    ["idempotency_key_conflict", "import attempt"],
    ["unknown_conflict", "conflicts with existing data"]
  ] as const) {
    conflictCode = code;
    await draft.getByRole("button", { name: "Import curated item" }).click();
    const blocked = workflow.locator("[data-import-blocked-conflict]");
    await expect(blocked).toContainText(copy);
    await expect(blocked.getByRole("button", { name: "Confirm merge" })).toHaveCount(0);
    await expect(workflow).not.toContainText("RAW conflict");
    if (code === "provider_identity_conflict") await expect(blocked.getByRole("button", { name: "Refresh external results" })).toBeVisible();
    if (code === "idempotency_key_conflict") {
      await blocked.getByRole("button", { name: "Start a fresh import attempt" }).click();
      await expect.poll(() => bodies.length).toBe(3);
      expect(keys[2]).not.toBe(keys[1]);
      await expect(blocked).toBeVisible();
    }
    await blocked.getByRole("button", { name: "Keep editing" }).click();
  }
  expect(bodies).toHaveLength(4);
  expect(bodies.every((body) => body.confirmNameConflict === false)).toBe(true);
});

test("contains malformed nested search and import payloads without crashing the workflow", async ({ page }) => {
  await stubAdminShell(page);
  let validSearch = false;
  await page.route(/\/api\/v1\/admin\/external-search\?.*$/, (route) => json(route, 200, {
    status: "ok",
    requestId: "malformed-search",
    data: validSearch ? externalEnvelope("usda", 1).data : { candidates: [{ ...candidate, warnings: null }], warnings: [], page: 1 }
  }));
  await page.route(/\/api\/v1\/admin\/imports$/, (route) => json(route, 201, {
    status: "ok",
    requestId: "malformed-import",
    data: { ...importEnvelope().data, merged: "yes" }
  }));
  await page.goto("/admin");
  const workflow = page.locator("[data-external-import-workflow]");
  await workflow.getByLabel("External food search").fill("apple");
  await workflow.getByRole("button", { name: "Search", exact: true }).click();
  await expect(workflow.locator("[data-external-error]")).toContainText("unexpected response");
  await expect(workflow).toBeVisible();

  validSearch = true;
  await workflow.getByRole("button", { name: "Retry search" }).click();
  await workflow.getByRole("button", { name: "Curate" }).click();
  await workflow.getByLabel("Density (g/ml)").fill("1.01");
  await workflow.getByRole("button", { name: "Import curated item" }).click();
  await expect(workflow.locator("[data-import-error]")).toContainText("unexpected response");
  await expect(workflow).toBeVisible();
});

// Verifies IT-ARCH-012-003, ARCH-012, DESIGN-012 RateLimitHandler, and SW-REQ-055.
test("ignores a stale external search response after a newer query wins", async ({ page }) => {
  await stubAdminShell(page);
  let releaseFirst!: () => void;
  const firstPending = new Promise<void>((resolve) => { releaseFirst = resolve; });
  await page.route(/\/api\/v1\/admin\/external-search\?.*$/, async (route) => {
    const query = new URL(route.request().url()).searchParams.get("query");
    if (query === "first") {
      await firstPending;
      await json(route, 200, { ...externalEnvelope("usda", 1), data: { ...externalEnvelope("usda", 1).data, candidates: [{ ...candidate, name: "Stale candidate" }] } }).catch(() => undefined);
      return;
    }
    await json(route, 200, { ...externalEnvelope("usda", 1), data: { ...externalEnvelope("usda", 1).data, candidates: [{ ...candidate, name: "Current candidate" }] } });
  });
  await page.goto("/admin");
  const workflow = page.locator("[data-external-import-workflow]");
  const search = workflow.getByLabel("External food search");
  await search.fill("first");
  await workflow.getByRole("button", { name: "Search", exact: true }).click();
  await expect(workflow.locator("[data-external-loading]")).toBeVisible();
  await search.fill("second");
  await search.press("Enter");
  await expect(workflow.getByText("Current candidate")).toBeVisible();
  releaseFirst();
  await expect(workflow.getByText("Stale candidate")).toHaveCount(0);
});

// Verifies IT-ARCH-009-010 and IT-ARCH-012-004, ARCH-009, ARCH-012,
// DESIGN-009 ExternalSearchProxy/DataImporter, DESIGN-012 RateLimitHandler,
// and SW-REQ-055.
for (const outcome of ["success", "conflict", "ambiguity", "failure"] as const) {
  test(`ignores a superseded import ${outcome} without overwriting the active draft`, async ({ page }) => {
    await stubAdminShell(page);
    await page.route(/\/api\/v1\/admin\/external-search\?.*$/, (route) => json(route, 200, lifecycleEnvelope("initial")));
    let releaseImport!: () => void;
    let markImportStarted!: () => void;
    const importRelease = new Promise<void>((resolve) => { releaseImport = resolve; });
    const importStarted = new Promise<void>((resolve) => { markImportStarted = resolve; });
    await page.route(/\/api\/v1\/admin\/imports$/, async (route) => {
      markImportStarted();
      await importRelease;
      if (outcome === "success") return json(route, 201, importEnvelope());
      if (outcome === "conflict") return json(route, 409, { status: "error", requestId: "conflict-266", error: { category: "validation", code: "name_conflict_confirmation_required", message: "superseded conflict", retryable: false } });
      if (outcome === "ambiguity") return route.abort("connectionreset");
      return json(route, 503, { status: "error", requestId: "failure-266", error: { category: "dependency", code: "external_source_unavailable", message: "superseded failure", retryable: true } });
    });
    await page.goto("/admin");
    const workflow = page.locator("[data-external-import-workflow]");
    await workflow.getByLabel("External food search").fill("initial");
    await workflow.getByRole("button", { name: "Search", exact: true }).click();
    await workflow.getByRole("article").filter({ hasText: "Deferred lifecycle candidate" }).getByRole("button", { name: "Curate" }).click();
    await workflow.getByLabel("Density (g/ml)").fill("1.01");
    await workflow.getByRole("button", { name: "Import curated item" }).click();
    await importStarted;

    const currentButton = workflow.getByRole("article").filter({ hasText: "Current lifecycle candidate" }).getByRole("button", { name: "Curate" });
    await expect(currentButton).toBeDisabled();
    await currentButton.evaluate((button) => {
      (button as HTMLButtonElement).disabled = false;
      (button as HTMLButtonElement).click();
    });
    await expect(workflow.getByLabel("Name")).toHaveValue("Current lifecycle candidate");

    const completion = outcome === "ambiguity"
      ? page.waitForEvent("requestfailed", (request) => request.url().endsWith("/api/v1/admin/imports"))
      : page.waitForResponse((response) => response.url().endsWith("/api/v1/admin/imports"));
    releaseImport();
    await completion;
    await page.evaluate(() => new Promise<void>((resolve) => setTimeout(resolve, 0)));

    await expect(workflow.getByLabel("Name")).toHaveValue("Current lifecycle candidate");
    await expect(workflow.locator("[data-import-result], [data-import-conflict], [data-import-blocked-conflict], [data-import-error]")).toHaveCount(0);
    await expect(workflow.getByRole("button", { name: "Import curated item" })).toBeEnabled();
    await expect(workflow).not.toContainText("superseded");
  });
}

// Verifies IT-ARCH-009-010 and IT-ARCH-012-004, ARCH-009, ARCH-012,
// DESIGN-009 ExternalSearchProxy/DataImporter, and SW-REQ-055.
test("disables incompatible controls during import and starts a completed workflow without a draft warning", async ({ page }) => {
  await stubAdminShell(page);
  await page.route(/\/api\/v1\/admin\/external-search\?.*$/, (route) => {
    const query = new URL(route.request().url()).searchParams.get("query") ?? "";
    return json(route, 200, lifecycleEnvelope(query));
  });
  let releaseImport!: () => void;
  let markImportStarted!: () => void;
  const importRelease = new Promise<void>((resolve) => { releaseImport = resolve; });
  const importStarted = new Promise<void>((resolve) => { markImportStarted = resolve; });
  await page.route(/\/api\/v1\/admin\/imports$/, async (route) => {
    markImportStarted();
    await importRelease;
    return json(route, 201, importEnvelope());
  });
  await page.goto("/admin");
  const workflow = page.locator("[data-external-import-workflow]");
  await workflow.getByLabel("External food search").fill("initial");
  await workflow.getByRole("button", { name: "Search", exact: true }).click();
  await workflow.getByRole("article").filter({ hasText: "Deferred lifecycle candidate" }).getByRole("button", { name: "Curate" }).click();
  const draft = workflow.locator("[data-curation-draft]");
  await draft.getByLabel("Density (g/ml)").fill("1.01");
  await draft.getByRole("button", { name: "Import curated item" }).click();
  await importStarted;

  await expect(workflow.getByLabel("External food search")).toBeDisabled();
  await expect(workflow.getByRole("button", { name: "Search", exact: true })).toBeDisabled();
  await expect(workflow.getByRole("button", { name: "Next" })).toBeDisabled();
  await expect(workflow.getByRole("button", { name: "Curate" }).first()).toBeDisabled();
  await expect(draft.getByLabel("Name")).toBeDisabled();
  await expect(draft.getByLabel("Physical state")).toBeDisabled();
  await expect(draft.getByLabel("Density (g/ml)")).toBeDisabled();
  await expect(draft.getByLabel("Fruit")).toBeDisabled();
  await expect(draft.getByRole("button", { name: "Importing…" })).toBeDisabled();

  const completion = page.waitForResponse((response) => response.url().endsWith("/api/v1/admin/imports"));
  releaseImport();
  await completion;
  await expect(workflow.locator("[data-import-result]")).toBeVisible();
  await expect(draft).toHaveCount(0);

  await workflow.getByLabel("External food search").fill("replacement");
  await workflow.getByRole("button", { name: "Search", exact: true }).click();
  await expect(workflow.locator("[data-draft-search-boundary]")).toHaveCount(0);
  await expect(workflow.locator("[data-import-result]")).toHaveCount(0);
  await expect(workflow.getByText("Replacement candidate")).toBeVisible();
});

// Verifies IT-ARCH-009-010 and IT-ARCH-012-004, ARCH-009, ARCH-012,
// DESIGN-009 ExternalSearchProxy/DataImporter, and SW-REQ-055.
test("keeps or discards an unsaved draft explicitly and invalidates discarded import ownership", async ({ page }) => {
  await stubAdminShell(page);
  let searches = 0;
  await page.route(/\/api\/v1\/admin\/external-search\?.*$/, (route) => {
    searches += 1;
    const query = new URL(route.request().url()).searchParams.get("query") ?? "";
    return json(route, 200, lifecycleEnvelope(query));
  });
  let releaseImport!: () => void;
  let markImportStarted!: () => void;
  const importRelease = new Promise<void>((resolve) => { releaseImport = resolve; });
  const importStarted = new Promise<void>((resolve) => { markImportStarted = resolve; });
  await page.route(/\/api\/v1\/admin\/imports$/, async (route) => {
    markImportStarted();
    await importRelease;
    return json(route, 201, importEnvelope());
  });
  await page.goto("/admin");
  const workflow = page.locator("[data-external-import-workflow]");
  const search = workflow.getByLabel("External food search");
  await search.fill("initial");
  await workflow.getByRole("button", { name: "Search", exact: true }).click();
  await workflow.getByRole("article").filter({ hasText: "Deferred lifecycle candidate" }).getByRole("button", { name: "Curate" }).click();
  const draft = workflow.locator("[data-curation-draft]");
  await draft.getByLabel("Name").fill("Unsaved administrator draft");
  await draft.getByLabel("Density (g/ml)").fill("1.01");
  await search.fill("replacement");
  await workflow.getByRole("button", { name: "Search", exact: true }).click();

  const boundary = workflow.locator("[data-draft-search-boundary]");
  const boundaryBackground = workflow.locator("[data-draft-search-background]");
  const backgroundCandidate = workflow.getByRole("article").filter({ hasText: "Current lifecycle candidate" }).getByRole("button", { name: "Curate" });
  await expect(boundary).toHaveAttribute("role", "alertdialog");
  await expect(boundary).toHaveAttribute("aria-modal", "true");
  await expect(boundaryBackground).toHaveAttribute("inert", "");
  await expect(boundary.getByRole("button", { name: "Keep editing" })).toBeFocused();
  let backgroundActivationBlocked = false;
  try {
    await backgroundCandidate.click({ timeout: 750 });
  } catch {
    backgroundActivationBlocked = true;
  }
  expect(backgroundActivationBlocked).toBe(true);
  await expect(boundary).toBeVisible();
  await expect(draft.getByLabel("Name")).toHaveValue("Unsaved administrator draft");
  await search.evaluate((input) => input.focus());
  await expect(boundary.getByRole("button", { name: "Keep editing" })).toBeFocused();
  await page.keyboard.press("Shift+Tab");
  await expect(boundary.getByRole("button", { name: "Discard draft and search" })).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(boundary.getByRole("button", { name: "Keep editing" })).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(boundary).toHaveCount(0);
  await expect(draft.getByLabel("Name")).toHaveValue("Unsaved administrator draft");
  await expect(draft.getByLabel("Name")).toBeFocused();
  expect(searches).toBe(1);

  await workflow.getByRole("button", { name: "Search", exact: true }).click();
  await expect(boundary.getByRole("button", { name: "Keep editing" })).toBeFocused();
  await boundary.getByRole("button", { name: "Keep editing" }).press("Enter");
  await expect(draft.getByLabel("Name")).toHaveValue("Unsaved administrator draft");
  await expect(draft.getByLabel("Name")).toBeFocused();
  expect(searches).toBe(1);

  await workflow.getByRole("button", { name: "Search", exact: true }).click();
  await draft.evaluate((form) => form.dispatchEvent(new SubmitEvent("submit", { bubbles: true, cancelable: true })));
  await importStarted;
  const discard = boundary.getByRole("button", { name: "Discard draft and search" });
  await discard.focus();
  await discard.press("Enter");
  await expect(workflow.getByText("Replacement candidate")).toBeVisible();
  await expect(draft).toHaveCount(0);
  expect(searches).toBe(2);

  const completion = page.waitForResponse((response) => response.url().endsWith("/api/v1/admin/imports"));
  releaseImport();
  await completion;
  await page.evaluate(() => new Promise<void>((resolve) => setTimeout(resolve, 0)));
  await expect(workflow.locator("[data-import-result], [data-import-conflict], [data-import-blocked-conflict], [data-import-error]")).toHaveCount(0);
  await expect(workflow.getByText("Replacement candidate")).toBeVisible();
  const axe = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
  expect(axe.violations.filter((violation) => violation.impact === "serious" || violation.impact === "critical")).toEqual([]);
});

// Verifies IT-ARCH-009-010 and IT-ARCH-012-004, ARCH-009, ARCH-012,
// DESIGN-009 UserAdminPanel/ExternalSearchProxy, and SW-REQ-054/SW-REQ-055.
test("restores visible focus after discarding from pagination or a disappearing refresh action", async ({ page }) => {
  await stubAdminShell(page);
  await page.route(/\/api\/v1\/admin\/external-search\?.*$/, (route) => {
    const url = new URL(route.request().url());
    const result = lifecycleEnvelope(url.searchParams.get("query") ?? "");
    result.data.page = Number(url.searchParams.get("page"));
    return json(route, 200, result);
  });
  await page.route(/\/api\/v1\/admin\/imports$/, (route) => json(route, 409, {
    status: "error",
    requestId: "provider-conflict",
    error: { category: "conflict", code: "provider_identity_conflict", message: "Refresh provider data.", retryable: false }
  }));
  await page.goto("/admin");

  const workflow = page.locator("[data-external-import-workflow]");
  const search = workflow.getByLabel("External food search");
  const boundary = workflow.locator("[data-draft-search-boundary]");
  await search.fill("initial");
  await workflow.getByRole("button", { name: "Search", exact: true }).click();
  await workflow.getByRole("article").filter({ hasText: "Deferred lifecycle candidate" }).getByRole("button", { name: "Curate" }).click();

  await workflow.getByRole("button", { name: "Next" }).click();
  await boundary.getByRole("button", { name: "Discard draft and search" }).click();
  await expect(workflow.getByText("Page 2")).toBeVisible();
  await expect(search).toBeFocused();

  await workflow.getByRole("article").filter({ hasText: "Deferred lifecycle candidate" }).getByRole("button", { name: "Curate" }).click();
  const draft = workflow.locator("[data-curation-draft]");
  await draft.getByLabel("Density (g/ml)").fill("1");
  await draft.getByRole("button", { name: "Import curated item" }).click();
  await workflow.getByRole("button", { name: "Refresh external results" }).click();
  await boundary.getByRole("button", { name: "Discard draft and search" }).click();
  await expect(workflow.locator("[data-external-results]")).toBeVisible();
  await expect(search).toBeFocused();
});

// Verifies IT-ARCH-012-003, ARCH-012, DESIGN-012 RateLimitHandler, and SW-REQ-055.
test("shows loading, empty, rate-limit, timeout, and unavailable states without raw diagnostics", async ({ page }) => {
  await stubAdminShell(page);
  let release!: () => void;
  let mode: "loading" | 429 | 504 | 503 = "loading";
  const pending = new Promise<void>((resolve) => { release = resolve; });
  await page.route(/\/api\/v1\/admin\/external-search\?.*$/, async (route) => {
    if (mode === "loading") {
      await pending;
      return json(route, 200, { status: "ok", requestId: "empty", data: { candidates: [], warnings: [], page: 1 } });
    }
    return json(route, mode, { status: "error", requestId: "safe", error: { category: "unknown", code: "provider-secret-code", message: "RAW provider host and stack", retryable: true } }, mode === 429 ? { "Retry-After": "9" } : undefined);
  });
  await page.goto("/admin");
  const workflow = page.locator("[data-external-import-workflow]");
  await workflow.getByLabel("External food search").fill("nothing");
  await workflow.getByRole("button", { name: "Search", exact: true }).click();
  await expect(workflow.locator("[data-external-loading]")).toBeVisible();
  release();
  await expect(workflow.locator("[data-external-empty]")).toBeVisible();

  for (const [status, message] of [[429, "rate limited"], [504, "timed out"], [503, "temporarily unavailable"]] as const) {
    mode = status;
    await workflow.getByRole("button", { name: "Search", exact: true }).click();
    await expect(workflow.locator("[data-external-error]")).toContainText(message);
    await expect(workflow).not.toContainText("RAW provider");
    await expect(workflow).not.toContainText("provider-secret-code");
  }
});

test("distinguishes provider failure, rejected candidates, and genuine zero matches", async ({ page }) => {
  await stubAdminShell(page);
  let warnings: ExternalSearchEnvelope["data"]["warnings"] = [];
  await page.route(/\/api\/v1\/admin\/external-search\?.*$/, (route) => json(route, 200, {
    status: "ok",
    requestId: "bounded-empty-state",
    data: { candidates: [], warnings, page: 1 }
  }));
  await page.goto("/admin");
  const workflow = page.locator("[data-external-import-workflow]");
  await workflow.getByLabel("External food search").fill("nothing");

  await workflow.getByRole("button", { name: "Search", exact: true }).click();
  await expect(workflow.locator("[data-external-empty]")).toHaveText("No external candidates matched this search.");

  warnings = [{ provider: "openfoodfacts", code: "invalid_external_payload", message: "invalid_external_payload" }];
  await workflow.getByRole("button", { name: "Search", exact: true }).click();
  await expect(workflow.locator("[data-external-empty]")).toHaveText("Provider candidates were rejected because their data could not be used.");

  warnings = [{ provider: "openfoodfacts", code: "provider_unavailable", message: "provider_unavailable" }];
  await workflow.getByRole("button", { name: "Search", exact: true }).click();
  await expect(workflow.locator("[data-external-empty]")).toHaveText("External providers could not complete this search.");
});

test("shows safe timeout copy when the search signal times out", async ({ page }) => {
  await stubAdminShell(page);
  await page.goto("/admin");
  await page.evaluate(() => {
    const NativeAbortController = window.AbortController;
    const nativeFetch = window.fetch.bind(window);
    let timeoutController: AbortController | undefined;
    window.AbortController = class extends NativeAbortController {
      constructor() {
        super();
        timeoutController = this;
      }
    };
    window.fetch = async (input, init) => {
      if (String(input).startsWith("/api/v1/admin/external-search?")) {
        timeoutController?.abort(new DOMException("RAW provider deadline", "TimeoutError"));
        throw init?.signal?.reason;
      }
      return nativeFetch(input, init);
    };
  });

  const workflow = page.locator("[data-external-import-workflow]");
  await workflow.getByLabel("External food search").fill("timeout apple");
  await workflow.getByRole("button", { name: "Search", exact: true }).click();
  await expect(workflow.locator("[data-external-error]")).toContainText("The request timed out. Try again.");
  await expect(workflow).not.toContainText("RAW provider deadline");
});

// Verifies IT-ARCH-009-003, ARCH-009, DESIGN-009 DataImporter, and SW-REQ-055.
test("replays one ambiguous import with the same key and displays one local item identity", async ({ page }) => {
  await stubAdminShell(page);
  await page.route(/\/api\/v1\/admin\/external-search\?.*$/, (route) => json(route, 200, externalEnvelope("usda", 1)));
  const keys: string[] = [];
  let calls = 0;
  await page.route(/\/api\/v1\/admin\/imports$/, async (route) => {
    keys.push(route.request().headers()["idempotency-key"] ?? "");
    calls += 1;
    if (calls === 1) return route.abort("connectionreset");
    return json(route, 201, importEnvelope(true, false));
  });
  await page.goto("/admin");
  const workflow = page.locator("[data-external-import-workflow]");
  await workflow.getByLabel("External food search").fill("apple");
  await workflow.getByRole("button", { name: "Search", exact: true }).click();
  await workflow.getByRole("button", { name: "Curate" }).click();
  await workflow.getByLabel("Density (g/ml)").fill("1.01");
  await workflow.getByRole("button", { name: "Import curated item" }).click();
  await expect(workflow.locator("[data-import-error]")).toContainText("could not be confirmed");
  await expect(workflow).not.toContainText("connectionreset");
  await workflow.getByRole("button", { name: "Retry import safely" }).click();
  await expect(workflow.locator("[data-import-result]")).toContainText("confirmed retry");
  await expect(workflow.locator("[data-import-result]")).toContainText("Curated Apple Drink");
  expect(keys).toHaveLength(2);
  expect(keys[1]).toBe(keys[0]);
});
