import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import {
  ActivateLicenseResponseSchema,
  DeactivateLicenseResponseSchema,
  LicenseErrorCode,
  LicenseStatus,
  ListLicensesResponseSchema,
  WatchLicensesResponseSchema,
} from '@numen/protocol'
import { createLicenseClient, type LicenseRpc } from './licenses'

const createRpc = (overrides: Partial<LicenseRpc> = {}): LicenseRpc => ({
  listLicenses: () =>
    Promise.resolve(
      create(ListLicensesResponseSchema, {
        licenses: [
          {
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
                productId: 'prod_creem_1',
                checkoutUrl: 'https://creem.io/checkout/numen-sync',
                paymentMethods: ['Cards', 'Apple Pay'],
              },
            ],
            status: LicenseStatus.ACTIVE,
            isActive: true,
            hasKey: true,
            keyHint: '••••-814A',
            activeProvider: 'creem',
            instanceName: 'workstation',
            expiresUnixMs: 1798675200000n,
            verifiedUnixMs: 1795003200000n,
          },
        ],
      }),
    ),
  activateLicense: () =>
    Promise.resolve(
      create(ActivateLicenseResponseSchema, {
        license: {
          id: 'numen.sync',
          title: 'Numen Sync',
          status: LicenseStatus.ACTIVE,
          isActive: true,
          hasKey: true,
          keyHint: '••••-814A',
          activeProvider: 'creem',
          instanceName: 'workstation',
        },
      }),
    ),
  deactivateLicense: () =>
    Promise.resolve(
      create(DeactivateLicenseResponseSchema, {
        license: {
          id: 'numen.sync',
          title: 'Numen Sync',
          status: LicenseStatus.INACTIVE,
          isActive: false,
          hasKey: false,
        },
      }),
    ),
  watchLicenses: async function* () {
    yield create(WatchLicensesResponseSchema, {
      licenses: [
        {
          id: 'numen.sync',
          title: 'Numen Sync',
          status: LicenseStatus.EXPIRED,
          isActive: false,
          hasKey: true,
          keyHint: '••••-814A',
        },
      ],
    })
  },
  ...overrides,
})

describe('createLicenseClient', () => {
  it('lists the declared licenses with their contributors, providers and state', async () => {
    const licenses = await createLicenseClient(createRpc()).list()

    expect(licenses).toStrictEqual([
      {
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
            productId: 'prod_creem_1',
            checkoutUrl: 'https://creem.io/checkout/numen-sync',
            paymentMethods: ['Cards', 'Apple Pay'],
          },
        ],
        status: 'active',
        isActive: true,
        hasKey: true,
        keyHint: '••••-814A',
        activeProvider: 'creem',
        instanceName: 'workstation',
        expiresUnixMs: 1798675200000,
        verifiedUnixMs: 1795003200000,
      },
    ])
  })

  it('activates and deactivates a license', async () => {
    const client = createLicenseClient(createRpc())

    const activated = await client.activate('numen.sync', 'KEY-814A')
    expect(activated.ok).toBe(true)
    expect(activated.ok && activated.value.status).toBe('active')

    const deactivated = await client.deactivate('numen.sync')
    expect(deactivated.ok).toBe(true)
    expect(deactivated.ok && deactivated.value.status).toBe('inactive')
  })

  it('reads each served error code as a failure', async () => {
    const codes: Array<[LicenseErrorCode, string]> = [
      [LicenseErrorCode.NOT_FOUND, 'notFound'],
      [LicenseErrorCode.INVALID_KEY, 'invalidKey'],
      [LicenseErrorCode.ACTIVATION_LIMIT_REACHED, 'activationLimitReached'],
      [LicenseErrorCode.EXPIRED, 'expired'],
      [LicenseErrorCode.DISABLED, 'disabled'],
      [LicenseErrorCode.PRODUCT_MISMATCH, 'productMismatch'],
      [LicenseErrorCode.PROVIDER_UNAVAILABLE, 'providerUnavailable'],
      [LicenseErrorCode.KEY_STORE_FAILED, 'keyStoreFailed'],
    ]
    for (const [code, expected] of codes) {
      const client = createLicenseClient(
        createRpc({
          activateLicense: () =>
            Promise.resolve(
              create(ActivateLicenseResponseSchema, { error: code, detail: 'refused' }),
            ),
        }),
      )
      const answer = await client.activate('numen.sync', 'BAD')
      expect(answer).toStrictEqual({
        ok: false,
        error: { error: expected, message: 'refused' },
      })
    }
  })

  it('streams license updates from watch', async () => {
    const client = createLicenseClient(createRpc())
    const updates = []
    for await (const batch of client.watch(new AbortController().signal)) {
      updates.push(batch)
    }
    expect(updates).toHaveLength(1)
    expect(updates[0]?.[0]?.status).toBe('expired')
  })
})
