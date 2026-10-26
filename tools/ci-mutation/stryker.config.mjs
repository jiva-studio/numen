/**
 * Stryker's settings for one run of a `ts` target, handed over by `mutate.mjs`
 * in `NUMEN_MUTATION`. Stryker runs from the repository root, and its sandbox holds the
 * target's package and the test data its tests read.
 */
import { keepOnly } from './targets.mjs'

const run = JSON.parse(process.env.NUMEN_MUTATION ?? 'null')
if (!run) throw new Error('stryker.config.mjs is read by mutate.mjs, which sets NUMEN_MUTATION')

const at = run.target.at

export default {
  testRunner: 'vitest',
  plugins: ['@stryker-mutator/vitest-runner'],
  vitest: { configFile: run.vitestConfig, dir: at, related: false },
  coverageAnalysis: 'perTest',
  // A static mutant is evaluated once at load and needs the whole suite per mutant.
  ignoreStatic: true,
  mutate: run.mutate,
  ignorePatterns: [
    ...keepOnly([at, 'tsconfig.base.json', '.mutation/vitest', ...run.target.testdata]),
    `/${at}/dist`,
    `/${at}/storybook-static`,
    `/${at}/coverage`,
  ],
  incremental: run.incrementalFile !== null,
  incrementalFile: run.incrementalFile ?? undefined,
  concurrency: run.concurrency,
  reporters: ['clear-text', 'progress-append-only', 'json', 'html'],
  jsonReporter: { fileName: run.report },
  htmlReporter: { fileName: run.html },
  clearTextReporter: { maxTestsToLog: 0, reportScoreTable: true },
  thresholds: { high: 80, low: 60, break: null },
  tempDirName: '.mutation/stryker-tmp',
  cleanTempDir: true,
  dryRunTimeoutMinutes: 30,
}
