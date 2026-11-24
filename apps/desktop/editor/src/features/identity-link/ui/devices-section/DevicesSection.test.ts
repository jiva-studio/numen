/** The list of devices, the revoke after a question, and the one way to join an account, against the real store. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createIdentityStore, type DeviceInfo } from '@/entities/identity'
import { click, dispose, findButton, findMenuItem, findText } from '@/testing/dom'
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

  it('gives every row the same actions slot and a menu button only on another device', async () => {
    const section = await mountSection([THIS, OTHER])
    const rows = section.findAll('li')
    expect(rows[0]?.find('.device-row__actions').exists()).toBe(true)
    expect(rows[1]?.find('.device-row__actions').exists()).toBe(true)
    expect(rows[0]?.find('button').exists()).toBe(false)
    expect(rows[1]?.find('button').attributes('aria-label')).toBe('More actions for Phone')
    expect(rows[1]?.text()).not.toContain('Remove')
  })

  it('revokes a device only after the person confirms the question', async () => {
    const revokeDevice = vi.fn(() => Promise.resolve({ ok: true as const, value: undefined }))
    await mountSection([THIS, OTHER], { revokeDevice })
    await click(findButton(/^More actions for Phone$/))
    await click(findMenuItem(/^Remove$/))
    expect(
      findText(
        'Phone will lose access. The files on the device stay.',
      ),
    ).not.toBeNull()
    expect(revokeDevice).not.toHaveBeenCalled()
    await click(findButton(/^Remove$/))
    expect(revokeDevice).toHaveBeenCalledWith(OTHER.fingerprint)
  })

  it('drops the device from the list once the person confirms', async () => {
    let listed: readonly DeviceInfo[] = [THIS, OTHER]
    const revokeDevice = () => {
      listed = [THIS]
      return Promise.resolve({ ok: true as const, value: undefined })
    }
    const section = await mountSection([THIS, OTHER], {
      revokeDevice,
      listDevices: () => Promise.resolve({ ok: true, value: listed }),
    })
    expect(section.findAll('li')).toHaveLength(2)
    await click(findButton(/^More actions for Phone$/))
    await click(findMenuItem(/^Remove$/))
    await click(findButton(/^Remove$/))
    await flushPromises()
    expect(section.findAll('li')).toHaveLength(1)
  })

  it('keeps the device when the person cancels the question', async () => {
    const revokeDevice = vi.fn(() => Promise.resolve({ ok: true as const, value: undefined }))
    await mountSection([THIS, OTHER], { revokeDevice })
    await click(findButton(/^More actions for Phone$/))
    await click(findMenuItem(/^Remove$/))
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
    await click(findButton(/^More actions for Phone$/))
    await click(findMenuItem(/^Remove$/))
    await click(findButton(/^Remove$/))
    expect(section.get('[role="alert"]').text()).toBe(
      'A device was added while you revoked. Check the list.',
    )
  })

  it('draws the link dialog when isLinking is set', async () => {
    const section = await mountSection([THIS], {}, true)
    expect(section.get('[role="dialog"] h2').text()).toBe('Link a device')
  })

  it('draws no link dialog and no link button of its own', async () => {
    const section = await mountSection([THIS])
    expect(section.find('[role="dialog"]').exists()).toBe(false)
    expect(section.text()).not.toContain('Link a new device')
  })
})
