import { expect, test, type Page } from "@playwright/test";

// Implements DESIGN-016 ThemeProvider browser persistence and live system-theme subscription.
//
// The binary theme switch lives in SidebarComponent and converts the default system
// preference into an explicit light/dark choice when the user toggles it.

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

/** Sets the resolved document theme through the binary sidebar switch only when needed. */
async function setResolvedTheme(page: Page, target: "light" | "dark"): Promise<void> {
  const current = await page.locator("html").getAttribute("data-theme");
  if (current !== target) {
    const toggle = page.getByLabel("Theme preference");
    const openedSidebar = !(await toggle.isVisible());
    if (openedSidebar) {
      await page.getByLabel("Open activity sidebar").click();
    }
    await toggle.click();
    if (openedSidebar) {
      await page.getByLabel("Close activity sidebar").click();
    }
  }
}

// Implements DESIGN-016 ThemeProvider explicit override persistence across reload verification.
test("explicit dark theme selection persists across a reload", async ({ page }) => {
  await stubApi(page);
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/");

  await setResolvedTheme(page, "dark");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");

  await page.reload();

  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await expect(page.getByLabel("Theme preference")).toHaveAttribute("aria-pressed", "true");
});

// Implements DESIGN-016 ThemeProvider explicit light override ignores system changes verification.
test("explicit light override ignores a live switch to system dark", async ({ page }) => {
  await stubApi(page);
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/");

  const toggle = page.getByLabel("Theme preference");
  const openedSidebar = !(await toggle.isVisible());
  if (openedSidebar) {
    await page.getByLabel("Open activity sidebar").click();
  }
  // Toggle twice: light (system) -> dark -> light (explicit)
  await toggle.click();
  await toggle.click();
  if (openedSidebar) {
    await page.getByLabel("Close activity sidebar").click();
  }

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

  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");

  await page.emulateMedia({ colorScheme: "dark" });

  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
});
