/** The buttons that only close the dialog "Join folder" never submit its form. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import type { AcceptInvitationClient } from '../api/invitations'
import { createAcceptanceView } from '../model/acceptance'
import AcceptInvitationDialog from './AcceptInvitationDialog.vue'

const PREVIEW = {
  folder: 'Recipes',
  ownerFingerprint: '0F3A9C2177B0D4E6',
  expiresAt: '2026-11-25T12:00:00Z',
  isReadOnly: true,
}

const createClient = (): AcceptInvitationClient => ({
  readSharePreview: vi.fn(() => Promise.resolve(PREVIEW)),
  acceptShareInvitation: vi.fn(() =>
    Promise.resolve({
      requestId: 'r1',
      folder: 'Recipes',
      ownerFingerprint: PREVIEW.ownerFingerprint,
    }),
  ),
  getShareInvitationStatus: vi.fn(() =>
    Promise.resolve({
      phase: 'waiting' as const,
      folder: 'Recipes',
      ownerFingerprint: PREVIEW.ownerFingerprint,
      expiresAt: PREVIEW.expiresAt,
    }),
  ),
})

const getButton = (text: string): HTMLButtonElement => {
  const found = [...document.body.querySelectorAll('button')].find((one) =>
    one.textContent.includes(text),
  )
  if (found === undefined) throw new Error(`no button ${text}`)
  return found
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('AcceptInvitationDialog', () => {
  it('closes without accepting the share when the person declines it', async () => {
    const client = createClient()
    const close = vi.fn()
    const view = createAcceptanceView({ client, close })
    await view.setLink('numen-share:abc')
    const wrapper = mount(AcceptInvitationDialog, { attachTo: document.body, props: { view } })

    getButton('Decline').click()
    await wrapper.vm.$nextTick()

    expect(close).toHaveBeenCalledTimes(1)
    expect(client.acceptShareInvitation).not.toHaveBeenCalled()
  })

  it('accepts the share when the person presses the accept button', async () => {
    const client = createClient()
    const view = createAcceptanceView({ client, close: vi.fn() })
    await view.setLink('numen-share:abc')
    const wrapper = mount(AcceptInvitationDialog, { attachTo: document.body, props: { view } })

    getButton('Accept').click()
    await wrapper.vm.$nextTick()

    expect(client.acceptShareInvitation).toHaveBeenCalledTimes(1)
    view.stop()
  })
})
