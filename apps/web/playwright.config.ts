import { defineConfig, devices } from "@playwright/test";

// The suite runs the production build against a mocked Console API (see
// e2e/mock-api.ts). It verifies UI behavior only; it is not evidence about a
// deployed API, indexer, wallet, or Testnet.
const WEB_PORT = 3147;

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["github"], ["list"]] : "list",
  use: { baseURL: `http://127.0.0.1:${WEB_PORT}`, trace: "retain-on-failure" },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: {
    command: `pnpm exec next start -p ${WEB_PORT} -H 127.0.0.1`,
    url: `http://127.0.0.1:${WEB_PORT}/api/health`,
    reuseExistingServer: false,
    timeout: 60_000,
  },
});
