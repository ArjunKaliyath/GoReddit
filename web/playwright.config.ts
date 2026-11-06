import { defineConfig } from "@playwright/test";
export default defineConfig({
  testDir: "./e2e",
  workers: 1,
  timeout: 30000,
  use: {
    baseURL: "http://127.0.0.1:18080",
    headless: true,
    launchOptions: { executablePath: process.env.CHROMIUM_PATH || undefined },
  },
  webServer: {
    command:
      "cd .. && go run ./cmd/server -addr 127.0.0.1:18080 -db data/e2e.db",
    url: "http://127.0.0.1:18080/api/health",
    reuseExistingServer: false,
    timeout: 120000,
  },
});
