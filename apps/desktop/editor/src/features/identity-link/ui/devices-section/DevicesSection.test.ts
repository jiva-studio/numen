/** The list of devices, the revoke after a question, and the one way to join an account, against the real store. */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import type { IdentityClient } from '@numen/core-client'
import { createIdentityStore, type DeviceInfo } from '@/entities/identity'
import { click, dispose, enterText, findButton, findButtons, findText } from '@/testing/dom'
import {
  createDeviceInfo,
  createIdentityClient,
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
  hasIdentity: boolean,
  devices: readonly DeviceInfo[],
  overrides: Partial<IdentityClient> = {},
) => {
  const client = createIdentityClient({
    listDevices: () => Promise.resolve({ ok: true, value: devices }),
    ...overrides,
  })
  const store = createIdentityStore(wrapIdentityClient(client), vi.fn())
  wrapper = mount(DevicesSection, {
    props: { calls: store.devices, hasIdentity },
    attachTo: document.body,
  })
  await flushPromises()
  return wrapper
}

describe('the devices section', () => {
  it('lists the devices of the account and marks this one', async () => {
    const section = await mountSection(true, [THIS, OTHER])
    const rows = section.findAll('li')
    expect(rows).toHaveLength(2)
    expect(rows[0]?.text()).toContain('This device')
    expect(rows[1]?.text()).toContain(OTHER.fingerprint)
  })

  it('offers no revoke on this device and one on another', async () => {
    const section = await mountSection(true, [THIS, OTHER])
    const rows = section.findAll('li')
    expect(rows[0]?.find('button').exists()).toBe(false)
    expect(rows[1]?.find('button').text()).toBe('Revoke')
  })

  it('offers no revoke on a revoked device', async () => {
    const section = await mountSection(true, [THIS, { ...OTHER, isRevoked: true }])
    const rows = section.findAll('li')
    expect(rows[1]?.text()).toContain('Revoked')
    expect(rows[1]?.find('button').exists()).toBe(false)
  })

  it('revokes a device only after the person confirms the question', async () => {
    const revokeDevice = vi.fn(() => Promise.resolve({ ok: true as const, value: undefined }))
    await mountSection(true, [THIS, OTHER], { revokeDevice })
    await click(findButtons(/^Revoke$/)[0] as HTMLElement)
    expect(
      findText(
        'Phone will lose access. New notes will not be readable on it. Its files stay on it.',
      ),
    ).not.toBeNull()
    expect(revokeDevice).not.toHaveBeenCalled()
    await click(findButtons(/^Revoke$/)[1] as HTMLElement)
    expect(revokeDevice).toHaveBeenCalledWith(OTHER.fingerprint, { reason: 'lost' })
  })

  it('keeps the device when the person cancels the question', async () => {
    const revokeDevice = vi.fn(() => Promise.resolve({ ok: true as const, value: undefined }))
    await mountSection(true, [THIS, OTHER], { revokeDevice })
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
    const section = await mountSection(true, [THIS, OTHER], { revokeDevice })
    await click(findButtons(/^Revoke$/)[0] as HTMLElement)
    await click(findButtons(/^Revoke$/)[1] as HTMLElement)
    expect(section.get('[role="alert"]').text()).toBe(
      'A device was added while you revoked. Check the list.',
    )
  })

  it('offers to link a new device where the device has the account, and not to join', async () => {
    const section = await mountSection(true, [THIS])
    expect(section.text()).toContain('Link a new device')
    expect(section.text()).not.toContain('Join with a QR code')
  })

  it('offers to join an account where the device has none, and no list', async () => {
    const section = await mountSection(false, [])
    expect(section.text()).toContain('Join with a QR code')
    expect(section.text()).not.toContain('Link a new device')
    expect(section.find('ul').exists()).toBe(false)
  })
})

const findRadio = (value: string): HTMLInputElement => {
  const found = document.body.querySelector<HTMLInputElement>(`input[type="radio"][value="${value}"]`)
  if (found === null) throw new Error(`no radio ${value}`)
  return found
}

const findDate = (): HTMLInputElement => {
  const found = document.body.querySelector<HTMLInputElement>('input[type="date"]')
  if (found === null) throw new Error('no date field')
  return found
}

describe('the reason of a revoke', () => {
  it('asks whether the device is lost or stolen and revokes it as lost by default', async () => {
    const revokeDevice = vi.fn(() => Promise.resolve({ ok: true as const, value: undefined }))
    await mountSection(true, [THIS, OTHER], { revokeDevice })
    await click(findButtons(/^Revoke$/)[0] as HTMLElement)

    expect(findRadio('lost').checked).toBe(true)
    expect(document.body.querySelector('input[type="date"]')).toBeNull()
    await click(findButtons(/^Revoke$/)[1] as HTMLElement)
    expect(revokeDevice).toHaveBeenCalledWith(OTHER.fingerprint, { reason: 'lost' })
  })

  it('wants the day a stolen device was lost and keeps the question open without it', async () => {
    const revokeDevice = vi.fn(() => Promise.resolve({ ok: true as const, value: undefined }))
    await mountSection(true, [THIS, OTHER], { revokeDevice })
    await click(findButtons(/^Revoke$/)[0] as HTMLElement)
    await click(findRadio('stolen'))

    await click(findButtons(/^Revoke$/)[1] as HTMLElement)

    expect(revokeDevice).not.toHaveBeenCalled()
    expect(findText('Choose the day it was stolen, or the first day you are not sure about.')).not.toBeNull()
    expect(findRadio('stolen').checked).toBe(true)
  })

  it('sends the day a stolen device was lost', async () => {
    const revokeDevice = vi.fn(() => Promise.resolve({ ok: true as const, value: undefined }))
    await mountSection(true, [THIS, OTHER], { revokeDevice })
    await click(findButtons(/^Revoke$/)[0] as HTMLElement)
    await click(findRadio('stolen'))
    await enterText(findDate(), '2026-11-20')

    await click(findButtons(/^Revoke$/)[1] as HTMLElement)

    expect(revokeDevice).toHaveBeenCalledWith(OTHER.fingerprint, {
      reason: 'stolen',
      since: new Date('2026-11-20T00:00:00').toISOString(),
    })
  })

  it('says who revoked a device, when and why', async () => {
    const section = await mountSection(true, [
      THIS,
      {
        ...OTHER,
        isRevoked: true,
        revocation: {
          revokedAt: '2026-11-21T10:00:00Z',
          revokedByName: 'Laptop',
          isByRecovery: false,
          reason: 'stolen',
          since: '2026-11-20T00:00:00Z',
        },
      },
    ])
    expect(section.findAll('li')[1]?.text()).toContain(
      'Revoked by Laptop on November 21, 2026 as stolen since November 20, 2026',
    )
  })

  it('tells every device when a device revoked another lately', async () => {
    const section = await mountSection(true, [
      THIS,
      {
        ...OTHER,
        isRevoked: true,
        revocation: {
          revokedAt: new Date().toISOString(),
          revokedByName: 'Laptop',
          isByRecovery: false,
          reason: 'lost',
          since: '',
        },
      },
    ])
    expect(section.get('[role="status"]').text()).toContain('Laptop revoked Phone on')
    expect(section.get('[role="status"]').text()).toContain('revoke it as stolen')
  })

  it('does not tell about a revoke made with the recovery words or long ago', async () => {
    const section = await mountSection(true, [
      THIS,
      {
        ...OTHER,
        isRevoked: true,
        revocation: {
          revokedAt: new Date().toISOString(),
          revokedByName: '',
          isByRecovery: true,
          reason: 'lost',
          since: '',
        },
      },
      {
        ...OTHER,
        fingerprint: 'OLD',
        isRevoked: true,
        revocation: {
          revokedAt: '2020-01-01T00:00:00Z',
          revokedByName: 'Laptop',
          isByRecovery: false,
          reason: 'lost',
          since: '',
        },
      },
    ])
    expect(section.find('[role="status"]').exists()).toBe(false)
  })
})
