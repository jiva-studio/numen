/**
 * The contract of the importer service: the fixtures in testdata/api are the JSON the Go messages
 * encode to, and they are read here as the TypeScript messages.
 */
import { describe, expect, it } from 'vitest'
import type { Methods } from './importer'

const fixtures = import.meta.glob('../../testdata/api/*.json', {
  eager: true,
  import: 'default',
}) as Record<string, unknown>

const read = (name: string): unknown => fixtures[`../../testdata/api/${name}.json`]

const EXPECTED: { [M in keyof Methods]: Methods[M] } = {
  GetLink: {
    request: { path: 'links/clip.md' },
    response: {
      url: 'https://www.youtube.com/watch?v=tidalmill01',
      isCopyable: true,
      copySize: 6000,
      isTextDeleted: false,
      embedPage: 'embed?address=https%3A%2F%2Fwww.youtube.com%2Fwatch%3Fv%3Dtidalmill01',
    },
  },
  ReadArticle: {
    request: { path: 'links/article.md' },
    response: { text: 'A tidal mill stored seawater.' },
  },
  FetchText: { request: { path: 'links/clip.md', isUserStarted: true }, response: {} },
  CopyVideo: { request: { path: 'links/clip.md' }, response: {} },
  DeleteCopy: { request: { path: 'links/clip.md' }, response: {} },
  CreateLinks: {
    request: {
      folder: 'talks',
      urls: ['https://www.youtube.com/watch?v=tidalmill01', 'not an address'],
    },
    response: { paths: ['talks/tidal-mill.md'] },
  },
}

describe('importer service contract', () => {
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
