import { describe, expect, it } from 'vitest'
import {
  findActionStart,
  findNextStart,
  findPreviousStart,
  findStepAt,
  getCursorAt,
  getSegments,
  getSetupLength,
  getStepPhases,
  getVideoLength,
} from './playback.js'

const starts = [0.5, 2, null, 4.25]

const fromSource = <T extends (...args: never[]) => unknown>(fn: T): T =>
  new Function(`return ${fn.toString()}`)() as T

describe('findStepAt', () => {
  it('finds the last step that started by the time', () => {
    expect(findStepAt(starts, 0.5)).toBe(0)
    expect(findStepAt(starts, 3.9)).toBe(1)
    expect(findStepAt(starts, 60)).toBe(3)
  })

  it('finds none before the first step', () => {
    expect(findStepAt(starts, 0.2)).toBe(-1)
  })
})

describe('findNextStart', () => {
  it('finds the start of the next step that ran', () => {
    expect(findNextStart(starts, 0)).toBe(0.5)
    expect(findNextStart(starts, 2.5)).toBe(4.25)
  })

  it('moves on from a step the video stands at', () => {
    expect(findNextStart(starts, 1.99)).toBe(4.25)
    expect(findNextStart(starts, 2.01)).toBe(4.25)
  })

  it('finds none after the last step', () => {
    expect(findNextStart(starts, 4.3)).toBeNull()
  })
})

describe('findPreviousStart', () => {
  it('goes back to the start of the current step after a second of it', () => {
    expect(findPreviousStart(starts, 3.5)).toBe(2)
  })

  it('goes to the step before within the first second of a step', () => {
    expect(findPreviousStart(starts, 4.5)).toBe(2)
    expect(findPreviousStart(starts, 2.2)).toBe(0.5)
  })

  it('goes to the start of the video from the first step', () => {
    expect(findPreviousStart(starts, 0.8)).toBe(0)
    expect(findPreviousStart(starts, 0.1)).toBe(0)
  })
})

describe('getStepPhases', () => {
  it('marks the steps before the current one played and the ones after it to come', () => {
    expect(getStepPhases(4, 1)).toEqual(['done', 'current', 'ahead', 'ahead'])
  })

  it('marks every step to come before the first step', () => {
    expect(getStepPhases(2, -1)).toEqual(['ahead', 'ahead'])
  })
})

describe('getSegments', () => {
  it('spans each step that ran from its start to the next start or the end', () => {
    expect(getSegments(starts, 5)).toEqual([
      { index: 0, left: 10, width: 30 },
      { index: 1, left: 40, width: 45 },
      { index: 3, left: 85, width: 15 },
    ])
  })

  it('keeps the last of the steps that start together', () => {
    expect(getSegments([0, 0, 0, 1], 2)).toEqual([
      { index: 2, left: 0, width: 50 },
      { index: 3, left: 50, width: 50 },
    ])
  })

  it('gives no segments for a video of unknown length', () => {
    expect(getSegments(starts, 0)).toEqual([])
  })
})

describe('getVideoLength', () => {
  const steps = [
    { start: 0.5, duration: 1200 },
    { start: 2, duration: 3000 },
    { start: null, duration: 0 },
  ]

  it('takes the duration of the media', () => {
    expect(getVideoLength(7.5, steps)).toBe(7.5)
  })

  it('takes the end of the last step when the media has no duration', () => {
    expect(getVideoLength(Number.POSITIVE_INFINITY, steps)).toBe(5)
    expect(getVideoLength(Number.NaN, steps)).toBe(5)
  })
})

describe('getSetupLength', () => {
  const steps = (...keywords: string[]) => keywords.map((keyword) => ({ keyword }))

  it('counts the Background and Given steps with the And after them', () => {
    expect(getSetupLength(steps('Given', 'And', 'Given', 'But', 'When', 'Then'))).toBe(4)
  })

  it('stops at a Then that comes before any When', () => {
    expect(getSetupLength(steps('Given', 'Then', 'When'))).toBe(1)
  })

  it('counts every step of a scenario with no When or Then', () => {
    expect(getSetupLength(steps('Given', 'And'))).toBe(2)
  })

  it('counts none for a scenario that starts with an action', () => {
    expect(getSetupLength(steps('When', 'And', 'Then'))).toBe(0)
  })

  it('counts none for no steps', () => {
    expect(getSetupLength([])).toBe(0)
  })
})

describe('findActionStart', () => {
  it('finds the start of the first step after the setup that ran', () => {
    expect(findActionStart(starts, 1)).toBe(2)
    expect(findActionStart(starts, 2)).toBe(4.25)
  })

  it('finds none when no step after the setup ran', () => {
    expect(findActionStart([1, 1, null], 2)).toBeNull()
    expect(findActionStart([1, 1], 2)).toBeNull()
  })
})

describe('the functions as the page runs them', () => {
  it('run from their own source', () => {
    expect(fromSource(findStepAt)(starts, 2.5)).toBe(1)
    expect(fromSource(findNextStart)(starts, 2.5)).toBe(4.25)
    expect(fromSource(findPreviousStart)(starts, 3.5)).toBe(2)
    expect(fromSource(getStepPhases)(2, 0)).toEqual(['current', 'ahead'])
    expect(fromSource(getSegments)(starts, 5)).toHaveLength(3)
    expect(fromSource(getVideoLength)(Number.NaN, [{ start: 1, duration: 500 }])).toBe(1.5)
    expect(fromSource(getSetupLength)([{ keyword: 'Given' }, { keyword: 'When' }])).toBe(1)
    expect(fromSource(findActionStart)(starts, 1)).toBe(2)
  })
})

describe('getCursorAt', () => {
  const clicks = [
    { t: 2, x: 0.2, y: 0.4 },
    { t: 5, x: 0.6, y: 0.8 },
  ]

  it('shows no cursor in a video without clicks', () => {
    expect(getCursorAt([], 3)).toBeNull()
  })

  it('rests on the first click before it happens', () => {
    expect(getCursorAt(clicks, 0.5)).toEqual({ x: 0.2, y: 0.4, ripple: null })
  })

  it('starts a ripple on the point of a click and ends it after 0.4 s', () => {
    expect(getCursorAt(clicks, 2)?.ripple).toEqual({ x: 0.2, y: 0.4, progress: 0 })
    expect(getCursorAt(clicks, 2.2)?.ripple?.progress).toBeCloseTo(0.5)
    expect(getCursorAt(clicks, 2.41)?.ripple).toBeNull()
  })

  it('rests on a click until 0.6 s before the next one', () => {
    expect(getCursorAt(clicks, 4.4)).toMatchObject({ x: 0.2, y: 0.4 })
  })

  it('travels to the next click with easing and lands on it', () => {
    const halfway = getCursorAt(clicks, 4.7)
    expect(halfway?.x).toBeCloseTo(0.4)
    expect(halfway?.y).toBeCloseTo(0.6)
    expect(getCursorAt(clicks, 4.55)?.x).toBeLessThan(0.3)
    expect(getCursorAt(clicks, 5)).toMatchObject({ x: 0.6, y: 0.8 })
  })

  it('travels between clicks closer than 0.6 s in the time between them', () => {
    const quick = [
      { t: 1, x: 0, y: 0 },
      { t: 1.2, x: 1, y: 1 },
    ]
    expect(getCursorAt(quick, 1.1)?.x).toBeCloseTo(0.5)
  })

  it('stays on the last click after it', () => {
    expect(getCursorAt(clicks, 9)).toEqual({ x: 0.6, y: 0.8, ripple: null })
  })

  it('works from its source text, as the report page runs it', () => {
    expect(fromSource(getCursorAt)(clicks, 4.7)?.x).toBeCloseTo(0.4)
  })
})
