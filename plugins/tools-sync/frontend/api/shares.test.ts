/**
 * The sharing methods as the dialog and the page call them: the name of the method, the request it
 * sends, and the message of the backend a failed call throws.
 */
import { describe, expect, it } from 'vitest'
import { createApi } from '@numen/plugin-sdk'
import { createFolderSharesClient, createReceivedSharesClient } from './shares'
import { SERVICE } from './vaults'

interface Sent {
  readonly url: string
  readonly body: unknown
}

const createClient = (respond: () => Response) => {
  const sent: Sent[] = []
  const api = createApi(SERVICE, {
    origin: 'http://window',
    fetch: async (url, init) => {
      sent.push({ url: String(url), body: JSON.parse(String(init?.body)) })
      return respond()
    },
  })
  return { folder: createFolderSharesClient(api), received: createReceivedSharesClient(api), sent }
}

describe('the client of the sharing methods', () => {
  it('adds a reader by the code, the label and the folder', async () => {
    const { folder, sent } = createClient(() =>
      Response.json({ invitation: 'numen-invitation-1.xyz', fingerprint: 'CCCC DDDD' }),
    )

    const added = await folder.addReader({ vaultId: 'v1', folder: 'Food', code: 'c', label: 'Anna' })

    expect(added).toStrictEqual({ invitation: 'numen-invitation-1.xyz', fingerprint: 'CCCC DDDD' })
    expect(sent.at(0)?.url).toMatch(/AddReader$/)
    expect(sent.at(0)?.body).toStrictEqual({ vaultId: 'v1', folder: 'Food', code: 'c', label: 'Anna' })
  })

  it('lists the shares received', async () => {
    const share = { vaultId: 'r1', name: 'Food', ownerFingerprint: 'E', state: 'ready', lastSyncedAt: '' }
    const { received } = createClient(() => Response.json({ shares: [share] }))

    expect(await received.listReceived()).toStrictEqual([share])
  })

  it('throws the message of the backend when a call fails', async () => {
    const { received } = createClient(() =>
      Response.json({ code: 'failed_precondition', message: 'This invitation is for another person' }, { status: 400 }),
    )

    await expect(received.acceptInvitation('text')).rejects.toThrow('This invitation is for another person')
  })
})
