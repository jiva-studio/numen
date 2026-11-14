import fs from 'node:fs'
import { createRequire } from 'node:module'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import {
  Check,
  ChevronRight,
  CircleAlert,
  Keyboard,
  Link,
  Maximize,
  Minimize,
  Pause,
  Play,
  Search,
  StepForward,
} from 'lucide'
import { createOverview, type GitRevision } from './overview.js'
import * as playback from './playback.js'
import type { ScenarioRecord } from './scenario.js'
import { buildTree } from './tree.js'

const PAGE_DIR = path.join(path.dirname(fileURLToPath(import.meta.url)), 'page')
const PAGE_FILES = [
  'index.html',
  'tokens/colors.css',
  'report.css',
  'controls.js',
  'video.js',
  'video-load.js',
  'report.js',
]
const ICONS = {
  check: Check,
  'chevron-right': ChevronRight,
  'circle-alert': CircleAlert,
  keyboard: Keyboard,
  link: Link,
  maximize: Maximize,
  minimize: Minimize,
  pause: Pause,
  play: Play,
  search: Search,
  'step-forward': StepForward,
}
const FONT = '@fontsource-variable/inter/files/inter-latin-wght-normal.woff2'

export type RunInfo = GitRevision & { readonly startedAt: Date }

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
  for (const file of PAGE_FILES) copyPageFile(outputDir, file)
  copyFont(outputDir)
  const features = buildTree(scenarios)
  const data = { overview: createOverview(features, run.startedAt, run), features }
  const functions = Object.entries(playback).map(([name, fn]) => `  ${name}: ${String(fn)},`)
  const script = [
    `window.REPORT = ${JSON.stringify(data)};`,
    `window.ICONS = ${JSON.stringify(ICONS)};`,
    'window.playback = {',
    ...functions,
    '};',
    '',
  ].join('\n')
  fs.writeFileSync(path.join(outputDir, 'data.js'), script)
}

const copyPageFile = (outputDir: string, file: string): void => {
  const to = path.join(outputDir, file)
  fs.mkdirSync(path.dirname(to), { recursive: true })
  fs.copyFileSync(path.join(PAGE_DIR, file), to)
}

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
