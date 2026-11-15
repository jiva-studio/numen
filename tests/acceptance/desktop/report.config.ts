import { defineConfig } from 'playwright/test'

export default defineConfig({
  testDir: '.features-gen',
  reporter: [['./report/scenario-report.ts', { outputFolder: 'acceptance-report' }]],
})
