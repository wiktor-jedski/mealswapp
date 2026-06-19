import { expect, test, type Page } from "@playwright/test";

// Implements DESIGN-016 LayoutGrid, ColorPalette, and TypographySystem responsive browser verification.
//
// Verifies the Phase 05 responsive style system: 12-column desktop grid with a left
// sidebar, single-column layout below 640px, no horizontal scrolling at 320px, Inter
// for UI text, Roboto Mono for data labels, and exact light/dark design tokens from
// docs/requirements/02_STYLE_GUIDE.md. Captures desktop + mobile screenshots in both
// themes for frontend verification. The ResultsGrid/ResultCard components are wired in
// by Task 151; until then the shell's results placeholder exercises the grid container.

const SCREENSHOT_DIR = "test-results/responsive";

/** Normalizes a CSS hex color to a lowercase 6-digit form so `#fff` and `#ffffff` compare equal. */
function normalizeHex(value: string): string {
  const hex = value.trim().toLowerCase().replace(/^#/, "");
  if (hex.length === 3) {
    return `#${hex
      .split("")
      .map((c) => c + c)
      .join("")}`;
  }
  return `#${hex}`;
}

/** Stubs the autocomplete and search endpoints so the SearchShell renders without a backend. */
async function stubApi(page: Page): Promise<void> {
  await page.route(/\/api\/v1\/search\/autocomplete(\?.*)?$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ status: "ok", requestId: "responsive-stub", data: { items: [] } })
    });
  });
  await page.route(/\/api\/v1\/search$/, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        status: "ok",
        requestId: "responsive-stub",
        data: { items: [], totalCount: 0, page: 1, similarityScores: [], similarityMetadata: [], warnings: [] }
      })
    });
  });
}

// Implements DESIGN-016 LayoutGrid no-horizontal-scroll above 320px verification.
test("no horizontal scrollbar at a 320px viewport width", async ({ page }) => {
  await stubApi(page);
  await page.setViewportSize({ width: 320, height: 600 });
  await page.goto("/");
  await expect(page.getByLabel("Food search")).toBeVisible();

  const overflow = await page.evaluate(() => ({
    docScrollWidth: document.documentElement.scrollWidth,
    bodyScrollWidth: document.body.scrollWidth,
    innerWidth: window.innerWidth
  }));
  expect(overflow.docScrollWidth).toBeLessThanOrEqual(overflow.innerWidth);
  expect(overflow.bodyScrollWidth).toBeLessThanOrEqual(overflow.innerWidth);
});

// Implements DESIGN-016 LayoutGrid 12-column desktop grid with sidebar left of main content.
test("desktop layout places the sidebar left of the main content in a 12-column grid", async ({ page }) => {
  await stubApi(page);
  await page.setViewportSize({ width: 1280, height: 720 });
  await page.goto("/");

  const aside = page.locator("main > section > aside");
  const main = page.locator("main > section > div");
  await expect(aside).toBeVisible();
  const asideBox = await aside.boundingBox();
  const mainBox = await main.boundingBox();
  expect(asideBox).not.toBeNull();
  expect(mainBox).not.toBeNull();
  // Sidebar sits to the left of the main content on the same row.
  expect(asideBox!.x).toBeLessThan(mainBox!.x);
  expect(asideBox!.y).toBe(mainBox!.y);

  // The sidebar spans 3 of 12 columns and the main content spans 9 (3 + 9 = 12).
  const asideSpan = await aside.evaluate((el) => getComputedStyle(el).gridColumnEnd);
  const mainSpan = await main.evaluate((el) => getComputedStyle(el).gridColumnEnd);
  expect(asideSpan).toBe("span 3");
  expect(mainSpan).toBe("span 9");
});

// Implements DESIGN-016 LayoutGrid single-column mobile layout with sidebar stacked above main content.
test("mobile layout stacks the sidebar above the main content in a single column", async ({ page }) => {
  await stubApi(page);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");

  const aside = page.locator("main > section > aside");
  const main = page.locator("main > section > div");
  const asideBox = await aside.boundingBox();
  const mainBox = await main.boundingBox();
  expect(asideBox).not.toBeNull();
  expect(mainBox).not.toBeNull();
  // Single column: main content sits below the sidebar at the same horizontal offset.
  expect(asideBox!.x).toBe(mainBox!.x);
  expect(mainBox!.y).toBeGreaterThan(asideBox!.y + asideBox!.height);

  // No explicit column span below the 640px breakpoint; the grid auto-flows a single column.
  const asideSpan = await aside.evaluate((el) => getComputedStyle(el).gridColumnEnd);
  expect(asideSpan).toBe("auto");
});

// Implements DESIGN-016 TypographySystem Inter UI text and Roboto Mono data labels verification.
test("UI text uses Inter and data labels use Roboto Mono", async ({ page }) => {
  await stubApi(page);
  await page.setViewportSize({ width: 1280, height: 720 });
  await page.goto("/");

  const heading = page.getByRole("heading", { name: "Mealswapp", level: 1 });
  const dataLabel = page.locator(".font-data").first();
  const headingFont = await heading.evaluate((el) => getComputedStyle(el).fontFamily);
  const dataFont = await dataLabel.evaluate((el) => getComputedStyle(el).fontFamily);
  expect(headingFont.toLowerCase()).toContain("inter");
  expect(dataFont.toLowerCase()).toContain("roboto mono");
});

// Implements DESIGN-016 ColorPalette exact light and dark design tokens verification.
test("light and dark design tokens match the documented style guide", async ({ page }) => {
  await stubApi(page);
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/");
  await page.getByLabel("Theme preference").selectOption("light");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");

  const lightTokens = await page.evaluate(() => ({
    bg: getComputedStyle(document.documentElement).getPropertyValue("--color-bg"),
    surface: getComputedStyle(document.documentElement).getPropertyValue("--color-surface"),
    text: getComputedStyle(document.documentElement).getPropertyValue("--color-text"),
    muted: getComputedStyle(document.documentElement).getPropertyValue("--color-muted"),
    primary: getComputedStyle(document.documentElement).getPropertyValue("--color-primary"),
    secondary: getComputedStyle(document.documentElement).getPropertyValue("--color-secondary"),
    accent: getComputedStyle(document.documentElement).getPropertyValue("--color-accent"),
    error: getComputedStyle(document.documentElement).getPropertyValue("--color-error")
  }));
  expect(normalizeHex(lightTokens.bg)).toBe("#f7fcf7");
  expect(normalizeHex(lightTokens.surface)).toBe("#ffffff");
  expect(normalizeHex(lightTokens.text)).toBe("#111827");
  expect(normalizeHex(lightTokens.muted)).toBe("#6b7280");
  expect(normalizeHex(lightTokens.primary)).toBe("#166534");
  expect(normalizeHex(lightTokens.secondary)).toBe("#dcfce7");
  expect(normalizeHex(lightTokens.accent)).toBe("#f97316");
  expect(normalizeHex(lightTokens.error)).toBe("#dc2626");

  await page.getByLabel("Theme preference").selectOption("dark");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");

  const darkTokens = await page.evaluate(() => ({
    bg: getComputedStyle(document.documentElement).getPropertyValue("--color-bg"),
    surface: getComputedStyle(document.documentElement).getPropertyValue("--color-surface"),
    text: getComputedStyle(document.documentElement).getPropertyValue("--color-text"),
    muted: getComputedStyle(document.documentElement).getPropertyValue("--color-muted"),
    primary: getComputedStyle(document.documentElement).getPropertyValue("--color-primary"),
    secondary: getComputedStyle(document.documentElement).getPropertyValue("--color-secondary"),
    accent: getComputedStyle(document.documentElement).getPropertyValue("--color-accent"),
    error: getComputedStyle(document.documentElement).getPropertyValue("--color-error")
  }));
  expect(normalizeHex(darkTokens.bg)).toBe("#0a0f0a");
  expect(normalizeHex(darkTokens.surface)).toBe("#161d16");
  expect(normalizeHex(darkTokens.text)).toBe("#f3f4f6");
  expect(normalizeHex(darkTokens.muted)).toBe("#9ca3af");
  expect(normalizeHex(darkTokens.primary)).toBe("#4ade80");
  expect(normalizeHex(darkTokens.secondary)).toBe("#86efac");
  expect(normalizeHex(darkTokens.accent)).toBe("#ffb86c");
  expect(normalizeHex(darkTokens.error)).toBe("#f87171");
});

// Implements DESIGN-016 LayoutGrid and ColorPalette screenshots suitable for frontend verification.
test("captures desktop and mobile screenshots in light and dark themes", async ({ page }) => {
  await stubApi(page);
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/");
  const select = page.getByLabel("Theme preference");

  // Desktop light.
  await page.setViewportSize({ width: 1280, height: 720 });
  await select.selectOption("light");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.screenshot({ path: `${SCREENSHOT_DIR}/responsive-desktop-light.png`, fullPage: true });

  // Desktop dark.
  await select.selectOption("dark");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await page.screenshot({ path: `${SCREENSHOT_DIR}/responsive-desktop-dark.png`, fullPage: true });

  // Mobile light.
  await page.setViewportSize({ width: 390, height: 844 });
  await select.selectOption("light");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "light");
  await page.screenshot({ path: `${SCREENSHOT_DIR}/responsive-mobile-light.png`, fullPage: true });

  // Mobile dark.
  await select.selectOption("dark");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await page.screenshot({ path: `${SCREENSHOT_DIR}/responsive-mobile-dark.png`, fullPage: true });

  expect(true).toBe(true);
});
