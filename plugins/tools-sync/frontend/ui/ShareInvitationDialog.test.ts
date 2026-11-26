/** The sharer allows a device only after reading its whole fingerprint and confirming that it is the one the person named. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { ShareInvitationClient, ShareRequest } from '../api/invitations'
import type { FolderSharesClient } from '../api/shares'
import { createInvitationView } from '../model/invitation'
import ShareInvitationDialog from './ShareInvitationDialog.vue'

const TARGET = { vaultId: 'v1', folder: 'Recipes' }
const REQUEST: ShareRequest = {
  requestId: 'r1',
  ...TARGET,
  deviceName: 'Phone',
  fingerprint: '1111 2222 3333 4444 5555 6666',
  code: '482913',
}

const createInvitations = (): ShareInvitationClient => ({
  startShareInvitation: vi.fn(() =>
    Promise.resolve({ id: 'i1', link: 'numen-share:x', expiresAt: '2026-11-25T12:00:00Z' }),
  ),
  listShareInvitations: vi.fn(() =>
    Promise.resolve([
      { id: 'i1', folder: 'Recipes', expiresAt: '2026-11-25T12:00:00Z', isRequested: true },
    ]),
  ),
  listShareRequests: vi.fn(() => Promise.resolve([REQUEST])),
  revokeShareInvitation: vi.fn(() => Promise.resolve()),
  approveShareRequest: vi.fn(() => Promise.resolve()),
  declineShareRequest: vi.fn(() => Promise.resolve()),
})

const shares: FolderSharesClient = {
  listShares: () => Promise.resolve([]),
  removeReader: () => Promise.resolve(),
  stopShare: () => Promise.resolve(),
}

async function open() {
  const invitations = createInvitations()
  const view = createInvitationView({
    invitations,
    shares,
    target: TARGET,
    close: () => undefined,
    now: () => new Date('2026-11-25T10:00:00Z'),
    refreshMs: 1_000_000,
  })
  await view.start()
  mount(ShareInvitationDialog, { props: { state: view }, attachTo: document.body })
  await flushPromises()
  return invitations
}

const getApprove = (): HTMLButtonElement => {
  const found = [...document.body.querySelectorAll('button')].find(
    (one) => one.textContent.trim() === 'Approve',
  )
  if (found === undefined) throw new Error('no button Approve')
  return found
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('ShareInvitationDialog', () => {
  it('shows the whole fingerprint of the device that asks and says that the code does not name the person', async () => {
    await open()

    expect(document.body.textContent).toContain(REQUEST.fingerprint)
    expect(document.body.textContent).toContain(
      'The code only shows the person has the link. Ask the person to read you their fingerprint from Numen, and allow only if it is the same as this one.',
    )
  })

  it('keeps Approve off until the person confirms the fingerprint', async () => {
    const invitations = await open()
    expect(getApprove().disabled).toBe(true)

    const box = document.body.querySelector<HTMLInputElement>('input[type="checkbox"]')
    box?.click()
    await flushPromises()
    getApprove().click()
    await flushPromises()

    expect(invitations.approveShareRequest).toHaveBeenCalledExactlyOnceWith(
      'r1',
      REQUEST.fingerprint,
    )
  })
})
