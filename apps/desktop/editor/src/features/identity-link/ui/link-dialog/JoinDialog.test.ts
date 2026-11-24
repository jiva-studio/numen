/** The dialog of a new device: the camera, the pasted link on request, the code and the account. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createIdentityStore, type LinkInfo } from '@/entities/identity'
import { click, dispose, enterText, findButton, findButtons, textInputs } from '@/testing/dom'
import {
  createIdentityClient,
  createLinkInfo,
  type IdentityCalls,
  wrapIdentityClient,
} from '@/testing/identityClient'
import JoinDialog from './JoinDialog.vue'

let wrapper: VueWrapper | undefined
afterEach(() => {
  if (wrapper !== undefined) dispose(wrapper)
  wrapper = undefined
})

const ok = <T>(value: T) => Promise.resolve({ ok: true as const, value })

const mountDialog = async (
  link: LinkInfo,
  overrides: Partial<IdentityCalls> = {},
  hasCamera = false,
  camera: () => Promise<MediaStream> = () => new Promise<MediaStream>(() => undefined),
) => {
  const client = createIdentityClient({
    startLink: () => ok(link),
    getLink: () => ok(link),
    ...overrides,
  })
  const store = createIdentityStore(wrapIdentityClient(client), vi.fn())
  wrapper = mount(JoinDialog, {
    props: { calls: store.devices.link, findCamera: () => Promise.resolve(hasCamera), camera },
    attachTo: document.body,
  })
  await flushPromises()
  return wrapper
}

const mountWithLink = async (link: LinkInfo, overrides: Partial<IdentityCalls> = {}) => {
  const dialog = await mountDialog(link, overrides)
  await enterText(textInputs()[0] as HTMLInputElement, 'numen://link?v=1')
  await click(findButton(/^Use link$/))
  return dialog
}

describe('the dialog that joins an account', () => {
  it('has a short title and asks for the camera first', async () => {
    const dialog = await mountDialog(createLinkInfo({ role: 'joiner' }))
    expect(dialog.get('h2').text()).toBe('Join account')
    expect(dialog.get('[role="status"]').text()).toBe(
      'No camera found. Paste the link from the other device.',
    )
  })

  it('offers no field until the person asks for it where the machine has a camera', async () => {
    const dialog = await mountDialog(createLinkInfo({ role: 'joiner' }), {}, true)
    expect(dialog.find('input').exists()).toBe(false)
    expect(findButtons(/^Use link$/)).toHaveLength(0)
    await click(findButton(/^Paste link instead$/))
    expect(dialog.find('input').exists()).toBe(true)
    expect(findButton(/^Use link$/)).toBeTruthy()
  })


  it('shows the camera and says where to point it', async () => {
    const dialog = await mountDialog(createLinkInfo({ role: 'joiner' }), {}, true)
    expect(dialog.find('video').exists()).toBe(true)
    expect(dialog.text()).toContain('Point the camera at the code on your other device.')
  })

  it('says that the camera is checked while the check runs', async () => {
    const client = createIdentityClient({})
    const store = createIdentityStore(wrapIdentityClient(client), vi.fn())
    wrapper = mount(JoinDialog, {
      props: { calls: store.devices.link, findCamera: () => new Promise<boolean>(() => undefined) },
      attachTo: document.body,
    })
    await flushPromises()
    expect(wrapper.find('video').exists()).toBe(false)
    expect(wrapper.get('[role="status"]').text()).toContain('Checking the camera')
  })

  it('shows the field and the reason when the camera is not allowed', async () => {
    const dialog = await mountDialog(createLinkInfo({ role: 'joiner' }), {}, true, () =>
      Promise.reject(new DOMException('no', 'NotAllowedError')),
    )
    expect(dialog.find('video').exists()).toBe(false)
    expect(dialog.get('[role="status"]').text()).toBe(
      'The camera is not allowed. Paste the link from the other device.',
    )
    expect(textInputs()).toHaveLength(1)
  })
  it('starts as the scanner with the link pasted', async () => {
    const startLink = vi.fn(() => ok(createLinkInfo({ phase: 'waitingForAccept', mode: 'scan' })))
    await mountWithLink(createLinkInfo(), { startLink })
    expect(startLink).toHaveBeenCalledWith('joiner', 'scan', 'numen://link?v=1')
  })

  it('shows the name and the fingerprint of a new device that waits to join', async () => {
    const link = createLinkInfo({
      role: 'joiner',
      phase: 'waitingForAccept',
      mode: 'scan',
      accountFingerprint: '',
    })
    const dialog = await mountWithLink(link)
    expect(dialog.text()).toContain(link.deviceName)
    expect(dialog.text()).toContain(link.deviceFingerprint)
  })

  it('shows a refused link and stays on the form', async () => {
    const startLink = () =>
      Promise.resolve({
        ok: false as const,
        error: { error: 'linkTextInvalid' as const, message: 'x' },
      })
    const dialog = await mountDialog(createLinkInfo(), { startLink })
    await enterText(textInputs()[0] as HTMLInputElement, 'nonsense')
    await click(findButton(/^Use link$/))
    expect(dialog.get('[role="alert"]').text()).toBe('This text is not the link of a QR code.')
  })

  it('sends the code the person typed from the other device', async () => {
    const link = createLinkInfo({ phase: 'enterCode', role: 'joiner', mode: 'scan', triesLeft: 2 })
    const enterLinkCode = vi.fn(() => ok(undefined))
    const dialog = await mountWithLink(link, { enterLinkCode })
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
    const dialog = await mountWithLink(link, { confirmJoin })
    expect(dialog.text()).toContain('Join account 3F2A 91C0?')
    await click(findButton(/^Join$/))
    expect(confirmJoin).toHaveBeenCalledWith(true)
  })
})
