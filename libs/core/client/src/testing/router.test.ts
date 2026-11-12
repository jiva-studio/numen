/**
 * The core answered from a test: the real clients send each call, and only the core is replaced.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import { search, theme } from '../clients'
import { installCoreAnswers, installCoreHandlers } from './router'

let restore: (() => void) | undefined

afterEach(() => {
  restore?.()
  restore = undefined
})

describe('a call answered by the test', () => {
  it('reaches the answer with the request the client built', async () => {
    const requests: unknown[] = []
    restore = installCoreAnswers({
      search: {
        searchNames: (request) => {
          requests.push(request)
          return { found: [{ note: { path: 'Llama.md', title: 'Llama' } }] }
        },
      },
    })

    const answer = await search.searchNames({ query: 'lla', limit: 8 })

    expect(requests).toMatchObject([{ query: 'lla', limit: 8 }])
    expect(answer.found[0]?.note?.title).toBe('Llama')
  })

  it('rejects with the error the answer threw, code and message kept', async () => {
    restore = installCoreAnswers({
      search: {
        searchNames: () => {
          throw new ConnectError('index is rebuilding', Code.Unavailable)
        },
      },
    })

    const failure = await search.searchNames({ query: 'lla', limit: 8 }).catch((error) => error)

    expect(ConnectError.from(failure).code).toBe(Code.Unavailable)
    expect(ConnectError.from(failure).rawMessage).toBe('index is rebuilding')
  })

  it('rejects as unimplemented where the test left the call unanswered', async () => {
    restore = installCoreAnswers({ theme: {} })

    const failure = await search.searchNames({ query: 'lla', limit: 8 }).catch((error) => error)

    expect(ConnectError.from(failure).code).toBe(Code.Unimplemented)
  })

  it('streams each item the answer yields', async () => {
    restore = installCoreAnswers({
      theme: {
        async *watchThemes() {
          yield { names: ['own/Dusk.css'] }
          yield { names: ['own/Dawn.css'] }
        },
      },
    })

    const changes: string[][] = []
    for await (const change of theme.watchThemes({})) changes.push([...change.names])

    expect(changes).toEqual([['own/Dusk.css'], ['own/Dawn.css']])
  })
})

describe('the answers installed', () => {
  it('give way to the ones installed before when restored', async () => {
    const restoreOuter = installCoreAnswers({
      search: { searchNames: () => ({ found: [{ note: { title: 'Outer' } }] }) },
    })
    const restoreInner = installCoreAnswers({
      search: { searchNames: () => ({ found: [{ note: { title: 'Inner' } }] }) },
    })
    try {
      expect((await search.searchNames({ query: 'a', limit: 1 })).found[0]?.note?.title).toBe(
        'Inner',
      )
      restoreInner()
      expect((await search.searchNames({ query: 'a', limit: 1 })).found[0]?.note?.title).toBe(
        'Outer',
      )
    } finally {
      restoreOuter()
    }
  })
})

describe('handlers named by their method', () => {
  it('answer the call of the service that declares the method', async () => {
    restore = installCoreHandlers({
      searchNames: () => ({ found: [{ note: { title: 'Llama' } }] }),
      async *watchThemes() {
        yield { names: ['own/Dusk.css'] }
      },
    })

    const answer = await search.searchNames({ query: 'lla', limit: 8 })
    const changes: string[][] = []
    for await (const change of theme.watchThemes({})) changes.push([...change.names])

    expect(answer.found[0]?.note?.title).toBe('Llama')
    expect(changes).toEqual([['own/Dusk.css']])
  })

  it('throw where a name is declared by no service', () => {
    expect(() => installCoreHandlers({ searchNamez: () => ({}) })).toThrow(/searchNamez/)
  })
})
