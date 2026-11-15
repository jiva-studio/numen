import fs from 'node:fs'
import path from 'node:path'
import type { TestCase, TestResult } from 'playwright/types/testReporter'
import { type BddFile, findBddTest } from './bdd-data.js'
import type { Environment } from './environment.js'
import { mapInputs, readInputs, type VideoInput } from './inputs.js'
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
  readonly errorMessage: string | undefined
  readonly video: string | undefined
  readonly inputs: readonly VideoInput[]
  readonly screenshot: string | undefined
  readonly environment: Environment
}

export type ScenarioSource = {
  readonly bddFile: BddFile
  readonly environment: Environment
  readonly feature: Feature
  readonly rootDir: string
  readonly outputDir: string
}

const ANSI = new RegExp(`${String.fromCharCode(27)}\\[[0-9;]*m`, 'g')

export const getFeatureKey = (featureUri: string): string =>
  featureUri
    .replace(/\.feature$/, '')
    .split('/')
    .map((part) => part.replace(/[^\w-]+/g, '-'))
    .join('.')

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
  const id = `${getFeatureKey(featureUri)}-${scenario.line}`
  const status = getScenarioStatus(result.status)
  const ran = collectBddSteps(result.steps, test.location.file)
  const videoStart = getVideoStart(result.annotations, ran) ?? result.startTime.getTime()
  const timings = mapStepTimes(
    bddTest.steps.map((step) => step.pwStepLine),
    ran,
    videoStart,
    status === 'failed',
  )
  const steps = scenario.steps.map((step, index) => ({ ...step, ...cleanTiming(timings[index]) }))
  const files = copyAttachments(
    result,
    path.join(source.outputDir, 'scenarios', id, source.environment.id),
    `scenarios/${id}/${source.environment.id}`,
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
    inputs: mapInputs(readInputs(result), videoStart),
    screenshot: files.screenshot,
    environment: source.environment,
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
  screenshot: 'screenshot.png',
} as const

const copyAttachments = (
  result: TestResult,
  dir: string,
  href: string,
): Partial<Pick<ScenarioRecord, 'video' | 'screenshot'>> => {
  const copied: Partial<Record<keyof typeof ATTACHMENTS, string>> = {}
  for (const [name, file] of Object.entries(ATTACHMENTS) as [keyof typeof ATTACHMENTS, string][]) {
    const found = result.attachments.filter((one) => one.name === name && one.path)
    const from = (name === 'screenshot' ? found.at(-1) : found[0])?.path
    if (!from || !fs.existsSync(from)) continue
    fs.mkdirSync(dir, { recursive: true })
    fs.copyFileSync(from, path.join(dir, file))
    copied[name] = `${href}/${file}`
  }
  return copied
}
