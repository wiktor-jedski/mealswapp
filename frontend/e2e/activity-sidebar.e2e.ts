import { expect, test } from "./fixtures";

// Implements DESIGN-001 SidebarComponent desktop activity and restoration verification.
test("desktop sidebar collapses and restores authenticated history", async ({ controlledPage }) => {
  await controlledPage.goto("/");
  const sidebar = controlledPage.getByRole("complementary", { name: "Activity sidebar" });
  const searchContent = controlledPage.getByRole("heading", { name: "Food discovery" }).locator("../..");
  await expect(sidebar).toBeVisible();
  expect((await sidebar.boundingBox())!.x).toBeLessThan((await searchContent.boundingBox())!.x);
  await expect(controlledPage.getByRole("heading", { name: "History" })).toBeVisible();
  await expect(controlledPage.getByText("food-1")).toBeVisible();
  await controlledPage.getByRole("button", { name: "banana" }).click();
  await expect(controlledPage.getByRole("textbox", { name: "Food search" })).toHaveValue("banana");
  await controlledPage.getByRole("button", { name: "Collapse sidebar" }).click();
  await expect(sidebar).toHaveAttribute("data-collapsed", "true");
  await controlledPage.getByRole("button", { name: "Expand sidebar" }).click();
  await expect(controlledPage.getByRole("link", { name: "Settings" })).toBeVisible();
});

// Implements DESIGN-001 SidebarComponent anonymous and failure isolation verification.
test("anonymous activity guidance does not block core search", async ({ controlledPage }) => {
  await controlledPage.route("**/api/v1/search-history", (route) => route.fulfill({ status: 401, body: "{}" }));
  await controlledPage.goto("/");
  await expect(controlledPage.getByText("Sign in to view history and favorites.")).toBeVisible();
  await expect(controlledPage.getByRole("textbox", { name: "Food search" })).toBeEnabled();
});

// Implements DESIGN-001 SidebarComponent mobile toggle and API-failure isolation verification.
test("mobile activity toggles and failures remain non-blocking", async ({ controlledPage }) => {
  await controlledPage.setViewportSize({ width: 390, height: 844 });
  await controlledPage.route("**/api/v1/search-history", (route) => route.fulfill({ status: 500, body: "{}" }));
  await controlledPage.goto("/");
  const sidebar = controlledPage.getByRole("complementary", { name: "Activity sidebar" });
  await expect(sidebar).toBeHidden();
  await controlledPage.getByRole("button", { name: "Activity" }).click();
  await expect(sidebar).toBeVisible();
  await expect(controlledPage.getByText("Activity unavailable. Search remains available.")).toBeVisible();
  await expect(controlledPage.getByRole("textbox", { name: "Food search" })).toBeEnabled();
});
