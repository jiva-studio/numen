import { VIDEO_START } from '../support/video.js'

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
  readonly start: number | null
  readonly duration: number
  readonly errorMessage?: string
}

export const collectBddSteps = (steps: readonly StepNode[], specFile: string): StepNode[] =>
  steps.flatMap((step) =>
    step.category === 'test.step' && step.location?.file === specFile
      ? [step]
      : collectBddSteps(step.steps, specFile),
  )

export const getVideoStart = (
  annotations: readonly Annotation[],
  steps: readonly StepNode[],
): number | undefined => {
  const marked = Number(annotations.find((one) => one.type === VIDEO_START)?.description)
  return Number.isFinite(marked) ? marked : steps[0]?.startTime.getTime()
}

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
    return {
      status: 'failed',
      start,
      duration: step.duration,
      ...(message && { errorMessage: message }),
    }
  }
  return { status: 'passed', start, duration: step.duration }
}
