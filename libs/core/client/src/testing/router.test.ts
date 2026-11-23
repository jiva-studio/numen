/**
 * The core answered from a test: the real clients send each call, and only the core is replaced.
 */
import { describe, expect, it } from 'vitest'
import { Code, ConnectError } from '@connectrpc/connect'
import { createSearchRpc, createThemeRpc } from '../clients'
import { createCoreHandlerTransport, createCoreTransport, type CoreAnswers } from './router'

const createClients = (answers: CoreAnswers) => {
  const transport = createCoreTransport(answers)
  return { search: createSearchRpc(transport), theme: createThemeRpc(transport) }
}

describe('a call answered by the test', () => {
  it('reaches the answer with the request the client built', async () => {
    const requests: unknown[] = []
    const { search } = createClients({
      search: {
        searchNames: (request) => {
          requests.push(request)
          return { matches: [{ note: { path: 'Llama.md', title: 'Llama' } }] }
        },
      },
    })

    const answer = await search.searchNames({ query: 'lla', limit: 8 })

    expect(requests).toMatchObject([{ query: 'lla', limit: 8 }])
    expect(answer.matches[0]?.note?.title).toBe('Llama')
  })

  it('rejects with the error the answer threw, code and message kept', async () => {
    const { search } = createClients({
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
    const { search } = createClients({ theme: {} })

    const failure = await search.searchNames({ query: 'lla', limit: 8 }).catch((error) => error)

    expect(ConnectError.from(failure).code).toBe(Code.Unimplemented)
  })

  it('streams each item the answer yields', async () => {
    const { theme } = createClients({
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

describe('handlers named by their method', () => {
  it('answer the call of the service that declares the method', async () => {
    const transport = createCoreHandlerTransport({
      searchNames: () => ({ matches: [{ note: { title: 'Llama' } }] }),
      async *watchThemes() {
        yield { names: ['own/Dusk.css'] }
      },
    })
    const search = createSearchRpc(transport)
    const theme = createThemeRpc(transport)

    const answer = await search.searchNames({ query: 'lla', limit: 8 })
    const changes: string[][] = []
    for await (const change of theme.watchThemes({})) changes.push([...change.names])

    expect(answer.matches[0]?.note?.title).toBe('Llama')
    expect(changes).toEqual([['own/Dusk.css']])
  })

  it('throw where a name is declared by no service', () => {
    expect(() => createCoreHandlerTransport({ searchNamez: () => ({}) })).toThrow(/searchNamez/)
  })
})
