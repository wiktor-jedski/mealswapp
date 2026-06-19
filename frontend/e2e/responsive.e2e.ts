import { expect, test } from "./fixtures";
import type { Page } from "@playwright/test";

const lightPalette = {
  "--color-bg": "#f7fcf7",
  "--color-surface": "#ffffff",
  "--color-primary": "#166534",
  "--color-secondary": "#dcfce7",
  "--color-accent": "#f97316",
  "--color-error": "#dc2626",
  "--color-text": "#111827",
  "--color-muted": "#6b7280"
};

const darkPalette = {
  "--color-bg": "#0a0f0a",
  "--color-surface": "#161d16",
  "--color-primary": "#4ade80",
  "--color-secondary": "#86efac",
  "--color-accent": "#ffb86c",
  "--color-error": "#f87171",
  "--color-text": "#f3f4f6",
  "--color-muted": "#9ca3af"
};

async function palette(page: Page) {
  return page.locator(":root").evaluate((element: Element, names: string[]) => {
    const styles = getComputedStyle(element);
    return Object.fromEntries(names.map((name) => [name, styles.getPropertyValue(name).trim()]));
  }, Object.keys(lightPalette));
}

async function searchAndMeasureCards(page: Page) {
  await page.getByRole("textbox", { name: "Food search" }).fill("apple");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  const cards = page.getByTestId("result-card");
  await expect(cards).toHaveCount(10);
  return cards.evaluateAll((elements: Element[]) => elements.map((element) => {
    const bounds = element.getBoundingClientRect();
    return { width: bounds.width, height: bounds.height };
  }));
}

// Implements DESIGN-016 LayoutGrid desktop/mobile, typography, tokens, and screenshot verification.
test("uses a 12-column desktop grid and exact light tokens", async ({ controlledPage }, testInfo) => {
  await controlledPage.setViewportSize({ width: 1280, height: 900 });
  await controlledPage.goto("/");
  const layout = controlledPage.locator("main > section");
  expect(await layout.evaluate((element) => getComputedStyle(element).gridTemplateColumns.split(" ").length)).toBe(12);
  expect(await controlledPage.locator("body").evaluate((element) => getComputedStyle(element).fontFamily)).toContain("Inter");
  expect(await controlledPage.locator(".font-data").first().evaluate((element) => getComputedStyle(element).fontFamily)).toContain("Roboto Mono");
  expect(await palette(controlledPage)).toEqual(lightPalette);
  const cards = await searchAndMeasureCards(controlledPage);
  expect(Math.max(...cards.map(({ width }) => width)) - Math.min(...cards.map(({ width }) => width))).toBeLessThanOrEqual(1);
  expect(Math.max(...cards.map(({ height }) => height)) - Math.min(...cards.map(({ height }) => height))).toBeLessThanOrEqual(1);
  expect(Math.min(...cards.map(({ height }) => height))).toBeGreaterThanOrEqual(288);
  expect(await controlledPage.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(1280);
  await controlledPage.screenshot({ path: testInfo.outputPath("desktop-light.png"), fullPage: true });
});

// Implements DESIGN-016 LayoutGrid single-column mobile and dark-token screenshot verification.
test("uses one column without horizontal overflow at 320px", async ({ controlledPage }, testInfo) => {
  await controlledPage.setViewportSize({ width: 320, height: 720 });
  await controlledPage.emulateMedia({ colorScheme: "dark" });
  await controlledPage.goto("/");
  const layout = controlledPage.locator("main > section");
  expect(await layout.evaluate((element) => getComputedStyle(element).gridTemplateColumns.split(" ").length)).toBe(1);
  expect(await palette(controlledPage)).toEqual(darkPalette);
  const cards = await searchAndMeasureCards(controlledPage);
  expect(Math.max(...cards.map(({ width }) => width)) - Math.min(...cards.map(({ width }) => width))).toBeLessThanOrEqual(1);
  expect(Math.max(...cards.map(({ height }) => height)) - Math.min(...cards.map(({ height }) => height))).toBeLessThanOrEqual(1);
  expect(Math.min(...cards.map(({ height }) => height))).toBeGreaterThanOrEqual(288);
  expect(await controlledPage.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(320);
  await controlledPage.screenshot({ path: testInfo.outputPath("mobile-dark.png"), fullPage: true });
});
