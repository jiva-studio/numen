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
import { createDeviceStore } from './devices'

const RESTORE_POLL_MS = 1000

const waitForIdentity = async (api: IdentityApi): Promise<IdentityInfo> => {
  for (;;) {
    await new Promise<void>((resolve) => setTimeout(resolve, RESTORE_POLL_MS))
    const found = await api.account.getIdentity().catch(() => undefined)
    if (found !== undefined) return found
  }
}

const attempt = async <T>(call: () => Promise<T>): Promise<Result<T, string>> => {
  try {
    return { ok: true, value: await call() }
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

  const generateWords = (): Promise<Result<readonly string[], string>> =>
    attempt(() => api.account.generateWords())

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

  const restore = async (proof: KitProof): Promise<Result<IdentityInfo, string>> => {
    const started = await settle(
      () =>
        'words' in proof
          ? api.account.startRestore(proof.words)
          : api.account.startRestoreFromKit(proof.sheets),
      'words' in proof ? 'words' : 'sheets',
    )
    if (!started.ok) return started
    return attempt(async () => {
      const found = await waitForIdentity(api)
      identity.value = found
      return found
    })
  }

  const remove = (): Promise<Result<void, string>> =>
    attempt(async () => {
      await api.account.deleteIdentity()
      identity.value = undefined
    })

  const saveKit = (proof: KitProof): Promise<Result<boolean, string>> =>
    settle(() => api.account.saveKit(proof), 'words' in proof ? 'words' : 'sheets')

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
  }
}

export type IdentityStore = ReturnType<typeof createIdentityStore>
