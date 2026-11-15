import { describe, expect, it } from 'vitest'
import type { RecordedInput } from '../support/inputs.js'
import { mapInputs } from './inputs.js'

const START = Date.UTC(2031, 4, 2, 9, 0, 0)

const createInput = (overrides: Partial<RecordedInput>): RecordedInput => ({
  page: 0,
  kind: 'down',
  at: START,
  x: 640,
  y: 430,
  width: 1280,
  height: 860,
  ...overrides,
})

describe('mapInputs', () => {
  it('gives each input its video time in seconds and its place as a share of the window', () => {
    expect(mapInputs([createInput({ at: START + 2500, x: 320, y: 215 })], START)).toEqual([
      { kind: 'down', t: 2.5, x: 0.25, y: 0.25 },
    ])
  })

  it('gives the element of a key its size as a share of the window', () => {
    const key = createInput({ kind: 'key', key: 'Enter', mods: [], x: 128, y: 86, w: 320, h: 43 })
    expect(mapInputs([key], START)).toEqual([
      { kind: 'key', key: 'Enter', mods: [], t: 0, x: 0.1, y: 0.1, w: 0.25, h: 0.05 },
    ])
  })

  it('keeps the inputs of the first window and orders them by time', () => {
    const inputs = [
      createInput({ at: START + 3000 }),
      createInput({ at: START + 1000, page: 1 }),
      createInput({ at: START + 2000 }),
    ]
    expect(mapInputs(inputs, START).map((input) => input.t)).toEqual([2, 3])
  })

  it('puts an input from before the video at its start', () => {
    expect(mapInputs([createInput({ at: START - 40 })], START)[0]?.t).toBe(0)
  })
})
