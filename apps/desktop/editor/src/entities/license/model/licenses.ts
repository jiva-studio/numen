/**
 * The licenses declared by the installed plugins and their activation state on this machine, as the
 * core holds them. A license key goes to the core once and is not kept here.
 */
import { shallowRef } from 'vue'
import { createRequestGuard } from '@numen/ui'
import type { Result } from '@numen/result'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { LicenseApi } from '../api/licenses'
import type { LicenseFailure, LicenseInfo } from '../lib/types'
import { LICENSE_WORDS } from '../words'

export function createLicenseStore(api: LicenseApi, notify: MessageWriter) {
  const licenses = shallowRef<readonly LicenseInfo[]>([])
  const guard = createRequestGuard()

  /**
   * Reads the declared licenses and their state again. A failed read keeps what the store held;
   * the answer to a read started before another is dropped.
   */
  const load = async (): Promise<void> => {
    const mine = guard.acquireRequest()
    try {
      const readLicenses = await api.listLicenses()
      if (!mine.isCurrent) return
      licenses.value = readLicenses
      notify('')
    } catch (caughtError) {
      if (!mine.isCurrent) return
      notify(`${LICENSE_WORDS.loadFailed} ${formatError(caughtError)}`, 'error')
    }
  }

  const replaceOne = (updated: LicenseInfo): void => {
    const existing = licenses.value
    const index = existing.findIndex((one) => one.id === updated.id)
    if (index === -1) {
      licenses.value = [...existing, updated]
      return
    }
    licenses.value = existing.map((one, at) => (at === index ? updated : one))
  }

  const run = async (
    operation: () => Promise<Result<LicenseInfo, LicenseFailure>>,
  ): Promise<Result<LicenseInfo, string>> => {
    try {
      const answer = await operation()
      if (!answer.ok) {
        return { ok: false, error: LICENSE_WORDS.describeFailure(answer.error) }
      }
      replaceOne(answer.value)
      return { ok: true, value: answer.value }
    } catch (caughtError) {
      return { ok: false, error: formatError(caughtError) }
    }
  }

  /** Follows the stream of license updates until `signal` is aborted. */
  const watch = async (signal: AbortSignal): Promise<void> => {
    try {
      for await (const updated of api.watchLicenses(signal)) {
        if (signal.aborted) return
        licenses.value = updated
      }
    } catch (caughtError) {
      if (signal.aborted) return
      notify(`${LICENSE_WORDS.loadFailed} ${formatError(caughtError)}`, 'error')
    }
  }

  return {
    licenses,
    load,
    watch,
    activate: (licenseId: string, key: string) =>
      run(() => api.activateLicense(licenseId, key)),
    deactivate: (licenseId: string) => run(() => api.deactivateLicense(licenseId)),
  }
}

export type LicenseStore = ReturnType<typeof createLicenseStore>
