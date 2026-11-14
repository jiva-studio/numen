/** Writes the report folder: the page, its script and style, and the data of the run as a script. */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import type { ScenarioRecord } from './scenario.js'
import { findStepAt } from './timeline.js'
import { buildTree, countStatuses } from './tree.js'

const PAGE_DIR = path.join(path.dirname(fileURLToPath(import.meta.url)), 'page')
const PAGE_FILES = ['index.html', 'report.css', 'report.js']

export const writeReport = (outputDir: string, scenarios: readonly ScenarioRecord[]): void => {
  fs.mkdirSync(outputDir, { recursive: true })
  for (const file of PAGE_FILES)
    fs.copyFileSync(path.join(PAGE_DIR, file), path.join(outputDir, file))
  const data = {
    createdAt: new Date().toISOString(),
    counts: countStatuses(scenarios),
    features: buildTree(scenarios),
  }
  // A script and not JSON: a page opened from file:// cannot fetch a file next to it.
  const script = [
    `window.REPORT = ${JSON.stringify(data)};`,
    `window.findStepAt = ${findStepAt.toString()};`,
    '',
  ].join('\n')
  fs.writeFileSync(path.join(outputDir, 'data.js'), script)
}
