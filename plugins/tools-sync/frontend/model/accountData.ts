import { computed, onScopeDispose, ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { AccountClient } from '../api/account'
import { WORDS as words } from '../words'

const REFRESH_MS = 10_000

/** Writes the text of a file to a place the person chooses, and answers false when the person chose none. */
export type SaveFile = (name: string, content: string) => Promise<boolean>

export interface AccountDataOptions {
  readonly client: AccountClient
  readonly saveFile: SaveFile
  readonly refreshMs?: number | undefined
}

const getMessage = (failure: unknown): string =>
  failure instanceof Error ? failure.message : String(failure)

/**
 * The deletion of the account as the sign-in of this device knows it, kept up to date; the deletion of the cloud
 * data, the restore of an account that is being deleted and the export of what the server holds. Each action
 * answers false when it failed, and the reason is in `problem`.
 */
export function useAccountData(options: AccountDataOptions) {
  const { client, saveFile, refreshMs = REFRESH_MS } = options
  const account = ref('')
  const isDeleting = ref(false)
  const deleteAt = ref('')
  const isBusy = ref(false)
  const problem = ref('')
  let timer: ReturnType<typeof setTimeout> | undefined
  let isDisposed = false

  async function read(): Promise<void> {
    const state = await client.getAccountState()
    account.value = state.hasIdentity ? state.account : ''
    isDeleting.value = state.hasIdentity && state.isDeleting
    deleteAt.value = isDeleting.value ? state.deleteAt : ''
  }

  function schedule(): void {
    clearTimeout(timer)
    if (isDisposed) return
    timer = setTimeout(() => runInBackground(load(), (failure) => (problem.value = getMessage(failure))), refreshMs)
  }

  /** Reads the state and keeps reading it. An unreadable state leaves the last one and says nothing: the card tells that the server is away. */
  async function load(): Promise<void> {
    try {
      await read()
    } catch {
      /* the subscription of the card reports the server that is away */
    } finally {
      schedule()
    }
  }

  async function run(action: () => Promise<void>): Promise<boolean> {
    problem.value = ''
    isBusy.value = true
    try {
      await action()
      return true
    } catch (failure) {
      problem.value = getMessage(failure)
      return false
    } finally {
      isBusy.value = false
    }
  }

  const remove = (): Promise<boolean> =>
    run(async () => {
      deleteAt.value = await client.deleteCloudData()
      isDeleting.value = true
    })

  const restore = (): Promise<boolean> =>
    run(async () => {
      await client.restoreAccount()
      isDeleting.value = false
      deleteAt.value = ''
    })

  const exportData = (): Promise<boolean> =>
    run(async () => {
      await saveFile(words.exportFileName, await client.exportAccountData())
    })

  onScopeDispose(() => {
    isDisposed = true
    clearTimeout(timer)
  })

  return {
    account,
    isDeleting,
    deleteAt: computed(() => deleteAt.value),
    isBusy,
    problem,
    load,
    remove,
    restore,
    exportData,
  }
}
