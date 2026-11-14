/**
 * The contract of the index stats service: the fixtures in testdata/api are the JSON the Go messages
 * encode to, and they are read here as the TypeScript messages.
 */
import { describe, expect, it } from 'vitest'
import type { Methods } from './stats'

const fixtures = import.meta.glob('../../testdata/api/*.json', {
  eager: true,
  import: 'default',
}) as Record<string, unknown>

const read = (name: string): unknown => fixtures[`../../testdata/api/${name}.json`]

const EXPECTED: { [M in keyof Methods]: Methods[M] } = {
  GetStats: {
    request: {},
    response: {
      vectors: 12480,
      chunks: 13100,
      waiting: 620,
      sources: 84,
      kinds: [
        { kind: 'notes', sources: 70, chunks: 4100 },
        { kind: 'pdf', sources: 9, chunks: 7000 },
        { kind: 'epub', sources: 5, chunks: 2000 },
      ],
      model: 'bge-m3',
      recipe: 'local|bge-m3|1024|512|mean|int8@0.4',
      dimensions: 1024,
      profile: 'balanced',
      threads: 4,
      speed: 18.5,
      changedAt: '2026-11-02T12:00:07Z',
      databaseBytes: 73400320,
      vectorBytes: 52428800,
    },
  },
}

describe('index stats service contract', () => {
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
