import { expect, test } from "./fixtures";

// Implements DESIGN-001 SearchView controlled-response browser smoke test.
test("renders the search shell with a controlled API fixture", async ({ controlledPage }) => {
  await controlledPage.goto("/");
  await expect(controlledPage.getByRole("heading", { name: "Mealswapp" })).toBeVisible();
  await expect(controlledPage.getByRole("group", { name: "Search mode" })).toBeVisible();
});
