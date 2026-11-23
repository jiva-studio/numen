/**
 * The sharing methods as the dialog and the page call them: the name of the method, the request it
 * sends, and the message of the backend a failed call throws.
 */
import { describe, expect, it } from 'vitest'
import { createApi } from '@numen/plugin-sdk'
import { createSharesClient } from './shares'
import { SERVICE } from './vaults'

interface Sent {
  readonly url: string
  readonly body: unknown
}

const createClient = (respond: () => Response) => {
  const sent: Sent[] = []
  const client = createSharesClient(
    createApi(SERVICE, {
      origin: 'http://window',
      fetch: async (url, init) => {
        sent.push({ url: String(url), body: JSON.parse(String(init?.body)) })
        return respond()
      },
    }),
  )
  return { client, sent }
}

describe('the client of the sharing methods', () => {
  it('adds a reader by the code, the label and the folder', async () => {
    const { client, sent } = createClient(() =>
      Response.json({ invitation: 'numen-invitation-1.xyz', fingerprint: 'CCCC DDDD' }),
    )

    const added = await client.addReader({ vaultId: 'v1', folder: 'Food', code: 'c', label: 'Anna' })

    expect(added).toStrictEqual({ invitation: 'numen-invitation-1.xyz', fingerprint: 'CCCC DDDD' })
    expect(sent.at(0)?.url).toMatch(/AddReader$/)
    expect(sent.at(0)?.body).toStrictEqual({ vaultId: 'v1', folder: 'Food', code: 'c', label: 'Anna' })
  })

  it('lists the shares received', async () => {
    const share = { vaultId: 'r1', name: 'Food', ownerFingerprint: 'E', state: 'ready', lastSyncedAt: '' }
    const { client } = createClient(() => Response.json({ shares: [share] }))

    expect(await client.listReceived()).toStrictEqual([share])
  })

  it('throws the message of the backend when a call fails', async () => {
    const { client } = createClient(() =>
      Response.json({ code: 'failed_precondition', message: 'This invitation is for another person' }, { status: 400 }),
    )

    await expect(client.acceptInvitation('text')).rejects.toThrow('This invitation is for another person')
  })
})
