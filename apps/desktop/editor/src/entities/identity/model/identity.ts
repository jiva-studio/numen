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
import type { IdentityInfo } from '../lib/types'
import { IDENTITY_WORDS } from '../words'

const RESTORE_POLL_MS = 1000
const RESTORE_READ_FAILURES = 5

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
    let failures = 0
    for (;;) {
      await new Promise<void>((resolve) => setTimeout(resolve, RESTORE_POLL_MS))
      try {
        const found = await api.getIdentity()
        failures = 0
        if (found !== undefined) return found
      } catch (caughtError) {
        failures += 1
        if (failures >= RESTORE_READ_FAILURES) throw caughtError
      }
    }
  }

  const restore = async (words: readonly string[]): Promise<Result<IdentityInfo, string>> => {
    try {
      const started = await api.startRestore(words)
      if (!started.ok) {
        return { ok: false, error: IDENTITY_WORDS.describeFailure(started.error) }
      }
      const found = await waitForIdentity()
      guard.acquireRequest()
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

  return { identity, load, generateWords, create, restore, delete: remove }
}

export type IdentityStore = ReturnType<typeof createIdentityStore>
