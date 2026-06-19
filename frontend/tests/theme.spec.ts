import { expect, test, type Page } from "@playwright/test";

// Implements DESIGN-016 ThemeProvider browser persistence and live system-theme subscription.
//
// The theme `<select>` already lives in SearchShell.svelte (Phase 00 Task 4 wired it to
// `setThemePreference`), so these flows exercise the real running app. Task 151 will
// extend the sidebar/shell theme selector surface; the cases below stay valid because
// they target the existing control and the document-root `data-theme` token.

/** Stubs the autocomplete and search endpoints so the SearchShell renders without a backend. */
async function stubApi(page: Page): Promise<void> {
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ status: "ok", requestId: "theme-stub", data: { items: [] } })
    });
  });
  await page.route(/\/api\/v1\/search$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        status: "ok",
        requestId: "theme-stub",
        data: { items: [], totalCount: 0, page: 1, similarityScores: [], similarityMetadata: [], warnings: [] }
      })
    });
  });
}

// Implements DESIGN-016 ThemeProvider explicit override persistence across reload verification.
test("explicit dark theme selection persists across a reload", async ({ page }) => {
  await stubApi(page);
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/");

  await page.getByLabel("Theme preference").selectOption("dark");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");

  await page.reload();

  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.getByLabel("Theme preference")).toHaveValue("dark");
});

// Implements DESIGN-016 ThemeProvider explicit light override ignores system changes verification.
test("explicit light override ignores a live switch to system dark", async ({ page }) => {
  await stubApi(page);
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/");

  await page.getByLabel("Theme preference").selectOption("light");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");

  // System flips to dark while the user explicitly chose light; the resolved theme stays light.
  await page.emulateMedia({ colorScheme: "dark" });

  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
});

// Implements DESIGN-016 ThemeProvider system mode follows live system changes verification.
test("system preference follows a live system theme change", async ({ page }) => {
  await stubApi(page);
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/");

  await page.getByLabel("Theme preference").selectOption("system");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");

  await page.emulateMedia({ colorScheme: "dark" });

  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
});
