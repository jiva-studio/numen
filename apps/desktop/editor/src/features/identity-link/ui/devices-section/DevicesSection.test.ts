/** The list of devices, the revoke after a question, and the one way to join an account, against the real store. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createIdentityStore, type DeviceInfo } from '@/entities/identity'
import { click, dispose, findButton, findButtons, findText } from '@/testing/dom'
import {
  createDeviceInfo,
  createIdentityClient,
  type IdentityCalls,
  wrapIdentityClient,
} from '@/testing/identityClient'
import DevicesSection from './DevicesSection.vue'

const THIS = createDeviceInfo()
const OTHER = createDeviceInfo({
  name: 'Phone',
  fingerprint: '1111 2222 3333 4444 5555 6666 7777 8888',
  isThis: false,
})

let wrapper: VueWrapper | undefined
afterEach(() => {
  if (wrapper !== undefined) dispose(wrapper)
  wrapper = undefined
})

const mountSection = async (
  devices: readonly DeviceInfo[],
  overrides: Partial<IdentityCalls> = {},
  isLinking = false,
) => {
  const client = createIdentityClient({
    listDevices: () => Promise.resolve({ ok: true, value: devices }),
    ...overrides,
  })
  const store = createIdentityStore(wrapIdentityClient(client), vi.fn())
  wrapper = mount(DevicesSection, {
    props: { calls: store.devices, isLinking },
    attachTo: document.body,
  })
  await flushPromises()
  return wrapper
}

describe('the devices section', () => {
  it('lists the devices of the account and marks this one', async () => {
    const section = await mountSection([THIS, OTHER])
    const rows = section.findAll('li')
    expect(rows).toHaveLength(2)
    expect(rows[0]?.text()).toContain('This device')
    expect(rows[1]?.text()).toContain(OTHER.fingerprint)
  })

  it('offers no revoke on this device and one on another', async () => {
    const section = await mountSection([THIS, OTHER])
    const rows = section.findAll('li')
    expect(rows[0]?.find('button').exists()).toBe(false)
    expect(rows[1]?.find('button').text()).toBe('Revoke')
  })

  it('offers no revoke on a revoked device', async () => {
    const section = await mountSection([THIS, { ...OTHER, isRevoked: true }])
    const rows = section.findAll('li')
    expect(rows[1]?.text()).toContain('Revoked')
    expect(rows[1]?.find('button').exists()).toBe(false)
  })

  it('revokes a device only after the person confirms the question', async () => {
    const revokeDevice = vi.fn(() => Promise.resolve({ ok: true as const, value: undefined }))
    await mountSection([THIS, OTHER], { revokeDevice })
    await click(findButtons(/^Revoke$/)[0] as HTMLElement)
    expect(
      findText(
        'Phone will lose access. New notes will not be readable on it. Its files stay on it.',
      ),
    ).not.toBeNull()
    expect(revokeDevice).not.toHaveBeenCalled()
    await click(findButtons(/^Revoke$/)[1] as HTMLElement)
    expect(revokeDevice).toHaveBeenCalledWith(OTHER.fingerprint)
  })

  it('keeps the device when the person cancels the question', async () => {
    const revokeDevice = vi.fn(() => Promise.resolve({ ok: true as const, value: undefined }))
    await mountSection([THIS, OTHER], { revokeDevice })
    await click(findButton(/^Revoke$/))
    await click(findButton(/^Cancel$/))
    expect(revokeDevice).not.toHaveBeenCalled()
  })

  it('shows the refusal of a revoke', async () => {
    const revokeDevice = () =>
      Promise.resolve({
        ok: false as const,
        error: { error: 'revokeRaced' as const, message: 'x' },
      })
    const section = await mountSection([THIS, OTHER], { revokeDevice })
    await click(findButtons(/^Revoke$/)[0] as HTMLElement)
    await click(findButtons(/^Revoke$/)[1] as HTMLElement)
    expect(section.get('[role="alert"]').text()).toBe(
      'A device was added while you revoked. Check the list.',
    )
  })

  it('draws the link dialog of a host when it is asked to link', async () => {
    const section = await mountSection([THIS], {}, true)
    expect(section.get('[role="dialog"] h2').text()).toBe('Link a device')
  })

  it('draws no link dialog and no link button of its own', async () => {
    const section = await mountSection([THIS])
    expect(section.find('[role="dialog"]').exists()).toBe(false)
    expect(section.text()).not.toContain('Link a new device')
  })
})
