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
import type { IdentityFailure, IdentityInfo, KitProof } from '../lib/types'
import { IDENTITY_WORDS } from '../words'

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
      guard.cancel()
      identity.value = answer.value
      return { ok: true, value: answer.value }
    } catch (caughtError) {
      return { ok: false, error: formatError(caughtError) }
    }
  }

  const remove = async (): Promise<Result<void, string>> => {
    try {
      await api.deleteIdentity()
      guard.cancel()
      identity.value = undefined
      return { ok: true, value: undefined }
    } catch (caughtError) {
      return { ok: false, error: formatError(caughtError) }
    }
  }

  const settle = async <T>(
    call: () => Promise<Result<T, IdentityFailure>>,
    proves: 'words' | 'sheets',
  ): Promise<Result<T, string>> => {
    try {
      const answer = await call()
      return answer.ok
        ? answer
        : { ok: false, error: IDENTITY_WORDS.describeFailure(answer.error, proves) }
    } catch (caughtError) {
      return { ok: false, error: formatError(caughtError) }
    }
  }

  const saveKit = (proof: KitProof): Promise<Result<boolean, string>> =>
    settle(() => api.saveKit(proof), 'words' in proof ? 'words' : 'sheets')

  const restoreFromKit = async (
    sheets: readonly (readonly string[])[],
  ): Promise<Result<IdentityInfo, string>> => {
    const answer = await settle(() => api.restoreFromKit(sheets), 'sheets')
    if (answer.ok) {
      guard.acquireRequest()
      identity.value = answer.value
    }
    return answer
  }

  return {
    identity,
    load,
    generateWords,
    create,
    delete: remove,
    saveKit,
    restoreFromKit,
  }
}

export type IdentityStore = ReturnType<typeof createIdentityStore>
