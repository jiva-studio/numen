/**
 * The contract of the todos service: the fixtures in testdata/api are the JSON the Go messages
 * encode to, and they are read here as the TypeScript messages.
 */
import { describe, expect, it } from 'vitest'
import type { Methods } from './todos'

const fixtures = import.meta.glob('../../testdata/api/*.json', {
  eager: true,
  import: 'default',
}) as Record<string, unknown>

const read = (name: string): unknown => fixtures[`../../testdata/api/${name}.json`]

const EXPECTED: { [M in keyof Methods]: Methods[M] } = {
  GetSummary: {
    request: { today: '2026-11-14' },
    response: {
      isReady: true,
      overdueCount: 2,
      todayCount: 1,
      noDateCount: 3,
      unreadableCount: 1,
    },
  },
  ListTasks: {
    request: {
      today: '2026-11-14',
      filter: {
        status: ['open'],
        due: 'overdue',
        from: '',
        to: '',
        projects: ['home'],
        contexts: ['phone'],
        priorities: ['A'],
        path: 'Projects/',
        text: 'milk',
      },
      limit: 20,
    },
    response: {
      tasks: [
        {
          path: 'Projects/Home.md',
          line: 7,
          raw: '- [ ] (A) buy milk due:2026-11-13 rec:!1w +home @phone',
          text: 'buy milk',
          status: 'open',
          priority: 'A',
          due: '2026-11-13',
          doneOn: '',
          recurrence: 'rec:!1w',
          projects: ['home'],
          contexts: ['phone'],
          issues: [],
          depth: 0,
        },
        {
          path: 'Daily/2026-11-12.md',
          line: 3,
          raw: '  - [ ] call due:2026-02-30',
          text: 'call due:2026-02-30',
          status: 'open',
          priority: '',
          due: '',
          doneOn: '',
          recurrence: '',
          projects: [],
          contexts: [],
          issues: ['invalidDue'],
          depth: 1,
        },
      ],
      total: 31,
      revision: 12,
    },
  },
  RefreshIndex: {
    request: {
      paths: ['Daily/2026-11-14.md', 'Projects/Home.md'],
      renamed: [{ from: 'Inbox.md', to: 'Projects/Inbox.md' }],
    },
    response: {},
  },
  WatchTasks: { request: {}, response: { revision: 12, isReady: true } },
}

describe('todos service contract', () => {
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
