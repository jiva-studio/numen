/** Writes the report folder: the page, its script, style and font, and the data of the run as a script. */
import fs from 'node:fs'
import { createRequire } from 'node:module'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { createOverview, type GitRevision } from './overview.js'
import * as playback from './playback.js'
import type { ScenarioRecord } from './scenario.js'
import { buildTree } from './tree.js'

const PAGE_DIR = path.join(path.dirname(fileURLToPath(import.meta.url)), 'page')
const PAGE_FILES = ['index.html', 'report.css', 'video.js', 'video-load.js', 'report.js']
const FONT = '@fontsource-variable/inter/files/inter-latin-wght-normal.woff2'

export type RunInfo = GitRevision & { readonly startedAt: Date }

/** Empties the folder and keeps the folder itself, which a server of the report may be serving from. */
export const clearFolder = (dir: string): void => {
  fs.mkdirSync(dir, { recursive: true })
  for (const entry of fs.readdirSync(dir))
    fs.rmSync(path.join(dir, entry), { recursive: true, force: true })
}

export const writeReport = (
  outputDir: string,
  scenarios: readonly ScenarioRecord[],
  run: RunInfo,
): void => {
  fs.mkdirSync(outputDir, { recursive: true })
  for (const file of PAGE_FILES)
    fs.copyFileSync(path.join(PAGE_DIR, file), path.join(outputDir, file))
  copyFont(outputDir)
  const features = buildTree(scenarios)
  const data = { overview: createOverview(features, run.startedAt, run), features }
  // A script and not JSON: a page opened from file:// cannot fetch a file next to it.
  const functions = Object.entries(playback).map(([name, fn]) => `  ${name}: ${String(fn)},`)
  const script = [
    `window.REPORT = ${JSON.stringify(data)};`,
    'window.playback = {',
    ...functions,
    '};',
    '',
  ].join('\n')
  fs.writeFileSync(path.join(outputDir, 'data.js'), script)
}

/** Whether the font was copied; without it the page falls back to the system font. */
const copyFont = (outputDir: string): boolean => {
  try {
    const from = createRequire(import.meta.url).resolve(FONT)
    fs.mkdirSync(path.join(outputDir, 'fonts'), { recursive: true })
    fs.copyFileSync(from, path.join(outputDir, 'fonts', 'inter.woff2'))
    return true
  } catch {
    return false
  }
}
