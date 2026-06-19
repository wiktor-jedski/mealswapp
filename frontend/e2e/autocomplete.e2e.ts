import { expect, test } from "./fixtures";

// Implements DESIGN-001 AutocompleteDropdown ranked keyboard and ARIA verification.
test("autocomplete expands in flow, preserves rank, and supports keyboard selection/dismissal", async ({ controlledPage }) => {
  await controlledPage.goto("/");
  await controlledPage.getByLabel("Search controls").getByRole("button", { name: "Substitution" }).click();
  const combobox = controlledPage.getByRole("combobox", { name: "Find substitution food" });
  const before = await controlledPage.getByRole("spinbutton", { name: "Quantity" }).boundingBox();
  await combobox.fill("app");
  await controlledPage.waitForTimeout(170);
  await expect(combobox).toHaveAttribute("aria-expanded", "true");
  const options = controlledPage.getByRole("listbox", { name: "Autocomplete suggestions" }).getByRole("option");
  await expect(options).toHaveText(["Apple sauce", "Apple"]);
  const after = await controlledPage.getByRole("spinbutton", { name: "Quantity" }).boundingBox();
  expect(after!.y).toBeGreaterThan(before!.y);
  await combobox.press("Tab");
  await expect(options.first()).toBeFocused();
  await options.first().press("Shift+Tab");
  await expect(combobox).toBeFocused();
  await combobox.press("Enter");
  await expect(controlledPage.getByText("food-2: 100 g")).toBeVisible();

  await combobox.fill("app");
  await controlledPage.waitForTimeout(170);
  await combobox.press("Escape");
  await expect(combobox).toHaveAttribute("aria-expanded", "false");
});
