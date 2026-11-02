// Writes shards.json from Playwright JSON reports (`--reporter=json`): the seconds of each spec
// file, the mean of the reports that ran it. A file the reports do not hold keeps its old seconds.
//
//   node scripts/measure-shards.mjs chromium.json webkit.json
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { listSpecs, readDurations, writeDurations } from './shards.mjs'

const reports = process.argv.slice(2)
if (reports.length === 0) {
  console.error('measure-shards: name the JSON reports of the runs')
  process.exit(1)
}

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const perReport = reports.map((report) => {
  const seconds = new Map()
  const visit = (suite) => {
    for (const spec of suite.specs ?? []) {
      const file = path
        .relative(root, path.resolve(root, 'tests', spec.file))
        .split(path.sep)
        .join('/')
      const took = spec.tests.reduce(
        (sum, one) => sum + one.results.reduce((inner, result) => inner + result.duration, 0),
        0,
      )
      seconds.set(file, (seconds.get(file) ?? 0) + took / 1000)
    }
    for (const child of suite.suites ?? []) visit(child)
  }
  for (const suite of JSON.parse(fs.readFileSync(report, 'utf8')).suites) visit(suite)
  return seconds
})

const old = readDurations()
const durations = {}
for (const spec of listSpecs()) {
  const measured = perReport.map((seconds) => seconds.get(spec)).filter((one) => one !== undefined)
  durations[spec] =
    measured.length > 0
      ? Math.max(1, Math.round(measured.reduce((a, b) => a + b, 0) / measured.length))
      : (old[spec] ?? 1)
}
writeDurations(durations)
