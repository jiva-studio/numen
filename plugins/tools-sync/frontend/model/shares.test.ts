import { describe, expect, it, vi } from 'vitest'
import type { FolderSharesClient, ReceivedSharesClient, SharedFolder } from '../api/shares'
import { createSharesView } from './shares'

const FOOD: SharedFolder = {
  vaultId: 'v1',
  folder: 'Food',
  state: 'active',
  readers: [
    { label: 'Phone', fingerprint: '1111 2222 3333 4444', state: 'active' },
    { label: 'Old tablet', fingerprint: '5555 6666 7777 8888', state: 'removed' },
  ],
}

const createClients = (shared: SharedFolder[] = [FOOD]) => {
  const given: FolderSharesClient = {
    listShares: vi.fn(() => Promise.resolve(shared)),
    removeReader: vi.fn(() => {
      shared[0] = { ...FOOD, readers: [] }
      return Promise.resolve()
    }),
    stopShare: vi.fn(() => {
      shared.length = 0
      return Promise.resolve()
    }),
  }
  const received: ReceivedSharesClient = {
    listReceived: vi.fn(() =>
      Promise.resolve([
        {
          vaultId: 'r1',
          name: 'Plans',
          ownerFingerprint: 'AAAA BBBB CCCC DDDD',
          state: 'ended' as const,
          lastSyncedAt: '',
        },
      ]),
    ),
    leaveShare: vi.fn(() => Promise.resolve()),
  }
  return { given, received, view: createSharesView({ given, received, refreshMs: 10_000 }) }
}

describe('the page of the shared folders', () => {
  it('lists the folders shared by this account with the devices that read them', async () => {
    const { view } = createClients()

    await view.load()

    expect(view.gives).toStrictEqual([
      {
        vaultId: 'v1',
        folder: 'Food',
        isCreating: false,
        devices: [{ fingerprint: '1111 2222 3333 4444', name: 'Device «Phone» (1111 2222 3333…)' }],
      },
    ])
  })

  it('lists the folders shared with this account and how they stand', async () => {
    const { view } = createClients()

    await view.load()

    expect(view.receives).toStrictEqual([
      {
        vaultId: 'r1',
        name: 'Plans',
        ownerShort: 'AAAA BBBB CCCC…',
        stateLabel: 'Sharing of this folder has ended',
        tone: 'alarm',
      },
    ])
  })

  it('removes the access of a device and lists the folders again', async () => {
    const { given, view } = createClients()
    await view.load()

    await view.removeAccess(view.gives[0]!, view.gives[0]!.devices[0]!)

    expect(given.removeReader).toHaveBeenCalledWith({
      vaultId: 'v1',
      folder: 'Food',
      fingerprint: '1111 2222 3333 4444',
    })
    expect(view.gives[0]?.devices).toStrictEqual([])
  })

  it('stops sharing a folder and drops it from the list', async () => {
    const { given, view } = createClients()
    await view.load()

    await view.stopSharing(view.gives[0]!)

    expect(given.stopShare).toHaveBeenCalledWith({ vaultId: 'v1', folder: 'Food' })
    expect(view.gives).toStrictEqual([])
  })

  it('leaves a received folder only after the question is answered', async () => {
    const { received, view } = createClients()
    await view.load()

    view.askLeave(view.receives[0]!)
    expect(received.leaveShare).not.toHaveBeenCalled()
    await view.confirmLeave()

    expect(received.leaveShare).toHaveBeenCalledWith({ vaultId: 'r1' })
    expect(view.pendingLeave).toBeNull()
  })

  it('leaves nothing when the person cancels', async () => {
    const { received, view } = createClients()
    await view.load()

    view.askLeave(view.receives[0]!)
    view.cancelLeave()
    await view.confirmLeave()

    expect(received.leaveShare).not.toHaveBeenCalled()
  })

  it('shows the words of a failed list', async () => {
    const { given, view } = createClients()
    vi.mocked(given.listShares).mockRejectedValue(new Error('No identity'))

    await view.load()

    expect(view.error).toBe('No identity')
  })
})
