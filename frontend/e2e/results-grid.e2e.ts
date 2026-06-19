import { expect, test } from "./fixtures";

// Implements DESIGN-001 ResultsGrid populated-card, fallback, and pagination verification.
test("renders at most ten generated result cards with fallback images and page boundaries", async ({ controlledPage }) => {
  await controlledPage.goto("/");
  await controlledPage.getByRole("textbox", { name: "Food search" }).fill("apple");
  await controlledPage.getByRole("button", { name: "Search", exact: true }).click();
  const cards = controlledPage.getByTestId("result-card");
  await expect(cards).toHaveCount(10);
  await expect(cards.first()).toContainText("Fruit");
  await expect(cards.first()).toContainText("Protein1g / 100g");
  await expect(cards.first()).toContainText("Calories102");
  await expect(cards.first()).toContainText("Similarity 90% · excellent");
  await expect(cards.first().locator("img")).toHaveAttribute("src", "/assets/placeholders/fruit.svg");
  await expect(controlledPage.getByRole("button", { name: "Previous" })).toBeDisabled();
  await controlledPage.getByRole("button", { name: "Next" }).click();
  await expect(cards).toHaveCount(1);
  await expect(controlledPage.getByText("Page 2 of 2")).toBeVisible();
  await expect(controlledPage.getByRole("button", { name: "Next" })).toBeDisabled();
});

// Implements DESIGN-001 SearchView previous-page retention during observed pagination.
test("keeps previous results visible while the next page loads", async ({ controlledPage }) => {
  await controlledPage.goto("/");
  await controlledPage.getByRole("textbox", { name: "Food search" }).fill("delayed-page");
  await controlledPage.getByRole("button", { name: "Search", exact: true }).click();
  await expect(controlledPage.getByTestId("result-card")).toHaveCount(10);
  await expect(controlledPage.getByRole("heading", { name: "Page 1 apple 1", exact: true })).toBeVisible();

  await controlledPage.getByRole("button", { name: "Next" }).click();

  await expect(controlledPage.getByRole("status")).toContainText("Loading page 1");
  await expect(controlledPage.getByTestId("result-card")).toHaveCount(10);
  await expect(controlledPage.getByRole("heading", { name: "Page 1 apple 1", exact: true })).toBeVisible();
  await expect(controlledPage.getByRole("heading", { name: "Page 2 apple 1", exact: true })).toBeVisible();
  await expect(controlledPage.getByTestId("result-card")).toHaveCount(1);
});

// Implements DESIGN-001 ResultsGrid loading, empty, and error-state verification.
test("shows skeleton, empty, and safe error states", async ({ controlledPage }) => {
  await controlledPage.goto("/");
  const search = controlledPage.getByRole("textbox", { name: "Food search" });
  await search.fill("slow");
  await controlledPage.getByRole("button", { name: "Search", exact: true }).click();
  await expect(controlledPage.getByLabel("Loading search results")).toBeVisible();
  await expect(controlledPage.getByTestId("result-card")).toHaveCount(10);
  await search.fill("empty");
  await controlledPage.getByRole("button", { name: "Search", exact: true }).click();
  await expect(controlledPage.getByText("No foods matched your search.")).toBeVisible();
  await search.fill("error");
  await controlledPage.getByRole("button", { name: "Search", exact: true }).click();
  await expect(controlledPage.getByRole("alert")).toContainText("Search unavailable");
});
