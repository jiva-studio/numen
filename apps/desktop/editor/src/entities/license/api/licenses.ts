/**
 * The calls of the licenses: the declared licenses and their state on this machine, machine
 * activation and deactivation, and the stream of updates when licenses change.
 */
import type { LicenseClient } from '@numen/core-client'
import type { Result } from '@numen/result'
import type { LicenseFailure, LicenseInfo } from '../lib/types'

/** The calls of the licenses over `licenses`. */
export const createLicenseApi = (licenses: LicenseClient) => ({
  /** Every declared license, in manifest order. */
  listLicenses: async (): Promise<readonly LicenseInfo[]> =>
    (await licenses.list()).map((one) => ({ ...one })),

  /** Activates `licenseId` on this machine with `key`. */
  activateLicense: async (
    licenseId: string,
    key: string,
  ): Promise<Result<LicenseInfo, LicenseFailure>> => licenses.activate(licenseId, key),

  /** Deactivates `licenseId` on this machine. */
  deactivateLicense: async (licenseId: string): Promise<Result<LicenseInfo, LicenseFailure>> =>
    licenses.deactivate(licenseId),

  /** Streams the full license list whenever a license changes. */
  watchLicenses: (signal: AbortSignal): AsyncGenerator<readonly LicenseInfo[]> =>
    licenses.watch(signal),
})

export type LicenseApi = ReturnType<typeof createLicenseApi>
