/**
 * The licenses that plugins declare: their contributors, checkout providers and activation state on
 * this machine. No raw license key is returned to the window.
 */
import type { Client } from '@connectrpc/connect'
import {
  ErrorCode as ProtoErrorCode,
  LicenseStatus as ProtoLicenseStatus,
  type LicenseInfo as ProtoLicenseInfo,
  type LicenseService,
} from '@numen/protocol'
import { asFailure, asValue, type Failure, type Result } from '@numen/result'

/** The state of a license on this machine. */
type LicenseStatus = 'inactive' | 'active' | 'expired' | 'unverified'

/** One plugin that declares a license and what it unlocks when the license is active. */
interface LicenseContributorInfo {
  readonly pluginId: string
  readonly pluginName: string
  readonly description: string
}

/** One service through which a person can buy a key for a license. */
interface LicenseProviderOptionInfo {
  readonly provider: string
  readonly providerName: string
  readonly productId: string
  readonly checkoutUrl: string
  readonly paymentMethods: readonly string[]
}

/** One license declared by one or more plugins, together with its activation state on this machine. */
export interface LicenseInfo {
  readonly id: string
  readonly title: string
  readonly contributors: readonly LicenseContributorInfo[]
  readonly providers: readonly LicenseProviderOptionInfo[]
  readonly status: LicenseStatus
  readonly isActive: boolean
  readonly hasKey: boolean
  readonly keyHint: string
  readonly activeProvider: string
  readonly instanceName: string
  readonly expiresUnixMs: number
  readonly verifiedUnixMs: number
}

/** Why an activation or deactivation call was rejected. */
export type LicenseErrorCode =
  | 'notFound'
  | 'invalidKey'
  | 'activationLimitReached'
  | 'expired'
  | 'disabled'
  | 'productMismatch'
  | 'providerUnavailable'
  | 'keyStoreFailed'

/** A served refusal with the reason the core gives. */
export type LicenseFailure = Failure<LicenseErrorCode>

/** The calls of LicenseService the licenses are managed through. */
export type LicenseRpc = Pick<
  Client<typeof LicenseService>,
  'listLicenses' | 'activateLicense' | 'deactivateLicense' | 'watchLicenses'
>

/** The licenses of the application. */
export interface LicenseClient {
  list(): Promise<readonly LicenseInfo[]>
  activate(licenseId: string, key: string): Promise<Result<LicenseInfo, LicenseFailure>>
  deactivate(licenseId: string): Promise<Result<LicenseInfo, LicenseFailure>>
  watch(signal: AbortSignal): AsyncGenerator<readonly LicenseInfo[]>
}

const readStatus = (status: ProtoLicenseStatus): LicenseStatus => {
  switch (status) {
    case ProtoLicenseStatus.ACTIVE:
      return 'active'
    case ProtoLicenseStatus.EXPIRED:
      return 'expired'
    case ProtoLicenseStatus.UNVERIFIED:
      return 'unverified'
    case ProtoLicenseStatus.INACTIVE:
    case ProtoLicenseStatus.UNSPECIFIED:
      return 'inactive'
  }
}

const readLicense = (info: ProtoLicenseInfo): LicenseInfo => ({
  id: info.id,
  title: info.title,
  contributors: info.contributors.map((one) => ({
    pluginId: one.pluginId,
    pluginName: one.pluginName,
    description: one.description,
  })),
  providers: info.providers.map((one) => ({
    provider: one.provider,
    providerName: one.providerName,
    productId: one.productId,
    checkoutUrl: one.checkoutUrl,
    paymentMethods: [...one.paymentMethods],
  })),
  status: readStatus(info.status),
  isActive: info.isActive,
  hasKey: info.hasKey,
  keyHint: info.keyHint,
  activeProvider: info.activeProvider,
  instanceName: info.instanceName,
  expiresUnixMs: Number(info.expiresUnixMs),
  verifiedUnixMs: Number(info.verifiedUnixMs),
})

const readErrorCode = (code: ProtoErrorCode): LicenseErrorCode | undefined => {
  switch (code) {
    case ProtoErrorCode.UNSPECIFIED:
      return undefined
    case ProtoErrorCode.NOT_FOUND:
      return 'notFound'
    case ProtoErrorCode.INVALID:
      return 'invalidKey'
    case ProtoErrorCode.ACTIVATION_LIMIT_REACHED:
      return 'activationLimitReached'
    case ProtoErrorCode.EXPIRED:
      return 'expired'
    case ProtoErrorCode.DISABLED:
      return 'disabled'
    case ProtoErrorCode.PRODUCT_MISMATCH:
      return 'productMismatch'
    case ProtoErrorCode.UNAVAILABLE:
      return 'providerUnavailable'
    case ProtoErrorCode.KEY_STORE_FAILED:
      return 'keyStoreFailed'
    default:
      return undefined
  }
}

const readAnswer = (answer: {
  error: ProtoErrorCode
  detail: string
  license?: ProtoLicenseInfo | undefined
}): Result<LicenseInfo, LicenseFailure> => {
  const error = readErrorCode(answer.error)
  if (error !== undefined) {
    return asFailure({ error, message: answer.detail })
  }
  if (answer.license === undefined) {
    return asFailure({ error: 'notFound', message: 'The core returned no license.' })
  }
  return asValue(readLicense(answer.license))
}

/** The licenses managed through a LicenseService client. */
export const createLicenseClient = (rpc: LicenseRpc): LicenseClient => ({
  list: async () => (await rpc.listLicenses({})).licenses.map(readLicense),
  activate: async (licenseId, key) => readAnswer(await rpc.activateLicense({ licenseId, key })),
  deactivate: async (licenseId) => readAnswer(await rpc.deactivateLicense({ licenseId })),
  watch: async function* (signal: AbortSignal): AsyncGenerator<readonly LicenseInfo[]> {
    for await (const update of rpc.watchLicenses({}, { signal })) {
      yield update.licenses.map(readLicense)
    }
  },
})
