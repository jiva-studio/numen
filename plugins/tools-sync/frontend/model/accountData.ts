import { formatErrorMessage } from '@numen/plugin-sdk'
import { computed, onScopeDispose, ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { AccountClient } from '../api/account'
import type { SaveFile } from '../api/saveFile'
import { WORDS as words } from '../words'

const REFRESH_MS = 10_000

export interface AccountDataOptions {
  readonly client: AccountClient
  readonly saveFile: SaveFile
  readonly refreshMs?: number | undefined
}

const getMessage = (failure: unknown): string =>
  failure instanceof Error ? failure.message : formatErrorMessage(failure)

/**
 * The deletion of the identity as the sign-in of this device knows it, kept up to date; the deletion of the cloud
 * data, the restore of an account that is being deleted and the export of what the server holds. Each action
 * answers false when it failed, and the reason is in `problem`.
 */
export function useAccountData(options: AccountDataOptions) {
  const { client, saveFile, refreshMs = REFRESH_MS } = options
  const isDeleting = ref(false)
  const deleteAt = ref('')
  const isBusy = ref(false)
  const problem = ref('')
  const isExporting = ref(false)
  let timer: ReturnType<typeof setTimeout> | undefined
  let isDisposed = false

  async function read(): Promise<void> {
    const state = await client.getAccountState()
    isDeleting.value = state.hasIdentity && state.isDeleting
    deleteAt.value = isDeleting.value ? state.deleteAt : ''
  }

  function schedule(): void {
    clearTimeout(timer)
    if (isDisposed) return
    const report = (failure: unknown): void => {
      problem.value = getMessage(failure)
    }
    timer = setTimeout(() => runInBackground(load(), report), refreshMs)
  }

  /** Reads the state and keeps reading it. It answers false when the state is unreadable and the last one stays. */
  async function load(): Promise<boolean> {
    try {
      await read()
      return true
    } catch {
      return false
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
      isExporting.value = true
      try {
        await saveFile(words.exportFileName, await client.exportAccountData())
      } finally {
        isExporting.value = false
      }
    })

  onScopeDispose(() => {
    isDisposed = true
    clearTimeout(timer)
  })

  return {
    state: {
      isDeleting,
      deleteAt: computed(() => deleteAt.value),
      isBusy,
      problem,
      isExporting,
    },
    actions: { load, remove, restore, exportData },
  }
}
