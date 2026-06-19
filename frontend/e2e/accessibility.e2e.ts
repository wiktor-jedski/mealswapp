import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "./fixtures";
import type { Locator } from "@playwright/test";

async function expectNoSeriousViolations(page: import("@playwright/test").Page) {
  const results = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze();
  expect(results.violations.filter((violation) => violation.impact === "serious" || violation.impact === "critical")).toEqual([]);
}

async function expectVisibleFocus(control: Locator) {
  await expect(control).toBeFocused();
  const indicator = await control.evaluate((element) => {
    const styles = getComputedStyle(element);
    return {
      color: styles.outlineColor,
      style: styles.outlineStyle,
      width: Number.parseFloat(styles.outlineWidth)
    };
  });
  expect(indicator.style).not.toBe("none");
  expect(indicator.width).toBeGreaterThanOrEqual(2);
  expect(indicator.color).not.toBe("rgba(0, 0, 0, 0)");
}

// Implements DESIGN-016 ComponentStyles desktop keyboard and axe verification.
test("desktop Catalog workflow is keyboard operable and axe-clean", async ({ controlledPage }, testInfo) => {
  await controlledPage.setViewportSize({ width: 1280, height: 900 });
  await controlledPage.goto("/");
  const query = controlledPage.getByRole("textbox", { name: "Food search" });
  await query.focus();
  await expectVisibleFocus(query);
  await query.fill("apple");
  const search = controlledPage.getByRole("button", { name: "Search", exact: true });
  await search.focus();
  await expectVisibleFocus(search);
  await controlledPage.keyboard.press("Enter");
  await expect(controlledPage.getByTestId("result-card")).toHaveCount(10);
  await expectNoSeriousViolations(controlledPage);
  await controlledPage.screenshot({ path: testInfo.outputPath("desktop-accessibility.png"), fullPage: true });
});

// Implements DESIGN-016 ComponentStyles mobile Substitution keyboard and axe verification.
test("mobile Substitution workflow is keyboard operable and axe-clean", async ({ controlledPage }, testInfo) => {
  await controlledPage.setViewportSize({ width: 390, height: 844 });
  await controlledPage.goto("/");
  const substitution = controlledPage.getByLabel("Search controls").getByRole("button", { name: "Substitution" });
  await substitution.focus();
  await expectVisibleFocus(substitution);
  await controlledPage.keyboard.press("Enter");
  const autocomplete = controlledPage.getByRole("combobox", { name: "Find substitution food" });
  await autocomplete.focus();
  await expectVisibleFocus(autocomplete);
  await autocomplete.fill("app");
  await controlledPage.waitForTimeout(170);
  await autocomplete.press("Enter");
  const search = controlledPage.getByRole("button", { name: "Search", exact: true });
  await search.focus();
  await expectVisibleFocus(search);
  await controlledPage.keyboard.press("Enter");
  await expect(controlledPage.getByTestId("result-card")).toHaveCount(10);
  await expectNoSeriousViolations(controlledPage);
  await controlledPage.screenshot({ path: testInfo.outputPath("mobile-accessibility.png"), fullPage: true });
});
