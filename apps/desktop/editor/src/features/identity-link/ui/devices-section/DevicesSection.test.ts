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

const openMenu = async (index: number): Promise<void> => {
  const row = document.querySelectorAll('li')[index]
  row?.dispatchEvent(
    new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 40, clientY: 40 }),
  )
  await flushPromises()
}

describe('the devices section', () => {
  it('lists the devices of the identity and marks this one', async () => {
    const section = await mountSection([THIS, OTHER])
    const rows = section.findAll('li')
    expect(rows).toHaveLength(2)
    expect(rows[0]?.text()).toContain('This device')
    expect(rows[1]?.text()).toContain(OTHER.fingerprint)
  })

  it('has no button on any row and opens a menu with Remove on a right-click of another device', async () => {
    const section = await mountSection([THIS, OTHER])
    expect(section.find('li button').exists()).toBe(false)
    expect(document.querySelector('[role="menuitem"]')).toBeNull()
    await openMenu(1)
    expect(findMenuItem(/^Remove$/)).not.toBeNull()
  })

  it('opens no menu on this device and makes only another device reachable by keyboard', async () => {
    const section = await mountSection([THIS, OTHER])
    await openMenu(0)
    expect(document.querySelector('[role="menuitem"]')).toBeNull()
    const rows = section.findAll('li')
    expect(rows[0]?.attributes('tabindex')).toBeUndefined()
    expect(rows[1]?.attributes('tabindex')).toBe('0')
  })


  it('revokes a device only after the person confirms the question', async () => {
    const revokeDevice = vi.fn(() => Promise.resolve({ ok: true as const, value: undefined }))
    await mountSection([THIS, OTHER], { revokeDevice })
    await openMenu(1)
    await click(findMenuItem(/^Remove$/))
    expect(
      findText(
        `Phone (${OTHER.fingerprint}) will lose access. The files on the device stay.`,
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
    await openMenu(1)
    await click(findMenuItem(/^Remove$/))
    await click(findButton(/^Remove$/))
    await flushPromises()
    expect(section.findAll('li')).toHaveLength(1)
  })

  it('keeps the device when the person cancels the question', async () => {
    const revokeDevice = vi.fn(() => Promise.resolve({ ok: true as const, value: undefined }))
    await mountSection([THIS, OTHER], { revokeDevice })
    await openMenu(1)
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
    await openMenu(1)
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

describe('devices with the same name', () => {
  it('removes the one that was chosen and keeps the other with its own fingerprint', async () => {
    const twin = { ...OTHER, fingerprint: 'aaaa bbbb cccc dddd eeee ffff 0000 1111' }
    let listed: readonly DeviceInfo[] = [THIS, OTHER, twin]
    const revokeDevice = vi.fn((fingerprint: string) => {
      listed = listed.filter((device) => device.fingerprint !== fingerprint)
      return Promise.resolve({ ok: true as const, value: undefined })
    })
    const section = await mountSection([THIS, OTHER, twin], {
      revokeDevice,
      listDevices: () => Promise.resolve({ ok: true, value: listed }),
    })
    await openMenu(2)
    await click(findMenuItem(/^Remove$/))
    expect(findText(twin.fingerprint)).not.toBeNull()
    await click(findButton(/^Remove$/))
    await flushPromises()
    expect(revokeDevice).toHaveBeenCalledWith(twin.fingerprint)
    const rows = section.findAll('li')
    expect(rows).toHaveLength(2)
    expect(rows[1]?.text()).toContain(OTHER.fingerprint)
  })
})
