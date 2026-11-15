/**
 * The contract of the relevance service: the fixtures in testdata/api are the JSON the Go messages
 * encode to, and they are read here as the TypeScript messages.
 */
import { describe, expect, it } from 'vitest'
import type { Methods } from './relevance'

const fixtures = import.meta.glob('../../../testdata/api/*.json', {
  eager: true,
  import: 'default',
}) as Record<string, unknown>

const read = (name: string): unknown => fixtures[`../../../testdata/api/${name}.json`]

const CALIBRATION = {
  median: 0.27,
  spread: 0.07,
  histogram: [0, 1, 4, 9, 4, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
  queryCount: 200,
  chunkCount: 500,
  calibratedAt: 1_792_000_000,
}

const EXPECTED: { [M in keyof Methods]: Methods[M] } = {
  GetFloor: {
    request: {},
    response: {
      floor: 0.41,
      source: 'calibrated',
      model: 'bge-m3',
      automaticFloor: 0.41,
      isManual: false,
      manualFloor: 0,
      calibration: CALIBRATION,
    },
  },
  CalibrateFloor: {
    request: {},
    response: {
      outcome: 'too-few',
      state: {
        floor: 0.3,
        source: 'default',
        model: 'bge-m3',
        automaticFloor: 0.3,
        isManual: false,
        manualFloor: 0,
      },
    },
  },
  SetFloor: {
    request: { isManual: true, floor: 0.6 },
    response: {
      state: {
        floor: 0.6,
        source: 'manual',
        model: 'bge-m3',
        automaticFloor: 0.4,
        isManual: true,
        manualFloor: 0.6,
      },
    },
  },
  ProbeFloor: {
    request: { query: 'entropy' },
    response: { bestSimilarity: 0.46, floor: 0.41, doesPass: true },
  },
}

describe('relevance service contract', () => {
  for (const [method, expected] of Object.entries(EXPECTED)) {
    it(`${method} fixtures equal the typed messages`, () => {
      expect(read(`${method}.request`)).toEqual(expected.request)
      expect(read(`${method}.response`)).toEqual(expected.response)
    })
  }

  it('has a fixture pair for every method', () => {
    const names = Object.keys(fixtures).map((path) => path.split('/').pop())
    expect(names).toHaveLength(Object.keys(EXPECTED).length * 2)
  })
})
