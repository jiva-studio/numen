/**
 * The identity of this device as the core holds it. The recovery words are not kept here: they pass
 * through the calls that make and use them.
 */
import { shallowRef } from 'vue'
import { createRequestGuard } from '@numen/ui'
import { ReportedError, type Result } from '@numen/result'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { IdentityApi } from '../api/identity'
import type { IdentityFailure, IdentityInfo, KitProof } from '../lib/types'
import { IDENTITY_WORDS } from '../words'
import { createDeviceStore } from './devices'

const RESTORE_POLL_MS = 1000
const RESTORE_READ_FAILURES = 5

const RESTORE_LIMIT_MS = 60_000

const sleep = (ms: number, signal: AbortSignal | undefined): Promise<void> =>
  new Promise<void>((resolve, reject) => {
    if (signal?.aborted === true) {
      reject(signal.reason)
      return
    }
    const onAbort = (): void => {
      clearTimeout(timer)
      reject(signal?.reason)
    }
    const timer = setTimeout(() => {
      signal?.removeEventListener('abort', onAbort)
      resolve()
    }, ms)
    signal?.addEventListener('abort', onAbort, { once: true })
  })

const waitForIdentity = async (
  api: IdentityApi,
  signal: AbortSignal | undefined,
): Promise<IdentityInfo> => {
  let failures = 0
  for (let polls = 0; polls < RESTORE_LIMIT_MS / RESTORE_POLL_MS; polls += 1) {
    await sleep(RESTORE_POLL_MS, signal)
    try {
      const found = await api.account.getIdentity()
      failures = 0
      signal?.throwIfAborted()
      if (found !== undefined) return found
    } catch (caughtError) {
      if (signal?.aborted === true) throw caughtError
      failures += 1
      if (failures >= RESTORE_READ_FAILURES) throw caughtError
    }
  }
  throw new ReportedError(IDENTITY_WORDS.loadFailed)
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

/** Makes a call once more after the person answered the question that its refusal raised. */
export type RetryAfterAsking = <T>(
  call: () => Promise<Result<T, string>>,
) => Promise<Result<T, string>>

const retryNever: RetryAfterAsking = (call) => call()

export function createIdentityStore(
  api: IdentityApi,
  notify: MessageWriter,
  retryAfterAsking: RetryAfterAsking = retryNever,
) {
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

  const create = (words: readonly string[]): Promise<Result<IdentityInfo, string>> =>
    retryAfterAsking(() => createOnce(words))

  const createOnce = async (words: readonly string[]): Promise<Result<IdentityInfo, string>> => {
    try {
      const answer = await api.account.createIdentity(words)
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

  const restore = async (
    proof: KitProof,
    signal?: AbortSignal,
  ): Promise<Result<IdentityInfo, string>> => {
    const started = await retryAfterAsking(() =>
      settle(
        () =>
          'words' in proof
            ? api.account.startRestore(proof.words)
            : api.account.startRestoreFromKit(proof.sheets),
        'words' in proof ? 'words' : 'sheets',
      ),
    )
    if (!started.ok) return started
    return attempt(async () => {
      const found = await waitForIdentity(api, signal)
      guard.acquireRequest()
      identity.value = found
      return found
    })
  }

  const remove = (): Promise<Result<void, string>> =>
    attempt(async () => {
      await api.account.deleteIdentity()
      guard.cancel()
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
