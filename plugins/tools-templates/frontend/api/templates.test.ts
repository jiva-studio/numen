/**
 * The contract of the templates service: the fixtures in testdata/api are the JSON the Go messages
 * encode to, and they are read here as the TypeScript messages.
 */
import { describe, expect, it } from 'vitest'
import type {
  CreateNoteRequest,
  CreateNoteResponse,
  ListFieldsRequest,
  ListFieldsResponse,
} from './templates'

const fixtures = import.meta.glob('../../testdata/api/*.json', {
  eager: true,
  import: 'default',
}) as Record<string, unknown>

const read = (name: string): unknown => fixtures[`../../testdata/api/${name}.json`]

interface Exchange<Request, Response> {
  method: string
  request: Request
  response: Response
}

const EXPECTED: [
  Exchange<ListFieldsRequest, ListFieldsResponse>,
  Exchange<CreateNoteRequest, CreateNoteResponse>,
] = [
  {
    method: 'ListFields',
    request: { template: 'Templates/Meeting.md', folder: 'notes' },
    response: {
      fields: [
        { kind: 'text', label: 'Client', options: [], default: 'Acme' },
        { kind: 'select', label: 'Kind', options: ['call', 'visit'], default: 'call' },
      ],
      etag: '9f2c1a',
    },
  },
  {
    method: 'CreateNote',
    request: {
      template: 'Templates/Meeting.md',
      folder: 'notes',
      values: { client: 'Acme', kind: 'visit' },
      etag: '9f2c1a',
    },
    response: {
      path: 'notes/Meeting.md',
      warning: 'note is made but not indexed: notes/Meeting.md: index is down',
    },
  },
]

describe('the templates service contract', () => {
  for (const { method, ...expected } of EXPECTED) {
    it(`${method} fixtures equal the typed messages`, () => {
      expect(read(`${method}.request`)).toEqual(expected.request)
      expect(read(`${method}.response`)).toEqual(expected.response)
    })
  }

  it('has a fixture pair for every method', () => {
    const names = Object.keys(fixtures).map((path) => path.split('/').pop())
    expect(names).toHaveLength(EXPECTED.length * 2)
  })
})
