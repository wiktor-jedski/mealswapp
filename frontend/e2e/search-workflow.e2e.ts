import { expect, test } from "./fixtures";

// Implements DESIGN-001 SearchView composed Catalog, filters, retry, and Substitution workflow verification.
test("composes catalog filters, retry, and substitution inputs", async ({ controlledPage, submittedSearchRequests }) => {
  await controlledPage.goto("/");
  const query = controlledPage.getByRole("textbox", { name: "Food search" });
  await query.fill("retry");
  await controlledPage.getByRole("textbox", { name: "Filter ID" }).fill("fruit");
  await controlledPage.getByRole("button", { name: "Add filter" }).click();
  await controlledPage.getByRole("button", { name: "Search", exact: true }).click();
  await expect(controlledPage.getByRole("button", { name: "Retry" })).toBeVisible();
  await controlledPage.getByRole("button", { name: "Retry" }).click();
  await expect(controlledPage.getByTestId("result-card")).toHaveCount(10);

  await controlledPage.getByLabel("Search controls").getByRole("button", { name: "Substitution" }).click();
  await controlledPage.getByRole("textbox", { name: "Selected food ID" }).fill("food-1");
  await controlledPage.getByRole("spinbutton", { name: "Quantity" }).fill("50");
  await controlledPage.getByRole("spinbutton", { name: "Quantity" }).press("Enter");
  await controlledPage.getByRole("button", { name: "Search", exact: true }).click();
  await expect(controlledPage.getByTestId("result-card")).toHaveCount(10);
  await expect.poll(() => submittedSearchRequests.at(-1)?.substitutionInputs).toEqual([{ foodObjectId: "food-1", quantity: 50, unit: "g" }]);
});

// Implements DESIGN-001 SearchView Daily Diet structured Phase 07 rejection verification.
test("daily diet mode exposes structured rejection without a job workflow", async ({ controlledPage }) => {
  await controlledPage.goto("/");
  await controlledPage.getByLabel("Search controls").getByRole("button", { name: "Daily Diet Alternative" }).click();
  await controlledPage.getByRole("textbox", { name: "Food search" }).fill("diet");
  await controlledPage.getByRole("textbox", { name: "Daily diet ID" }).fill("61e0cae4-0f45-4854-8ac5-b228214cdd1d");
  await controlledPage.getByRole("button", { name: "Search", exact: true }).click();
  const rejection = controlledPage.getByRole("alert");
  await expect(rejection).toContainText("Daily Diet Alternative requires Phase 07 data");
  await expect(rejection).toContainText("daily_diet_phase_07_required");
  await expect(rejection).toContainText("dailyDietId");
  await expect(controlledPage.getByRole("button", { name: /job/i })).toHaveCount(0);
});
