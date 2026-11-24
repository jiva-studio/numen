/** The dialog of a device that has the account: the QR code, the new device, the code and the end. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createIdentityStore, type LinkInfo } from '@/entities/identity'
import { click, dispose, findButton } from '@/testing/dom'
import {
  createIdentityClient,
  createLinkInfo,
  type IdentityCalls,
  wrapIdentityClient,
} from '@/testing/identityClient'
import LinkDialog from './LinkDialog.vue'

let wrapper: VueWrapper | undefined
afterEach(() => {
  if (wrapper !== undefined) dispose(wrapper)
  wrapper = undefined
})

const ok = <T>(value: T) => Promise.resolve({ ok: true as const, value })

const mountDialog = async (link: LinkInfo, overrides: Partial<IdentityCalls> = {}) => {
  const client = createIdentityClient({
    startLink: () => ok(link),
    getLink: () => ok(link),
    ...overrides,
  })
  const store = createIdentityStore(wrapIdentityClient(client), vi.fn())
  wrapper = mount(LinkDialog, { props: { calls: store.devices.link }, attachTo: document.body })
  await flushPromises()
  return wrapper
}

describe('the dialog that links a new device', () => {
  it('has the title Link a device and lists three steps next to the QR code', async () => {
    const dialog = await mountDialog(createLinkInfo())
    expect(dialog.get('h2').text()).toBe('Link a device')
    expect(dialog.findAll('ol li')).toHaveLength(3)
    expect(dialog.get('svg[role="img"]').attributes('aria-label')).toBe('QR code')
    expect(dialog.text()).toContain('Waiting for the other device to scan')
  })

  it('shows neither the link as a field nor the fingerprint of the account', async () => {
    const link = createLinkInfo()
    const dialog = await mountDialog(link)
    expect(dialog.find('input').exists()).toBe(false)
    expect(dialog.text()).not.toContain(link.accountFingerprint)
    expect(dialog.text()).not.toContain(link.qrText)
  })

  it('copies the link with one button', async () => {
    const link = createLinkInfo()
    const writeText = vi.fn(() => Promise.resolve())
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    const dialog = await mountDialog(link)
    await click(findButton(/^Copy link$/))
    expect(writeText).toHaveBeenCalledWith(link.qrText)
    expect(dialog.text()).toContain('Copied')
    vi.unstubAllGlobals()
  })

  it('keeps the place of the QR code and says that the link is prepared while the channel is awaited', async () => {
    const dialog = await mountDialog(createLinkInfo({ phase: 'waitingForChannel', qrText: '' }))
    expect(dialog.find('svg[role="img"]').exists()).toBe(false)
    expect(dialog.find('.link-qr__code').exists()).toBe(true)
    expect(dialog.get('[role="status"]').text()).toContain('Preparing the link')
    expect(findButton(/^Copy link$/).hasAttribute('disabled')).toBe(true)
  })

  it('shows the new device and the code large and passes the approval on', async () => {
    const link = createLinkInfo({
      phase: 'confirmPeer',
      peerName: 'Phone',
      peerFingerprint: '1111 2222',
      code: '482913',
    })
    const decideLink = vi.fn(() => ok(undefined))
    const dialog = await mountDialog(link, { decideLink })
    expect(dialog.text()).toContain('Phone')
    expect(dialog.text()).toContain('1111 2222')
    expect(dialog.text()).toContain('482 913')
    await click(findButton(/^Link device$/))
    expect(decideLink).toHaveBeenCalledWith(true)
  })

  it('passes the decline on', async () => {
    const link = createLinkInfo({
      phase: 'confirmPeer',
      peerName: 'Phone',
      peerFingerprint: '1111 2222',
    })
    const decideLink = vi.fn(() => ok(undefined))
    await mountDialog(link, { decideLink })
    await click(findButton(/^Don't link$/))
    expect(decideLink).toHaveBeenCalledWith(false)
  })

  it('waits for the code once the new device was approved', async () => {
    const link = createLinkInfo({
      phase: 'confirmPeer',
      isApproved: true,
      peerName: 'Phone',
      peerFingerprint: '1111 2222',
      code: '482913',
    })
    const dialog = await mountDialog(link)
    expect(dialog.text()).toContain('Waiting for the new device')
    expect(dialog.text()).toContain('482 913')
    expect(dialog.text()).not.toContain('Link device')
  })

  it('says that the device is linked', async () => {
    const dialog = await mountDialog(createLinkInfo({ phase: 'linked' }))
    expect(dialog.text()).toContain('The device is linked')
  })

  it('says why a link failed and offers to try again', async () => {
    const dialog = await mountDialog(createLinkInfo({ phase: 'failed', reason: 'tooManyWrongCodes' }))
    expect(dialog.get('[role="alert"]').text()).toContain('Too many wrong codes. The link is closed.')
    expect(findButton(/^Try again$/)).toBeTruthy()
  })

  it('cancels the link and notifies the page when it closes', async () => {
    const cancelLink = vi.fn(() => ok(undefined))
    const dialog = await mountDialog(createLinkInfo(), { cancelLink })
    await dialog.get('[role="dialog"]').trigger('keydown', { key: 'Escape' })
    expect(dialog.emitted('close')).toHaveLength(1)
    dialog.unmount()
    await flushPromises()
    expect(cancelLink).toHaveBeenCalledOnce()
  })
})
