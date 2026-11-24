/**
 * How the keys of this computer are kept when it has no system keychain, and the question the person
 * is asked about it. A call that was refused because the file is not ready waits for the answer and is
 * made once more.
 */
import { ref, shallowRef } from 'vue'
import type { Result } from '@numen/result'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'
import type { KeyFileApi } from '../api/keyFile'
import type { KeyFileRequest, KeyFileState } from '../lib/types'
import { KEY_FILE_WORDS } from '../words'

const readRequest = (state: KeyFileState): KeyFileRequest | undefined => {
  if (state.isChoiceNeeded) return 'choose'
  return state.isLocked ? 'unlock' : undefined
}

export function createKeyFileStore(api: KeyFileApi, notify: MessageWriter) {
  const state = shallowRef<KeyFileState | undefined>(undefined)
  const request = shallowRef<KeyFileRequest | undefined>(undefined)
  const problem = ref('')
  const isBusy = ref(false)
  let answer: ((isDone: boolean) => void) | undefined

  const load = async (): Promise<KeyFileState | undefined> => {
    try {
      state.value = await api.getState()
      return state.value
    } catch (caughtError) {
      notify(`${KEY_FILE_WORDS.loadFailed} ${formatError(caughtError)}`, 'error')
      return undefined
    }
  }

  const close = (isDone: boolean): void => {
    request.value = undefined
    const waiting = answer
    answer = undefined
    waiting?.(isDone)
  }

  const show = (kind: KeyFileRequest, resolve: (isDone: boolean) => void): void => {
    answer?.(false)
    problem.value = ''
    request.value = kind
    answer = resolve
  }

  const ask = (kind: KeyFileRequest): Promise<boolean> =>
    new Promise<boolean>((resolve) => show(kind, resolve))

  const askWhenNeeded = async (): Promise<boolean> => {
    const found = await load()
    const kind = found === undefined ? undefined : readRequest(found)
    return kind === undefined ? false : ask(kind)
  }

  const retryAfterAsking = async <T>(
    call: () => Promise<Result<T, string>>,
  ): Promise<Result<T, string>> => {
    const first = await call()
    if (first.ok || !(await askWhenNeeded())) return first
    return call()
  }

  const settle = async (
    send: () => ReturnType<KeyFileApi['unlock']>,
  ): Promise<void> => {
    isBusy.value = true
    problem.value = ''
    try {
      const done = await send()
      if (!done.ok) {
        problem.value = KEY_FILE_WORDS.describeFailure(done.error)
        return
      }
      await load()
      close(true)
    } catch (caughtError) {
      problem.value = formatError(caughtError)
    } finally {
      isBusy.value = false
    }
  }

  const choose = (mode: 'plaintext' | 'passphrase', passphrase: string): Promise<void> =>
    settle(() => api.choose(mode, passphrase))

  const unlock = (passphrase: string): Promise<void> => settle(() => api.unlock(passphrase))

  const change = (): void => show('choose', () => undefined)

  return {
    state,
    request,
    problem,
    isBusy,
    load,
    askWhenNeeded,
    retryAfterAsking,
    dialog: { choose, unlock, change, cancel: () => close(false) },
  }
}

export type KeyFileStore = ReturnType<typeof createKeyFileStore>
