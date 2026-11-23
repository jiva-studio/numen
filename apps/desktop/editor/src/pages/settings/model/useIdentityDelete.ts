/**
 * The deletion of the identity from this device: asking the person first, then deleting.
 */
import { ref } from 'vue'
import type { Result } from '@numen/result'
import type { InstallationIdentity } from '../types'

type Run = <T>(call: () => Promise<Result<T, string>>) => Promise<Result<T, string>>

export function useIdentityDelete(
  getIdentity: () => InstallationIdentity,
  run: Run,
  isBusy: () => boolean,
) {
  const isAskingDelete = ref(false)

  const askDelete = (): void => {
    isAskingDelete.value = true
  }

  const cancelDelete = (): void => {
    isAskingDelete.value = false
  }

  const confirmDelete = async (): Promise<void> => {
    if (isBusy()) return
    isAskingDelete.value = false
    await run(() => getIdentity().delete())
  }

  return { isAskingDelete, askDelete, cancelDelete, confirmDelete }
}
