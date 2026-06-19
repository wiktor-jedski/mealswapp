import { expect, test } from "./fixtures";
import type { Locator } from "@playwright/test";

async function expectVisibleFocus(control: Locator) {
  await expect(control).toBeFocused();
  await expect(control).toHaveCSS("outline-style", "solid");
  await expect(control).toHaveCSS("outline-width", "2px");
}

// Implements DESIGN-001 SettingsPanel browser component interaction verification.
test("settings controls are labelled, keyboard operable, and persistent", async ({ controlledPage }) => {
  await controlledPage.goto("/");
  const macros = ["Protein", "Carbohydrate", "Fat"];
  for (const name of macros) {
    const macro = controlledPage.getByRole("checkbox", { name });
    await expect(macro).toBeChecked();
    await macro.focus();
    await expectVisibleFocus(macro);
    await controlledPage.keyboard.press("Space");
    await expect(macro).not.toBeChecked();
  }

  const units = controlledPage.getByRole("combobox", { name: "Unit system" });
  await units.focus();
  await expectVisibleFocus(units);
  await controlledPage.keyboard.press("ArrowDown");
  await expect(units).toHaveValue("imperial");
  await controlledPage.reload();
  for (const name of macros) {
    await expect(controlledPage.getByRole("checkbox", { name })).not.toBeChecked();
  }
  await expect(controlledPage.getByRole("combobox", { name: "Unit system" })).toHaveValue("imperial");
});
