import { describe, expect, it } from 'vitest'
import { createApi } from '@numen/plugin-sdk'
import { createFolderSharesClient, createReceivedSharesClient } from './shares'
import { SERVICE } from './vaults'

interface Sent {
  readonly url: string
  readonly body: unknown
}

const createClients = (respond: () => Response) => {
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
  it('lists the folders shared with their readers', async () => {
    const shared = {
      vaultId: 'v1',
      folder: 'Food',
      state: 'active',
      readers: [{ label: 'Phone', fingerprint: 'AAAA', state: 'active' }],
    }
    const { folder, sent } = createClients(() => Response.json({ shares: [shared] }))

    expect(await folder.listShares()).toStrictEqual([shared])
    expect(sent.at(0)?.url).toMatch(/ListShares$/)
  })

  it('removes a reader by the fingerprint and stops a share', async () => {
    const { folder, sent } = createClients(() => Response.json({}))

    await folder.removeReader({ vaultId: 'v1', folder: 'Food', fingerprint: 'AAAA' })
    await folder.stopShare({ vaultId: 'v1', folder: 'Food' })

    expect(sent.map((one) => [one.url.split('/').at(-1), one.body])).toStrictEqual([
      ['RemoveReader', { vaultId: 'v1', folder: 'Food', fingerprint: 'AAAA' }],
      ['StopShare', { vaultId: 'v1', folder: 'Food' }],
    ])
  })

  it('lists the shares received', async () => {
    const share = {
      vaultId: 'r1',
      name: 'Food',
      ownerFingerprint: 'EEEE',
      state: 'ready',
      lastSyncedAt: '',
    }
    const { received } = createClients(() => Response.json({ shares: [share] }))

    expect(await received.listReceived()).toStrictEqual([share])
  })

  it('leaves a received share by its vault', async () => {
    const { received, sent } = createClients(() => Response.json({}))

    await received.leaveShare({ vaultId: 'r1' })

    expect(sent.map((one) => [one.url.split('/').at(-1), one.body])).toStrictEqual([
      ['LeaveShare', { vaultId: 'r1' }],
    ])
  })

  it('throws the message of a failed call', async () => {
    const { folder } = createClients(() =>
      Response.json({ code: 'failed_precondition', message: 'No identity' }, { status: 400 }),
    )

    await expect(folder.stopShare({ vaultId: 'v1', folder: 'Food' })).rejects.toThrow('No identity')
  })
})
