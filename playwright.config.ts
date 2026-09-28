import {defineConfig} from '@playwright/test';

export default defineConfig({
  testDir: 'tests',
  testMatch: '*.spec.ts',
  fullyParallel: false,
  workers: 1,
  timeout: 120000,
  expect: {timeout: 15000},
  use: {
    viewport: {width: 1280, height: 720},
    trace: 'off',
  },
});
