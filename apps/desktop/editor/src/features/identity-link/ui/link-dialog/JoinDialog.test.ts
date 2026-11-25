/** The dialog of a new device: the camera, the pasted link without one, and the account. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createIdentityStore, type LinkInfo } from '@/entities/identity'
import { createFakeCamera } from '@/testing/fakeCamera'
import {
  click,
  dispose,
  enterText,
  findButton,
  findButtons,
  getScreen,
  textInputs,
} from '@/testing/dom'
import {
  createIdentityClient,
  createLinkInfo,
  type IdentityCalls,
  wrapIdentityClient,
} from '@/testing/identityClient'
import JoinDialog from './JoinDialog.vue'

let wrapper: VueWrapper | undefined
afterEach(() => {
  vi.useRealTimers()
  vi.restoreAllMocks()
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

describe('the dialog that joins an identity', () => {
  it('has a short title and asks for the camera first', async () => {
    await mountDialog(createLinkInfo({ role: 'joiner' }))
    expect(getScreen().get('h2').text()).toBe('Use existing')
    expect(getScreen().text()).toContain('Paste the link shown on your other device.')
    expect(getScreen().text()).not.toContain('Scan the code')
    expect(getScreen().find('label').exists()).toBe(false)
    expect(getScreen().get('input').attributes('aria-label')).toBe('Link from the other device')
  })

  it('offers no field and no way to paste where the machine has a camera', async () => {
    await mountDialog(createLinkInfo({ role: 'joiner' }), {}, true)
    expect(getScreen().find('input').exists()).toBe(false)
    expect(findButtons(/paste|^Use link$/i)).toHaveLength(0)
    expect(findButtons(/^Cancel$|^Close$/)).toHaveLength(1)
  })

  it('shows the camera and says where to point it', async () => {
    await mountDialog(createLinkInfo({ role: 'joiner' }), {}, true)
    expect(getScreen().find('video').exists()).toBe(true)
    expect(getScreen().text()).toContain('Scan the code shown on your other device.')
  })

  it('says that the camera is checked while the check runs', async () => {
    const client = createIdentityClient({})
    const store = createIdentityStore(wrapIdentityClient(client), vi.fn())
    wrapper = mount(JoinDialog, {
      props: { calls: store.devices.link, findCamera: () => new Promise<boolean>(() => undefined) },
      attachTo: document.body,
    })
    await flushPromises()
    expect(getScreen().find('video').exists()).toBe(false)
    expect(getScreen().get('[role="status"]').text()).toContain('Checking the camera')
  })

  it('shows the field to paste the link when the machine never answers the check of the camera', async () => {
    vi.useFakeTimers()
    try {
      const client = createIdentityClient({})
      const store = createIdentityStore(wrapIdentityClient(client), vi.fn())
      wrapper = mount(JoinDialog, {
        props: {
          calls: store.devices.link,
          findCamera: () => new Promise<boolean>(() => undefined),
        },
        attachTo: document.body,
      })
      await vi.advanceTimersByTimeAsync(2500)
      expect(textInputs()).toHaveLength(1)
      expect(getScreen().find('video').exists()).toBe(false)
    } finally {
      vi.useRealTimers()
    }
  })

  it('scans again from a button after a code that is not a link', async () => {
    vi.useFakeTimers()
    const eye = createFakeCamera()
    const startLink = vi
      .fn()
      .mockResolvedValueOnce({ ok: false, error: { error: 'linkTextInvalid', message: 'x' } })
      .mockResolvedValue(await ok(createLinkInfo({ phase: 'waitingForAccept', mode: 'scan' })))
    const client = createIdentityClient({ startLink })
    const store = createIdentityStore(wrapIdentityClient(client), vi.fn())
    wrapper = mount(JoinDialog, {
      props: {
        calls: store.devices.link,
        findCamera: () => Promise.resolve(true),
        camera: eye.camera,
      },
      attachTo: document.body,
    })
    await flushPromises()
    eye.show('https://example.com')
    await eye.waitFrames(1)
    expect(getScreen().get('[role="alert"]').text()).toBe('This text is not the link of a QR code.')
    expect(eye.camera).toHaveBeenCalledOnce()

    await click(findButton(/^Try again$/))
    await flushPromises()
    expect(eye.camera).toHaveBeenCalledTimes(2)
    expect(findButtons(/^Try again$/)).toHaveLength(0)
    expect(getScreen().find('[role="alert"]').exists()).toBe(false)

    eye.show('numen://link?v=1')
    await eye.waitFrames(1)
    expect(startLink).toHaveBeenLastCalledWith('joiner', 'scan', 'numen://link?v=1')
  })

  it('shows the field and the paste hint when the camera is not allowed', async () => {
    await mountDialog(createLinkInfo({ role: 'joiner' }), {}, true, () =>
      Promise.reject(new DOMException('no', 'NotAllowedError')),
    )
    expect(getScreen().find('video').exists()).toBe(false)
    expect(getScreen().text()).toContain('Paste the link shown on your other device.')
    expect(getScreen().text()).not.toContain('not allowed')
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
    await mountWithLink(link)
    expect(getScreen().text()).toContain(link.deviceName)
    expect(getScreen().text()).toContain(link.deviceFingerprint)
  })

  it('shows a refused link and stays on the form', async () => {
    const startLink = () =>
      Promise.resolve({
        ok: false as const,
        error: { error: 'linkTextInvalid' as const, message: 'x' },
      })
    await mountDialog(createLinkInfo(), { startLink })
    await enterText(textInputs()[0] as HTMLInputElement, 'nonsense')
    await click(findButton(/^Use link$/))
    expect(getScreen().get('[role="alert"]').text()).toBe('This text is not the link of a QR code.')
  })

  it('asks the new device to join the identity and passes the answer on', async () => {
    const link = createLinkInfo({
      phase: 'confirmAccount',
      role: 'joiner',
      accountFingerprint: '3F2A 91C0',
    })
    const confirmJoin = vi.fn(() => ok(undefined))
    await mountWithLink(link, { confirmJoin })
    expect(getScreen().text()).toContain('Use identity 3F2A 91C0 on this device?')
    await click(findButton(/^Use identity$/))
    expect(confirmJoin).toHaveBeenCalledWith(true)
  })

  it('emits linked once the link is finished', async () => {
    const dialog = await mountWithLink(createLinkInfo({ phase: 'linked', role: 'joiner' }))
    expect(getScreen().text()).toContain('The device is linked')
    expect(dialog.emitted('linked')).toHaveLength(1)
  })
})
