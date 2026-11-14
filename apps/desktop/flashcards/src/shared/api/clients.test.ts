/** The part of the application the core lists a request of this window as coming from. */
import { describe, expect, it } from 'vitest'
import { CALLER_HEADER } from '@numen/core-client'
import { createCoreTransport } from '@numen/core-client/testing'
import { createWindowClients } from './clients'

describe('a request of the window', () => {
  it('names no part of another window as its caller', async () => {
    const callers: (string | null)[] = []
    const clients = createWindowClients(
      createCoreTransport({
        notes: {
          resolveAddresses: (_request, context) => {
            callers.push(context.requestHeader.get(CALLER_HEADER))
            return {}
          },
        },
      }),
    )

    await clients.notes.resolveAddresses({ text: '[[Peat]]' })

    expect(callers).toStrictEqual([null])
  })
})
