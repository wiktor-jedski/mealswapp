import { defineConfig, devices } from "@playwright/test";

// Implements DESIGN-001 SearchView browser test harness and docs/design/01_TECH_STACK.md Playwright + axe toolchain.
const webServerCommand =
  process.env.MEALSWAPP_PLAYWRIGHT_REUSE_BUILD === "1"
    ? "bun run preview"
    : "bun run build && bun run preview";

export default defineConfig({
  testDir: "./tests",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.MEALSWAPP_PLAYWRIGHT_WORKERS
    ? Number(process.env.MEALSWAPP_PLAYWRIGHT_WORKERS)
    : process.env.CI
      ? 1
      : undefined,
  reporter: [["list"]],
  use: {
    baseURL: "http://localhost:4173",
    trace: "on-first-retry",
    screenshot: "only-on-failure"
  },
  projects: [
    {
      name: "desktop-chromium",
      use: { ...devices["Desktop Chrome"] }
    },
    {
      name: "mobile-chromium",
      use: { ...devices["Pixel 5"] }
    }
  ],
  webServer: {
    command: webServerCommand,
    url: "http://localhost:4173",
    reuseExistingServer: !process.env.CI,
    timeout: 120_000
  }
});
