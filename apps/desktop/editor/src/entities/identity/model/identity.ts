/**
 * The identity of this device as the core holds it. The recovery words are not kept here: they pass
 * through the calls that make and use them.
 */
import { shallowRef } from 'vue'
import { createRequestGuard } from '@numen/ui'
import type { Result } from '@numen/result'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { IdentityApi } from '../api/identity'
import type {
  DeviceInfo,
  IdentityFailure,
  IdentityInfo,
  LinkInfo,
  LinkMode,
  LinkRole,
} from '../lib/types'
import { IDENTITY_WORDS } from '../words'

const RESTORE_POLL_MS = 1000

export function createIdentityStore(api: IdentityApi, notify: MessageWriter) {
  const identity = shallowRef<IdentityInfo | undefined>(undefined)
  const guard = createRequestGuard()

  /**
   * Reads the identity again. A failed read keeps what the store held; the answer to a read started
   * before another is dropped.
   */
  const load = async (): Promise<void> => {
    const mine = guard.acquireRequest()
    try {
      const found = await api.getIdentity()
      if (!mine.isCurrent) return
      identity.value = found
      notify('')
    } catch (caughtError) {
      if (!mine.isCurrent) return
      notify(`${IDENTITY_WORDS.loadFailed} ${formatError(caughtError)}`, 'error')
    }
  }

  const generateWords = async (): Promise<Result<readonly string[], string>> => {
    try {
      return { ok: true, value: await api.generateWords() }
    } catch (caughtError) {
      return { ok: false, error: formatError(caughtError) }
    }
  }

  const create = async (words: readonly string[]): Promise<Result<IdentityInfo, string>> => {
    try {
      const answer = await api.createIdentity(words)
      if (!answer.ok) {
        return { ok: false, error: IDENTITY_WORDS.describeFailure(answer.error) }
      }
      identity.value = answer.value
      return { ok: true, value: answer.value }
    } catch (caughtError) {
      return { ok: false, error: formatError(caughtError) }
    }
  }

  const waitForIdentity = async (): Promise<IdentityInfo> => {
    for (;;) {
      await new Promise<void>((resolve) => setTimeout(resolve, RESTORE_POLL_MS))
      const found = await api.getIdentity().catch(() => undefined)
      if (found !== undefined) return found
    }
  }

  const restore = async (words: readonly string[]): Promise<Result<IdentityInfo, string>> => {
    try {
      const started = await api.startRestore(words)
      if (!started.ok) {
        return { ok: false, error: IDENTITY_WORDS.describeFailure(started.error) }
      }
      const found = await waitForIdentity()
      identity.value = found
      return { ok: true, value: found }
    } catch (caughtError) {
      return { ok: false, error: formatError(caughtError) }
    }
  }

  const remove = async (): Promise<Result<void, string>> => {
    try {
      await api.deleteIdentity()
      identity.value = undefined
      return { ok: true, value: undefined }
    } catch (caughtError) {
      return { ok: false, error: formatError(caughtError) }
    }
  }

  const devices = shallowRef<readonly DeviceInfo[]>([])

  const loadDevices = async (): Promise<void> => {
    try {
      const answer = await api.listDevices()
      if (answer.ok) devices.value = answer.value
    } catch (caughtError) {
      notify(`${IDENTITY_WORDS.loadFailed} ${formatError(caughtError)}`, 'error')
    }
  }

  /** Runs a call that answers with a refusal or a value and returns the refusal as text. */
  const ask = async <T>(call: () => Promise<Result<T, IdentityFailure>>): Promise<Result<T, string>> => {
    try {
      const answer = await call()
      return answer.ok ? answer : { ok: false, error: IDENTITY_WORDS.describeFailure(answer.error) }
    } catch (caughtError) {
      return { ok: false, error: formatError(caughtError) }
    }
  }

  const link = {
    start: (role: LinkRole, mode: LinkMode, text: string): Promise<Result<LinkInfo, string>> =>
      ask(() => api.startLink(role, mode, text)),
    get: (): Promise<Result<LinkInfo, string>> => ask(() => api.getLink()),
    decide: (approve: boolean) => ask(() => api.decideLink(approve)),
    enterCode: (code: string) => ask(() => api.enterLinkCode(code)),
    confirmJoin: (approve: boolean) => ask(() => api.confirmJoin(approve)),
    cancel: () => ask(() => api.cancelLink()),
  }

  const revoke = async (fingerprint: string): Promise<Result<undefined, string>> => {
    const answer = await ask(() => api.revokeDevice(fingerprint))
    await loadDevices()
    return answer
  }

  return { identity, devices, load, loadDevices, generateWords, create, restore, delete: remove, link, revoke }
}

export type IdentityStore = ReturnType<typeof createIdentityStore>
