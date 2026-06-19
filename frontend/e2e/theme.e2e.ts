import { expect, test } from "./fixtures";

// Implements DESIGN-016 ThemeProvider persistence and live system-theme browser verification.
test("system theme follows media while explicit sidebar overrides persist", async ({ controlledPage }) => {
  await controlledPage.emulateMedia({ colorScheme: "dark" });
  await controlledPage.goto("/");
  await expect(controlledPage.locator("html")).toHaveAttribute("data-theme", "dark");
  const preference = controlledPage.getByRole("combobox", { name: "Theme preference" });
  await preference.selectOption("light");
  await expect(controlledPage.locator("html")).toHaveAttribute("data-theme", "light");
  await controlledPage.emulateMedia({ colorScheme: "dark" });
  await expect(controlledPage.locator("html")).toHaveAttribute("data-theme", "light");
  await controlledPage.reload();
  await expect(preference).toHaveValue("light");
  await preference.selectOption("system");
  await controlledPage.emulateMedia({ colorScheme: "light" });
  await expect(controlledPage.locator("html")).toHaveAttribute("data-theme", "light");
});
