import { describe, expect, it } from 'vitest'
import type { RecordedClick } from '../support/clicks.js'
import { mapClicks } from './clicks.js'

const START = Date.UTC(2031, 4, 2, 9, 0, 0)

const createClick = (overrides: Partial<RecordedClick>): RecordedClick => ({
  page: 0,
  at: START,
  x: 640,
  y: 430,
  width: 1280,
  height: 860,
  ...overrides,
})

describe('mapClicks', () => {
  it('gives each click its video time in seconds and its place as a share of the window', () => {
    expect(mapClicks([createClick({ at: START + 2500, x: 320, y: 215 })], START)).toEqual([
      { t: 2.5, x: 0.25, y: 0.25 },
    ])
  })

  it('keeps the clicks of the first window and orders them by time', () => {
    const clicks = [
      createClick({ at: START + 3000 }),
      createClick({ at: START + 1000, page: 1 }),
      createClick({ at: START + 2000 }),
    ]
    expect(mapClicks(clicks, START).map((click) => click.t)).toEqual([2, 3])
  })

  it('puts a click from before the video at its start', () => {
    expect(mapClicks([createClick({ at: START - 40 })], START)[0]?.t).toBe(0)
  })
})
