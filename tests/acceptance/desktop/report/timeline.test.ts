import { describe, expect, it } from 'vitest'
import { collectBddSteps, getVideoStart, mapStepTimes, type StepNode } from './timeline.js'

const SPEC = '/repo/.features-gen/features/garden/water-plants.feature.spec.js'
const T0 = Date.UTC(2031, 4, 2, 9, 0, 0)

const createStep = (overrides: Partial<StepNode> & { line?: number; at?: number }): StepNode => {
  const { line, at = 0, ...rest } = overrides
  return {
    category: 'test.step',
    startTime: new Date(T0 + at),
    duration: 100,
    steps: [],
    ...(line !== undefined && { location: { file: SPEC, line } }),
    ...rest,
  }
}

describe('collectBddSteps', () => {
  it('finds the steps of the spec under hooks and leaves out the steps of the drivers', () => {
    const background = createStep({ line: 7, at: 200 })
    const tree = [
      createStep({
        category: 'hook',
        steps: [
          createStep({ category: 'fixture' }),
          createStep({ category: 'hook', steps: [background] }),
        ],
      }),
      createStep({
        line: 12,
        at: 900,
        steps: [createStep({ location: { file: '/repo/drivers/garden.ts', line: 40 } })],
      }),
    ]

    expect(collectBddSteps(tree, SPEC).map((step) => step.location?.line)).toEqual([7, 12])
  })
})

describe('getVideoStart', () => {
  it('takes the mark of the page fixture', () => {
    const marked = [{ type: 'video-start', description: String(T0 - 1500) }]
    expect(getVideoStart(marked, [createStep({ line: 7 })])).toBe(T0 - 1500)
  })

  it('takes the start of the first step without a mark', () => {
    expect(getVideoStart([{ type: 'issue' }], [createStep({ line: 7, at: 300 })])).toBe(T0 + 300)
  })
})

describe('mapStepTimes', () => {
  it('gives each step its start in the video in seconds, and a step that did not run none', () => {
    const steps = [
      createStep({ line: 7, at: 1250, duration: 400 }),
      createStep({ line: 8, at: 2000, error: { message: 'no plant "Fern"' } }),
    ]

    expect(mapStepTimes([7, 8, 9], steps, T0, true)).toEqual([
      { status: 'passed', start: 1.25, duration: 400 },
      { status: 'failed', start: 2, duration: 100, errorMessage: 'no plant "Fern"' },
      { status: 'skipped', start: null, duration: 0 },
    ])
  })

  it('blames the last step that ran when a failed scenario has no step error', () => {
    const steps = [createStep({ line: 7, at: 100 }), createStep({ line: 8, at: 600 })]

    expect(mapStepTimes([7, 8, 9], steps, T0, true).map((step) => step.status)).toEqual([
      'passed',
      'failed',
      'skipped',
    ])
  })

  it('starts a step that began before the video at zero', () => {
    expect(mapStepTimes([7], [createStep({ line: 7, at: -50 })], T0, false)[0]?.start).toBe(0)
  })
})
