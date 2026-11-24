/** The dialog of a link: the steps, the QR code, the new device, the code and the account, against the real store. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createIdentityStore, type LinkInfo, type LinkRole } from '@/entities/identity'
import { click, dispose, enterText, findButton, textInputs } from '@/testing/dom'
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

const mountDialog = async (
  role: LinkRole,
  link: LinkInfo,
  overrides: Partial<IdentityCalls> = {},
) => {
  const client = createIdentityClient({
    startLink: () => ok(link),
    getLink: () => ok(link),
    ...overrides,
  })
  const store = createIdentityStore(wrapIdentityClient(client), vi.fn())
  wrapper = mount(LinkDialog, {
    props: { role, calls: store.devices.link, findCamera: () => Promise.resolve(false) },
    attachTo: document.body,
  })
  await flushPromises()
  return wrapper
}

const mountJoined = async (link: LinkInfo, overrides: Partial<IdentityCalls> = {}) => {
  const dialog = await mountDialog('joiner', link, overrides)
  await enterText(textInputs()[0] as HTMLInputElement, 'numen://link?v=1')
  await click(findButton(/^Use link$/))
  return dialog
}

describe('the link dialog', () => {
  it('names what it does and lists the steps for a device that has the account', async () => {
    const dialog = await mountDialog('host', createLinkInfo())
    expect(dialog.get('h2').text()).toBe('Link a new device to your account')
    expect(dialog.findAll('ol li')).toHaveLength(3)
  })

  it('shows the QR code, its text and the account fingerprint of a device that has the account', async () => {
    const link = createLinkInfo()
    const dialog = await mountDialog('host', link)
    expect(dialog.get('svg[role="img"]').attributes('aria-label')).toBe('QR code')
    expect(dialog.get('input[readonly]').element).toHaveProperty('value', link.qrText)
    expect(dialog.text()).toContain(link.accountFingerprint)
    expect(dialog.text()).toContain('Waiting for the other device to scan')
  })

  it('says that the link is prepared while the channel is awaited', async () => {
    const dialog = await mountDialog('host', createLinkInfo({ phase: 'waitingForChannel' }))
    expect(dialog.get('[role="status"]').text()).toContain('Preparing the link')
    expect(dialog.find('svg[role="img"]').exists()).toBe(false)
  })

  it('names the join of a new device and asks for the camera first', async () => {
    const dialog = await mountDialog('joiner', createLinkInfo({ role: 'joiner' }))
    expect(dialog.get('h2').text()).toBe('Join your account on this device')
    expect(dialog.get('[role="status"]').text()).toBe(
      'No camera found: paste the code from the other device.',
    )
  })

  it('starts as the scanner with the text pasted', async () => {
    const startLink = vi.fn(() => ok(createLinkInfo({ phase: 'waitingForAccept', mode: 'scan' })))
    await mountJoined(createLinkInfo(), { startLink })
    expect(startLink).toHaveBeenCalledWith('joiner', 'scan', 'numen://link?v=1')
  })

  it('shows the name and the fingerprint of a new device that waits to join', async () => {
    const link = createLinkInfo({
      role: 'joiner',
      phase: 'waitingForAccept',
      mode: 'scan',
      accountFingerprint: '',
    })
    const dialog = await mountJoined(link)
    expect(dialog.text()).toContain(link.deviceName)
    expect(dialog.text()).toContain(link.deviceFingerprint)
  })

  it('shows a refused text and stays on the form', async () => {
    const startLink = () =>
      Promise.resolve({
        ok: false as const,
        error: { error: 'linkTextInvalid' as const, message: 'x' },
      })
    const dialog = await mountDialog('joiner', createLinkInfo(), { startLink })
    await enterText(textInputs()[0] as HTMLInputElement, 'nonsense')
    await click(findButton(/^Use link$/))
    expect(dialog.get('[role="alert"]').text()).toBe('This text is not the link of a QR code.')
  })

  it('shows the new device and the code large and passes the approval on', async () => {
    const link = createLinkInfo({
      phase: 'confirmPeer',
      peerName: 'Phone',
      peerFingerprint: '1111 2222',
      code: '482913',
    })
    const decideLink = vi.fn(() => ok(undefined))
    const dialog = await mountDialog('host', link, { decideLink })
    expect(dialog.text()).toContain('Phone')
    expect(dialog.text()).toContain('1111 2222')
    expect(dialog.text()).toContain('482 913')
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
    await click(findButton(/^Decline$/))
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
    const dialog = await mountDialog('host', link)
    expect(dialog.text()).toContain('Waiting for the new device')
    expect(dialog.text()).toContain('482 913')
    expect(dialog.text()).not.toContain('Link this device')
  })

  it('sends the code the person typed on the new device', async () => {
    const link = createLinkInfo({ phase: 'enterCode', role: 'joiner', mode: 'scan', triesLeft: 2 })
    const enterLinkCode = vi.fn(() => ok(undefined))
    const dialog = await mountJoined(link, { enterLinkCode })
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
    const dialog = await mountJoined(link, { confirmJoin })
    expect(dialog.text()).toContain('Join account 3F2A 91C0?')
    await click(findButton(/^Join$/))
    expect(confirmJoin).toHaveBeenCalledWith(true)
  })

  it('says that the device is linked', async () => {
    const dialog = await mountDialog('host', createLinkInfo({ phase: 'linked' }))
    expect(dialog.text()).toContain('The device is linked')
  })

  it('says why a link failed and offers to try again', async () => {
    const link = createLinkInfo({ phase: 'failed', reason: 'tooManyWrongCodes' })
    const dialog = await mountDialog('host', link)
    expect(dialog.get('[role="alert"]').text()).toContain('Too many wrong codes. The link is closed.')
    expect(findButton(/^Try again$/)).toBeTruthy()
  })

  it('cancels the link and notifies the page when it closes', async () => {
    const cancelLink = vi.fn(() => ok(undefined))
    const dialog = await mountDialog('host', createLinkInfo(), { cancelLink })
    await dialog.get('[role="dialog"]').trigger('keydown', { key: 'Escape' })
    expect(dialog.emitted('close')).toHaveLength(1)
    dialog.unmount()
    await flushPromises()
    expect(cancelLink).toHaveBeenCalledOnce()
  })
})
