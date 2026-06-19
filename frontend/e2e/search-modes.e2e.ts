import { expect, test } from "./fixtures";

// Implements DESIGN-001 SearchView mode and Substitution Input browser verification.
test("mode controls precede search/settings and substitution inputs add with Enter", async ({ controlledPage }) => {
  await controlledPage.goto("/");
  const modeGroup = controlledPage.getByRole("group", { name: "Search mode" });
  const search = controlledPage.getByRole("textbox", { name: "Food search" });
  const settings = controlledPage.getByRole("heading", { name: "Search settings" });
  expect((await modeGroup.boundingBox())!.y).toBeLessThan((await search.boundingBox())!.y);
  expect((await search.boundingBox())!.y).toBeLessThan((await settings.boundingBox())!.y);

  await controlledPage.getByLabel("Search controls").getByRole("button", { name: "Substitution" }).click();
  await controlledPage.getByRole("textbox", { name: "Selected food ID" }).fill("food-1");
  await controlledPage.getByRole("spinbutton", { name: "Quantity" }).fill("125");
  await controlledPage.getByRole("combobox", { name: "Unit", exact: true }).selectOption("g");
  await controlledPage.getByRole("spinbutton", { name: "Quantity" }).press("Enter");
  await expect(controlledPage.getByText("food-1: 125 g")).toBeVisible();
  await controlledPage.getByRole("button", { name: "Remove food-1" }).click();
  await expect(controlledPage.getByText("food-1: 125 g")).toHaveCount(0);

  await controlledPage.getByRole("button", { name: "Daily Diet Alternative" }).click();
  await expect(controlledPage.getByRole("textbox", { name: "Daily diet ID" })).toBeVisible();
  await expect(controlledPage.getByRole("status")).toContainText("Phase 07");
});
