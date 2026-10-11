/**
 * Stryker's settings for one run of a `ts` target, handed over by `mutate.mjs`
 * in `NUMEN_MUTATION`. Stryker runs from `modules/`, and its sandbox holds the
 * target's package and the test data its tests read.
 */
import { keepOnly } from './targets.mjs'

const run = JSON.parse(process.env.NUMEN_MUTATION ?? 'null')
if (!run) throw new Error('stryker.config.mjs is read by mutate.mjs, which sets NUMEN_MUTATION')

const strip = (path) => path.replace(/^modules\//, '')

const at = strip(run.target.at)

export default {
  testRunner: 'vitest',
  plugins: ['@stryker-mutator/vitest-runner'],
  vitest: { configFile: `${at}/vitest.mutation.config.ts`, dir: at, related: false },
  coverageAnalysis: process.env.NUMEN_MUTATION_COVERAGE ?? 'perTest',
  mutate: run.mutate,
  ignorePatterns: [
    ...keepOnly([at, ...(run.target.testdata ?? []).map(strip)]),
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
