/**
 * What a license is, as the window holds it: the plugins that contribute to it, the checkout
 * providers through which a key can be purchased, and its activation state on this machine.
 */

/** The state of a license on this machine. */
type LicenseStatus = 'inactive' | 'active' | 'expired' | 'unverified'

/** One plugin that declares a license and what it unlocks when the license is active. */
interface LicenseContributorInfo {
  readonly pluginId: string
  readonly pluginName: string
  readonly description: string
}

/** One service through which a person can buy a key for a license. */
export interface LicenseProviderOptionInfo {
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
export interface LicenseFailure {
  readonly error: LicenseErrorCode
  readonly message: string
}
