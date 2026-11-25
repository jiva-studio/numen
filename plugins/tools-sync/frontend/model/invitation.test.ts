import { describe, expect, it, vi } from 'vitest'
import type { ShareInvitationClient, ShareRequest } from '../api/invitations'
import type { FolderSharesClient, SharedFolder } from '../api/shares'
import { createInvitationView } from './invitation'
import { ReportedError } from '@numen/plugin-sdk'

const TARGET = { vaultId: 'v1', folder: 'Food' }
const PHONE = { label: 'Phone', fingerprint: '1111 2222', state: 'active' } as const
const REQUEST: ShareRequest = {
  requestId: 'r1',
  ...TARGET,
  deviceName: 'Phone',
  fingerprint: '1111 2222',
  code: '482913',
}

interface World {
  alive: boolean
  requests: ShareRequest[]
  shared: SharedFolder[]
}

const createWorld = (): {
  world: World
  invitations: ShareInvitationClient
  shares: FolderSharesClient
} => {
  const world: World = { alive: true, requests: [], shared: [] }
  return {
    world,
    invitations: {
      startShareInvitation: vi.fn(() =>
        Promise.resolve({ id: 'i1', link: 'numen-share:abc', expiresAt: '2026-11-25T12:00:00Z' }),
      ),
      listShareInvitations: vi.fn(() =>
        Promise.resolve(
          world.alive
            ? [{ id: 'i1', folder: 'Food', expiresAt: '2026-11-25T12:00:00Z', isRequested: false }]
            : [],
        ),
      ),
      revokeShareInvitation: vi.fn(() => {
        world.alive = false
        return Promise.resolve()
      }),
      listShareRequests: vi.fn(() => Promise.resolve(world.requests)),
      approveShareRequest: vi.fn(() => {
        world.alive = false
        world.requests = []
        world.shared = [{ ...TARGET, state: 'active', readers: [PHONE] }]
        return Promise.resolve()
      }),
      declineShareRequest: vi.fn(() => {
        world.requests = []
        return Promise.resolve()
      }),
    },
    shares: {
      listShares: vi.fn(() => Promise.resolve(world.shared)),
      removeReader: vi.fn(() => {
        world.shared = [{ ...TARGET, state: 'active', readers: [{ ...PHONE, state: 'removed' }] }]
        return Promise.resolve()
      }),
      stopShare: vi.fn(() => Promise.resolve()),
    },
  }
}

const create = (refreshMs = 10_000) => {
  const { world, invitations, shares } = createWorld()
  const close = vi.fn()
  const view = createInvitationView({
    invitations,
    shares,
    target: TARGET,
    close,
    refreshMs,
    now: () => new Date('2026-11-25T11:50:00Z'),
  })
  return { world, invitations, shares, view, close }
}

describe('the dialog of a folder shared by a link', () => {
  it('has the link and its expiry once it is started, and waits for a device', async () => {
    const { view } = create()

    await view.start()

    expect([view.link, view.expiresAt, view.isLinkAlive, view.request]).toStrictEqual([
      'numen-share:abc',
      '2026-11-25T12:00:00Z',
      true,
      undefined,
    ])
    view.stop()
  })

  it('shows the request of a device for this folder only', async () => {
    const { world, view } = create()
    world.requests = [{ ...REQUEST, folder: 'Drinks' }, REQUEST]

    await view.start()

    expect(view.request?.requestId).toBe('r1')
    view.stop()
  })

  it('lists the device as a reader once it is allowed, and the link is used up', async () => {
    const { world, invitations, view } = create()
    world.requests = [REQUEST]
    await view.start()

    await view.approve()

    expect(invitations.approveShareRequest).toHaveBeenCalledWith('r1', REQUEST.fingerprint)
    expect(view.request).toBeUndefined()
    expect(view.isLinkAlive).toBe(false)
    expect(view.people).toStrictEqual([{ deviceName: 'Phone', fingerprint: '1111 2222' }])
  })

  it('has no reader after the device is declined', async () => {
    const { world, invitations, view } = create()
    world.requests = [REQUEST]
    await view.start()

    await view.decline()

    expect(invitations.declineShareRequest).toHaveBeenCalledWith('r1')
    expect(view.request).toBeUndefined()
    expect(view.people).toStrictEqual([])
    view.stop()
  })

  it('stops the link when it is revoked', async () => {
    const { invitations, view } = create()
    await view.start()

    await view.revoke()

    expect(invitations.revokeShareInvitation).toHaveBeenCalledWith('i1')
    expect(view.isLinkAlive).toBe(false)
  })

  it('removes the access of a device and drops it from the list', async () => {
    const { world, shares, view } = create()
    world.requests = [REQUEST]
    await view.start()
    await view.approve()

    await view.removeAccess(view.people[0] as never)

    expect(shares.removeReader).toHaveBeenCalledWith({ ...TARGET, fingerprint: '1111 2222' })
    expect(view.people).toStrictEqual([])
  })

  it('keeps the words of a failure to start the link', async () => {
    const { invitations, view } = create()
    vi.mocked(invitations.startShareInvitation).mockRejectedValue(new ReportedError('No identity'))

    await view.start()

    expect([view.error, view.isStarting, view.isLinkAlive]).toStrictEqual([
      'No identity',
      false,
      false,
    ])
  })

  it('asks again while the link works, and no more once it is stopped', async () => {
    vi.useFakeTimers()
    const { invitations, shares, view } = create(1000)
    await view.start()

    await vi.advanceTimersByTimeAsync(2000)
    expect(vi.mocked(invitations.listShareInvitations).mock.calls.length).toBeGreaterThan(2)
    view.stop()
    const asked = vi.mocked(shares.listShares).mock.calls.length
    await vi.advanceTimersByTimeAsync(5000)

    expect(vi.mocked(shares.listShares).mock.calls.length).toBe(asked)
    vi.useRealTimers()
  })

  it('stops asking after the asks keep failing', async () => {
    vi.useFakeTimers()
    const { invitations, view } = create(1000)
    await view.start()
    vi.mocked(invitations.listShareInvitations).mockRejectedValue(new ReportedError('No identity'))

    await vi.advanceTimersByTimeAsync(30_000)
    const asked = vi.mocked(invitations.listShareInvitations).mock.calls.length
    await vi.advanceTimersByTimeAsync(30_000)

    expect(vi.mocked(invitations.listShareInvitations).mock.calls.length).toBe(asked)
    expect(asked).toBeLessThanOrEqual(10)
    vi.useRealTimers()
  })
})
