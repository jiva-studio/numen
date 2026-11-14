/** The part of the window the core lists a request of the editor as coming from. */
import { describe, expect, it } from 'vitest'
import { CALLER_HEADER } from '@numen/core-client'
import { createCoreTransport } from '@numen/core-client/testing'
import { createWindowClients } from './clients'

describe('a request of the editor', () => {
  it('names the part of the window that makes it', async () => {
    const callers: (string | null)[] = []
    const transport = createCoreTransport({
      notes: {
        resolveAddresses: (_request, context) => {
          callers.push(context.requestHeader.get(CALLER_HEADER))
          return {}
        },
      },
      search: {
        searchNames: (_request, context) => {
          callers.push(context.requestHeader.get(CALLER_HEADER))
          return {}
        },
      },
    })
    const clients = createWindowClients({ transport, keepalive: transport })

    await clients.notes.resolveAddresses({ text: '[[Peat]]' })
    await clients.search.searchNames({ query: 'Peat', limit: 1 })

    expect(callers).toStrictEqual(['editor', 'search'])
  })
})
