/**
 * The contract of the updater service: the fixtures in testdata/api are the JSON the Go messages
 * encode to, and they are read here as the TypeScript messages.
 */
import { describe, expect, it } from 'vitest'
import type { Methods } from './updater'

const fixtures = import.meta.glob('../../testdata/api/*.json', {
  eager: true,
  import: 'default',
}) as Record<string, unknown>

const read = (name: string): unknown => fixtures[`../../testdata/api/${name}.json`]

const EXPECTED: { [M in keyof Methods]: Methods[M] } = {
  Check: {
    request: { app: { version: '2026.9.0', build: 900, platform: 'windows' } },
    response: {
      state: 'available',
      update: {
        version: '2026.10.1',
        build: 950,
        channel: 'stable',
        releasedAt: '2026-10-15T12:00:00Z',
        notesUrl: 'https://github.com/jiva-studio/numen/releases/tag/v2026.10.1',
        downloadUrl: 'https://dl.numen.md/numen-windows-setup.exe',
        file: {
          name: 'numen-windows-setup.exe',
          arch: 'amd64',
          kind: 'exe',
          url: 'https://dl.numen.md/numen-windows-setup.exe',
          size: 75000000,
          sha256: '0f1e2d3c',
        },
      },
    },
  },
  GetStatus: { request: {}, response: { state: 'current' } },
}

describe('updater service contract', () => {
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
