/** The dialog of a device that has the account: the QR code, the new device, the code and the end. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createIdentityStore, type LinkInfo } from '@/entities/identity'
import { click, dispose, findButton, findButtons, getScreen } from '@/testing/dom'
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
    await mountDialog(createLinkInfo())
    expect(getScreen().get('h2').text()).toBe('Link a device')
    expect(getScreen().findAll('ol li')).toHaveLength(3)
    expect(getScreen().get('svg[role="img"]').attributes('aria-label')).toBe('QR code')
    expect(getScreen().get('ol li[data-state="active"]').text()).toContain('choose Use existing')
    expect(getScreen().find('[role="status"]').exists()).toBe(false)
  })

  it('shows neither the link as a field nor the fingerprint of the identity', async () => {
    const link = createLinkInfo()
    await mountDialog(link)
    expect(getScreen().find('input').exists()).toBe(false)
    expect(getScreen().text()).not.toContain(link.accountFingerprint)
    expect(getScreen().text()).not.toContain(link.qrText)
  })

  it('copies the link with one button', async () => {
    const link = createLinkInfo()
    const writeText = vi.fn(() => Promise.resolve())
    vi.stubGlobal('navigator', { clipboard: { writeText } })
    await mountDialog(link)
    await click(findButton(/^Copy link$/))
    expect(writeText).toHaveBeenCalledWith(link.qrText)
    expect(getScreen().text()).toContain('Copied')
    vi.unstubAllGlobals()
  })

  it('keeps the place of the QR code while the channel is awaited', async () => {
    await mountDialog(createLinkInfo({ phase: 'waitingForChannel', qrText: '' }))
    expect(getScreen().find('svg[role="img"]').exists()).toBe(false)
    expect(getScreen().findAll('ol li[data-state="active"]')).toHaveLength(0)
    expect(findButton(/^Copy link$/).hasAttribute('disabled')).toBe(true)
  })

  it('shows the new device and passes the approval on', async () => {
    const link = createLinkInfo({
      phase: 'confirmPeer',
      peerName: 'Phone',
      peerFingerprint: '1111 2222',
    })
    const decideLink = vi.fn(() => ok(undefined))
    await mountDialog(link, { decideLink })
    expect(getScreen().text()).toContain('Phone')
    expect(getScreen().text()).toContain('1111 2222')
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

  it('checks the steps once linked and keeps the same three', async () => {
    await mountDialog(createLinkInfo({ phase: 'linked' }))
    expect(getScreen().findAll('ol li[data-state="done"]')).toHaveLength(3)
  })

  it('ends with one button, Done, once the device is linked', async () => {
    await mountDialog(createLinkInfo({ phase: 'linked' }))
    expect(findButton(/^Done$/)).toBeTruthy()
    expect(findButtons(/^Cancel$|^Close$|^Link device$/)).toHaveLength(0)
  })

  it('says why a link failed and offers to try again', async () => {
    await mountDialog(createLinkInfo({ phase: 'failed', reason: 'staleChain' }))
    expect(getScreen().get('[role="alert"]').text()).toContain(
      'The identity changed while linking. Try again.',
    )
    expect(findButton(/^Try again$/)).toBeTruthy()
  })

  it('cancels the link and notifies the page when it closes', async () => {
    const cancelLink = vi.fn(() => ok(undefined))
    const dialog = await mountDialog(createLinkInfo(), { cancelLink })
    await getScreen().get('[role="dialog"]').trigger('keydown', { key: 'Escape' })
    expect(dialog.emitted('close')).toHaveLength(1)
    dialog.unmount()
    await flushPromises()
    expect(cancelLink).toHaveBeenCalledOnce()
  })
})
