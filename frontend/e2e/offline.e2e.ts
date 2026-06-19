import { expect, test } from "./fixtures";

// Implements DESIGN-001 OfflineBanner cached, uncached, and reconnect browser verification.
test("offline searches use cache and uncached searches recover after reconnect", async ({ controlledPage, context }) => {
  await controlledPage.goto("/");
  const search = controlledPage.getByRole("textbox", { name: "Food search" });
  await search.fill("apple");
  await controlledPage.getByRole("button", { name: "Search", exact: true }).click();
  await expect(controlledPage.getByTestId("result-card")).toHaveCount(10);
  await context.setOffline(true);
  await controlledPage.getByRole("button", { name: "Search", exact: true }).click();
  await expect(controlledPage.getByRole("status").filter({ hasText: "Cached results" })).toBeVisible();
  await search.fill("uncached");
  await controlledPage.getByRole("button", { name: "Search", exact: true }).click();
  await expect(controlledPage.getByRole("status").filter({ hasText: "not cached" })).toBeVisible();
  await context.setOffline(false);
  await controlledPage.getByRole("button", { name: "Search", exact: true }).click();
  await expect(controlledPage.getByTestId("result-card")).toHaveCount(10);
});

// Implements DESIGN-001 OfflineBanner stale cached-result browser verification.
test("stale cached results remain visible while offline", async ({ controlledPage, context }) => {
  await controlledPage.goto("/");
  const search = controlledPage.getByRole("textbox", { name: "Food search" });
  await search.fill("apple");
  await controlledPage.getByRole("button", { name: "Search", exact: true }).click();
  await expect(controlledPage.getByTestId("result-card")).toHaveCount(10);

  await controlledPage.evaluate(() => {
    const key = "mealswapp.search-cache.v1";
    const cache = JSON.parse(localStorage.getItem(key) ?? "null") as { entries?: Array<{ staleAt: string }> } | null;
    if (!cache?.entries?.length) throw new Error("expected a primed search cache");
    for (const entry of cache.entries) entry.staleAt = "2000-01-01T00:00:00.000Z";
    localStorage.setItem(key, JSON.stringify(cache));
  });

  await controlledPage.reload();
  await search.fill("apple");
  await context.setOffline(true);
  await controlledPage.getByRole("button", { name: "Search", exact: true }).click();

  await expect(controlledPage.getByTestId("result-card")).toHaveCount(10);
  await expect(controlledPage.getByRole("status").filter({ hasText: "Cached stale results are shown" })).toBeVisible();
});
