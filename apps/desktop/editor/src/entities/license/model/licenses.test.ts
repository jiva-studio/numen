import { describe, expect, it, vi } from 'vitest'
import type { LicenseClient } from '@numen/core-client'
import type { Result } from '@numen/result'
import type { LicenseFailure, LicenseInfo } from '../lib/types'
import { createLicenseApi } from '../api/licenses'
import { createLicenseStore } from './licenses'

const asValue = <T>(value: T): Result<T, LicenseFailure> => ({ ok: true, value })
const asFailure = (error: LicenseFailure): Result<never, LicenseFailure> => ({
  ok: false,
  error,
})

const createSampleLicense = (overrides: Partial<LicenseInfo> = {}): LicenseInfo => ({
  id: 'numen.sync',
  title: 'Numen Sync',
  contributors: [
    {
      pluginId: 'md.numen.views.agent-chat',
      pluginName: 'Agent Chat',
      description: 'Multi-agent conversations',
    },
  ],
  providers: [
    {
      provider: 'creem',
      providerName: 'Creem',
      productId: 'prod_1',
      checkoutUrl: 'https://creem.io/checkout/numen-sync',
      paymentMethods: ['Cards'],
    },
  ],
  status: 'inactive',
  isActive: false,
  hasKey: false,
  keyHint: '',
  activeProvider: '',
  instanceName: '',
  expiresUnixMs: 0,
  verifiedUnixMs: 0,
  ...overrides,
})

const createClient = (overrides: Partial<LicenseClient> = {}): LicenseClient => ({
  list: () => Promise.resolve([createSampleLicense()]),
  activate: () =>
    Promise.resolve(
      asValue(
        createSampleLicense({
          status: 'active',
          isActive: true,
          hasKey: true,
          keyHint: '••••-814A',
          activeProvider: 'creem',
          instanceName: 'workstation',
        }),
      ),
    ),
  deactivate: () => Promise.resolve(asValue(createSampleLicense())),
  watch: async function* () {
    yield [createSampleLicense({ status: 'expired', hasKey: true, keyHint: '••••-814A' })]
  },
  ...overrides,
})

const createStore = (client: LicenseClient) => {
  const notify = vi.fn()
  const api = createLicenseApi(client)
  return { api, notify, store: createLicenseStore(api, notify) }
}

describe('createLicenseStore', () => {
  it('loads declared licenses and clears any notice', async () => {
    const { notify, store } = createStore(createClient())
    expect(store.licenses.value).toStrictEqual([])

    await store.load()

    expect(store.licenses.value.map((one) => one.id)).toStrictEqual(['numen.sync'])
    expect(notify).toHaveBeenCalledWith('')
  })

  it('activates and deactivates a license in place', async () => {
    const { store } = createStore(createClient())
    await store.load()

    const activated = await store.activate('numen.sync', 'KEY-814A')
    expect(activated.ok).toBe(true)
    expect(store.licenses.value[0]?.status).toBe('active')
    expect(store.licenses.value[0]?.keyHint).toBe('••••-814A')

    const deactivated = await store.deactivate('numen.sync')
    expect(deactivated.ok).toBe(true)
    expect(store.licenses.value[0]?.status).toBe('inactive')
  })

  it('translates activation refusals to user messages', async () => {
    const { store } = createStore(
      createClient({
        activate: () => Promise.resolve(asFailure({ error: 'invalidKey', message: 'bad key' })),
      }),
    )

    const answer = await store.activate('numen.sync', 'BAD')
    expect(answer).toStrictEqual({
      ok: false,
      error: 'This license key was not recognized.',
    })
  })

  it('updates licenses from the watch stream', async () => {
    const { store } = createStore(createClient())
    await store.watch(new AbortController().signal)
    expect(store.licenses.value[0]?.status).toBe('expired')
  })
})
