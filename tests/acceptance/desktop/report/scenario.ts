/** One scenario of the report: its Gherkin, how each step went and where it is in the video, and its files. */
import fs from 'node:fs'
import path from 'node:path'
import type { TestCase, TestResult } from 'playwright/types/testReporter'
import { type BddFile, findBddTest } from './bdd-data.js'
import { findScenario, type Feature, type GherkinStep } from './gherkin.js'
import {
  collectBddSteps,
  getVideoStart,
  mapStepTimes,
  type StepStatus,
  type StepTiming,
} from './timeline.js'
import type { ScenarioEntry } from './tree.js'

export type ReportStep = GherkinStep & StepTiming

export type ScenarioRecord = ScenarioEntry & {
  readonly steps: readonly ReportStep[]
  /** The error of a scenario that failed outside its steps, in a hook or a fixture. */
  readonly errorMessage: string | undefined
  /** Paths relative to the report folder. */
  readonly video: string | undefined
  readonly trace: string | undefined
  readonly screenshot: string | undefined
  /** The trace, absolute, for `npx playwright show-trace`. */
  readonly tracePath: string | undefined
}

export type ScenarioSource = {
  readonly bddFile: BddFile
  readonly feature: Feature
  /** The repository root, which feature paths in the report are relative to. */
  readonly rootDir: string
  readonly outputDir: string
}

const ANSI = new RegExp(`${String.fromCharCode(27)}\\[[0-9;]*m`, 'g')

export const stripAnsi = (text: string): string => text.replace(ANSI, '')

export const getScenarioStatus = (status: TestResult['status']): StepStatus => {
  if (status === 'passed') return 'passed'
  return status === 'skipped' ? 'skipped' : 'failed'
}

export const createScenarioRecord = (
  test: TestCase,
  result: TestResult,
  source: ScenarioSource,
): ScenarioRecord | undefined => {
  const bddTest = findBddTest(source.bddFile, test.location.line)
  const scenario = bddTest && findScenario(source.feature, bddTest.pickleLine)
  if (!bddTest || !scenario) return undefined
  const featureUri = path
    .relative(source.rootDir, source.bddFile.featurePath)
    .split(path.sep)
    .join('/')
  const id = `${featureUri.replace(/\.feature$/, '').replace(/[^\w-]+/g, '-')}-${scenario.line}`
  const status = getScenarioStatus(result.status)
  const ran = collectBddSteps(result.steps, test.location.file)
  const timings = mapStepTimes(
    bddTest.steps.map((step) => step.pwStepLine),
    ran,
    getVideoStart(result.annotations, ran) ?? result.startTime.getTime(),
    status === 'failed',
  )
  const steps = scenario.steps.map((step, index) => ({ ...step, ...cleanTiming(timings[index]) }))
  const files = copyAttachments(
    result,
    path.join(source.outputDir, 'scenarios', id),
    `scenarios/${id}`,
  )
  return {
    id,
    featureUri,
    featureName: scenario.featureName,
    name: scenario.name,
    line: scenario.line,
    status,
    duration: result.duration,
    steps,
    errorMessage: readScenarioError(result, steps),
    video: files.video,
    trace: files.trace,
    screenshot: files.screenshot,
    tracePath: files.tracePath,
  }
}

const SKIPPED: StepTiming = { status: 'skipped', start: null, duration: 0 }

const cleanTiming = (timing: StepTiming = SKIPPED): StepTiming =>
  timing.errorMessage ? { ...timing, errorMessage: stripAnsi(timing.errorMessage) } : timing

const readScenarioError = (
  result: TestResult,
  steps: readonly ReportStep[],
): string | undefined => {
  const message = result.errors[0]?.message
  if (!message || steps.some((step) => step.errorMessage)) return undefined
  return stripAnsi(message)
}

const ATTACHMENTS = {
  video: 'video.webm',
  trace: 'trace.zip',
  screenshot: 'screenshot.png',
} as const

/**
 * Copies the video, the trace and the screenshot of the result next to the report. The first video
 * is the window's (support/video.ts); the last screenshot is of the window, an earlier one of the
 * blank page of the fixture.
 */
const copyAttachments = (
  result: TestResult,
  dir: string,
  href: string,
): Partial<Pick<ScenarioRecord, 'video' | 'trace' | 'screenshot' | 'tracePath'>> => {
  const copied: Partial<Record<keyof typeof ATTACHMENTS, string>> = {}
  for (const [name, file] of Object.entries(ATTACHMENTS) as [keyof typeof ATTACHMENTS, string][]) {
    const found = result.attachments.filter((one) => one.name === name && one.path)
    const from = (name === 'screenshot' ? found.at(-1) : found[0])?.path
    if (!from || !fs.existsSync(from)) continue
    fs.mkdirSync(dir, { recursive: true })
    fs.copyFileSync(from, path.join(dir, file))
    copied[name] = `${href}/${file}`
  }
  return { ...copied, ...(copied.trace && { tracePath: path.join(dir, ATTACHMENTS.trace) }) }
}
