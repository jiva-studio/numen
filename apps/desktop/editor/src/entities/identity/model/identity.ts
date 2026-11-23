/**
 * The identity of this device as the core holds it. The recovery words are not kept here: they pass
 * through the calls that make and use them.
 */
import { shallowRef } from 'vue'
import { RestoreRefusedError } from '@numen/core-client'
import { createRequestGuard } from '@numen/ui'
import type { Result } from '@numen/result'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { IdentityApi } from '../api/identity'
import type { IdentityFailure, IdentityInfo, KitProof } from '../lib/types'
import { IDENTITY_WORDS } from '../words'
import { createDeviceStore } from './devices'

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
      const found = await api.account.getIdentity()
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
      return { ok: true, value: await api.account.generateWords() }
    } catch (caughtError) {
      return { ok: false, error: formatError(caughtError) }
    }
  }

  const create = async (words: readonly string[]): Promise<Result<IdentityInfo, string>> => {
    try {
      const answer = await api.account.createIdentity(words)
      if (!answer.ok) {
        return { ok: false, error: IDENTITY_WORDS.describeFailure(answer.error) }
      }
      identity.value = answer.value
      return { ok: true, value: answer.value }
    } catch (caughtError) {
      return { ok: false, error: formatError(caughtError) }
    }
  }

  /** Reads the identity until the restore makes it; the restore that the core refused ends the wait with its refusal. */
  const waitForIdentity = async (): Promise<IdentityInfo> => {
    for (;;) {
      await new Promise<void>((resolve) => setTimeout(resolve, RESTORE_POLL_MS))
      const found = await api.account.getIdentity().catch((caughtError: unknown) => {
        if (caughtError instanceof RestoreRefusedError) throw caughtError
        return undefined
      })
      if (found !== undefined) return found
    }
  }

  const describeRestoreFailure = (caughtError: unknown): string =>
    caughtError instanceof RestoreRefusedError
      ? IDENTITY_WORDS.describeFailure(caughtError.failure)
      : formatError(caughtError)

  const restore = async (
    words: readonly string[],
    chainMark: string,
  ): Promise<Result<IdentityInfo, string>> => {
    try {
      const started = await api.account.startRestore(words, chainMark)
      if (!started.ok) {
        return { ok: false, error: IDENTITY_WORDS.describeFailure(started.error) }
      }
      const found = await waitForIdentity()
      identity.value = found
      return { ok: true, value: found }
    } catch (caughtError) {
      return { ok: false, error: describeRestoreFailure(caughtError) }
    }
  }

  const remove = async (): Promise<Result<void, string>> => {
    try {
      await api.account.deleteIdentity()
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
    settle(() => api.account.saveKit(proof), 'words' in proof ? 'words' : 'sheets')

  const restoreFromKit = async (
    sheets: readonly (readonly string[])[],
    chainMark: string,
  ): Promise<Result<IdentityInfo, string>> => {
    const started = await settle(
      () => api.account.startRestoreFromKit(sheets, chainMark),
      'sheets',
    )
    if (!started.ok) return started
    try {
      const found = await waitForIdentity()
      identity.value = found
      return { ok: true, value: found }
    } catch (caughtError) {
      return { ok: false, error: describeRestoreFailure(caughtError) }
    }
  }

  const devices = createDeviceStore(api, notify)

  return {
    identity,
    devices,
    load,
    generateWords,
    create,
    restore,
    delete: remove,
    saveKit,
    restoreFromKit,
  }
}

export type IdentityStore = ReturnType<typeof createIdentityStore>
