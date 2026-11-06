import { defineConfig, devices } from '@playwright/test';
import type { OptionsCollector } from './e2e/support/collector';
import { lireLangues } from './e2e/support/langues';

const port = 4173;
const baseURL = `http://127.0.0.1:${String(port)}`;
const enCi = process.env.CI !== undefined && process.env.CI !== '';
const langues = lireLangues(process.env.E2E_LANGUES);
const bascule = /AC1-.*\.spec\.ts$/;
const navigateur = { ...devices['Desktop Chrome'], locale: 'fr-FR', timezoneId: 'Europe/Paris' };

export default defineConfig<OptionsCollector>({
  testDir: './e2e',
  outputDir: './test-results/e2e',
  fullyParallel: true,
  forbidOnly: enCi,
  retries: enCi ? 1 : 0,
  workers: 2,
  timeout: 60_000,
  expect: { timeout: 10_000 },
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'playwright-report' }]],
  use: {
    baseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [
    ...langues.map((langue) => ({
      name: langue,
      testIgnore: bascule,
      use: { ...navigateur, langue },
    })),
    {
      name: 'bascule-fr-en',
      testMatch: bascule,
      use: { ...navigateur, langue: 'fr' },
    },
  ],
  webServer: {
    command: `npm run build && npm run preview -- --host 127.0.0.1 --port ${String(port)} --strictPort`,
    url: baseURL,
    reuseExistingServer: !enCi,
    timeout: 240_000,
  },
});
