/** Where each step of a scenario is in its video, and how it ended. */
import { VIDEO_START } from '../support/video-start.js'

/** The part of Playwright's `TestStep` the report reads. */
export type StepNode = {
  readonly category: string
  readonly location?: { readonly file: string; readonly line: number }
  readonly startTime: Date
  readonly duration: number
  readonly error?: { readonly message?: string }
  readonly steps: readonly StepNode[]
}

export type Annotation = { readonly type: string; readonly description?: string }

export type StepStatus = 'passed' | 'failed' | 'skipped'

export type StepTiming = {
  readonly status: StepStatus
  /** Seconds from the start of the video; null for a step that did not run. */
  readonly start: number | null
  /** Milliseconds. */
  readonly duration: number
  readonly errorMessage?: string
}

/**
 * The steps of the scenario among Playwright's steps: playwright-bdd runs each one as a `test.step`
 * located at its call in the generated spec, a Background step under the hook that runs it.
 */
export const collectBddSteps = (steps: readonly StepNode[], specFile: string): StepNode[] =>
  steps.flatMap((step) =>
    step.category === 'test.step' && step.location?.file === specFile
      ? [step]
      : collectBddSteps(step.steps, specFile),
  )

/** The video start the page fixture marked, or the start of the first step when it marked none. */
export const getVideoStart = (
  annotations: readonly Annotation[],
  steps: readonly StepNode[],
): number | undefined => {
  const marked = Number(annotations.find((one) => one.type === VIDEO_START)?.description)
  return Number.isFinite(marked) ? marked : steps[0]?.startTime.getTime()
}

/**
 * The timing of each step, by the line it is called from in the generated spec. A failed scenario
 * whose steps carry no error failed in the last step that ran: a timeout ends a step without one.
 */
export const mapStepTimes = (
  pwStepLines: readonly number[],
  steps: readonly StepNode[],
  videoStart: number,
  isFailed: boolean,
): StepTiming[] => {
  const ran = pwStepLines.map((line) => steps.find((step) => step.location?.line === line))
  const lastRan = ran.reduce((last, step, index) => (step ? index : last), -1)
  const blamesLast = isFailed && !ran.some((step) => step?.error)
  return ran.map((step, index) =>
    step ? toTiming(step, videoStart, blamesLast && index === lastRan) : SKIPPED,
  )
}

const SKIPPED: StepTiming = { status: 'skipped', start: null, duration: 0 }

const toTiming = (step: StepNode, videoStart: number, isBlamed: boolean): StepTiming => {
  const start = Math.max(0, (step.startTime.getTime() - videoStart) / 1000)
  const message = step.error?.message
  if (message !== undefined || isBlamed) {
    return { status: 'failed', start, duration: step.duration, ...(message && { errorMessage: message }) }
  }
  return { status: 'passed', start, duration: step.duration }
}

/**
 * The index of the step the video shows at `time` seconds: the last that started by then, or -1.
 * The page runs this function as written here (scenario-report.ts writes its source into the page),
 * so it uses nothing from outside it.
 */
export const findStepAt = (starts: readonly (number | null)[], time: number): number => {
  let found = -1
  starts.forEach((start, index) => {
    if (start !== null && start <= time) found = index
  })
  return found
}
