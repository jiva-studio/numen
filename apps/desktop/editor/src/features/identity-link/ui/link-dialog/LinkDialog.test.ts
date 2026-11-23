/** The dialog of a link: the choice, the QR code, the new device, the code and the account, against the real store. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { IdentityClient } from '@numen/core-client'
import { createIdentityStore, type LinkInfo, type LinkRole } from '@/entities/identity'
import { click, dispose, enterText, findButton, textInputs } from '@/testing/dom'
import { createIdentityClient, createLinkInfo, wrapIdentityClient } from '@/testing/identityClient'
import LinkDialog from './LinkDialog.vue'

let wrapper: VueWrapper | undefined
afterEach(() => {
  if (wrapper !== undefined) dispose(wrapper)
  wrapper = undefined
})

const ok = <T>(value: T) => Promise.resolve({ ok: true as const, value })

const mountDialog = async (
  role: LinkRole,
  link: LinkInfo,
  overrides: Partial<IdentityClient> = {},
) => {
  const client = createIdentityClient({
    startLink: () => ok(link),
    getLink: () => ok(link),
    ...overrides,
  })
  const store = createIdentityStore(wrapIdentityClient(client), vi.fn())
  wrapper = mount(LinkDialog, {
    props: { role, calls: store.devices.link },
    attachTo: document.body,
  })
  await flushPromises()
  return wrapper
}

describe('the link dialog', () => {
  it('offers to show or to scan a QR code first', async () => {
    const dialog = await mountDialog('host', createLinkInfo())
    expect(dialog.text()).toContain('Show a QR code')
    expect(dialog.text()).toContain('Scan a QR code')
  })

  it('shows the QR code, its text and the account fingerprint of a device that has the account', async () => {
    const link = createLinkInfo()
    const dialog = await mountDialog('host', link)
    await click(findButton(/^Show a QR code$/))
    expect(dialog.get('svg[role="img"]').attributes('aria-label')).toBe('QR code')
    expect(dialog.get('input[readonly]').element).toHaveProperty('value', link.qrText)
    expect(dialog.text()).toContain(link.accountFingerprint)
  })

  it('shows the name and the fingerprint of a new device that waits to join', async () => {
    const link = createLinkInfo({ role: 'joiner', mode: 'show', accountFingerprint: '' })
    const dialog = await mountDialog('joiner', link)
    await click(findButton(/^Show a QR code$/))
    expect(dialog.text()).toContain(link.deviceName)
    expect(dialog.text()).toContain(link.deviceFingerprint)
  })

  it('starts as the scanner with the text pasted', async () => {
    const startLink = vi.fn(() => ok(createLinkInfo({ phase: 'waitingForAccept', mode: 'scan' })))
    await mountDialog('host', createLinkInfo(), { startLink })
    await click(findButton(/^Scan a QR code$/))
    await enterText(textInputs()[0] as HTMLInputElement, 'numen://link?v=1')
    await click(findButton(/^Use link$/))
    expect(startLink).toHaveBeenCalledWith('host', 'scan', 'numen://link?v=1')
  })

  it('shows a refused text and stays on the form', async () => {
    const startLink = () =>
      Promise.resolve({
        ok: false as const,
        error: { error: 'linkTextInvalid' as const, message: 'x' },
      })
    const dialog = await mountDialog('host', createLinkInfo(), { startLink })
    await click(findButton(/^Scan a QR code$/))
    await enterText(textInputs()[0] as HTMLInputElement, 'nonsense')
    await click(findButton(/^Use link$/))
    expect(dialog.get('[role="alert"]').text()).toBe('This text is not the link of a QR code.')
  })

  it('shows the new device and passes the approval on', async () => {
    const link = createLinkInfo({
      phase: 'confirmPeer',
      peerName: 'Phone',
      peerFingerprint: '1111 2222',
      code: '482913',
    })
    const decideLink = vi.fn(() => ok(undefined))
    const dialog = await mountDialog('host', link, { decideLink })
    await click(findButton(/^Show a QR code$/))
    expect(dialog.text()).toContain('Phone')
    expect(dialog.text()).toContain('1111 2222')
    expect(dialog.text()).toContain('482913')
    await click(findButton(/^Link this device$/))
    expect(decideLink).toHaveBeenCalledWith(true)
  })

  it('passes the decline on', async () => {
    const link = createLinkInfo({
      phase: 'confirmPeer',
      peerName: 'Phone',
      peerFingerprint: '1111 2222',
    })
    const decideLink = vi.fn(() => ok(undefined))
    await mountDialog('host', link, { decideLink })
    await click(findButton(/^Show a QR code$/))
    await click(findButton(/^Decline$/))
    expect(decideLink).toHaveBeenCalledWith(false)
  })

  it('waits for the code once the new device was approved', async () => {
    const link = createLinkInfo({
      phase: 'confirmPeer',
      isApproved: true,
      peerName: 'Phone',
      peerFingerprint: '1111 2222',
    })
    const dialog = await mountDialog('host', link)
    await click(findButton(/^Show a QR code$/))
    expect(dialog.text()).toContain('Waiting for the code on the new device.')
    expect(dialog.text()).not.toContain('Link this device')
  })

  it('sends the code the person typed on the new device', async () => {
    const link = createLinkInfo({ phase: 'enterCode', role: 'joiner', mode: 'scan', triesLeft: 2 })
    const enterLinkCode = vi.fn(() => ok(undefined))
    const dialog = await mountDialog('joiner', link, { enterLinkCode })
    await click(findButton(/^Show a QR code$/))
    expect(dialog.text()).toContain('2 tries left.')
    await enterText(textInputs()[0] as HTMLInputElement, '482913')
    await click(findButton(/^Confirm$/))
    expect(enterLinkCode).toHaveBeenCalledWith('482913')
  })

  it('asks the new device to join the account and passes the answer on', async () => {
    const link = createLinkInfo({
      phase: 'confirmAccount',
      role: 'joiner',
      accountFingerprint: '3F2A 91C0',
    })
    const confirmJoin = vi.fn(() => ok(undefined))
    const dialog = await mountDialog('joiner', link, { confirmJoin })
    await click(findButton(/^Show a QR code$/))
    expect(dialog.text()).toContain('Join account 3F2A 91C0?')
    await click(findButton(/^Join$/))
    expect(confirmJoin).toHaveBeenCalledWith(true)
  })

  it('says why a link failed', async () => {
    const link = createLinkInfo({ phase: 'failed', reason: 'tooManyWrongCodes' })
    const dialog = await mountDialog('host', link)
    await click(findButton(/^Show a QR code$/))
    expect(dialog.get('[role="alert"]').text()).toBe('Too many wrong codes. The link is closed.')
  })

  it('cancels the link and notifies the page when it closes', async () => {
    const cancelLink = vi.fn(() => ok(undefined))
    const dialog = await mountDialog('host', createLinkInfo(), { cancelLink })
    await click(findButton(/^Show a QR code$/))
    await dialog.get('[role="dialog"]').trigger('keydown', { key: 'Escape' })
    expect(dialog.emitted('close')).toHaveLength(1)
    dialog.unmount()
    await flushPromises()
    expect(cancelLink).toHaveBeenCalledOnce()
  })
})
