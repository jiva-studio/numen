/**
 * The three ways to restore the identity: the twelve typed words, two scanned sheets and two typed
 * sheets. Each keeps what was entered; a whole proof restores through the core and `onRestored` runs
 * once it has.
 */
import type { Ref } from 'vue'
import type { Result } from '@numen/result'
import type { InstallationIdentity } from '../types'
import { createIdentityRestore } from './createIdentityRestore'
import { readWordsOnly } from './readProof'
import { useRecoveryProof } from './useRecoveryProof'
import { useSheetScan } from './useSheetScan'
import { useTypedSheets } from './useTypedSheets'

type Run = <T>(call: () => Promise<Result<T, string>>) => Promise<Result<T, string>>

interface RestoreSourcesOptions {
  getIdentity: () => InstallationIdentity
  run: Run
  problem: Ref<string>
  isBusy: () => boolean
  onRestored: () => void
}

export function useRestoreSources(options: RestoreSourcesOptions) {
  const { getIdentity, run, problem, isBusy, onRestored } = options
  const words = useRecoveryProof(problem, readWordsOnly)
  const typed = useTypedSheets(problem)
  const identityRestore = createIdentityRestore(getIdentity, run, words.read)
  const scanning = useSheetScan(problem, async (proof) => {
    if (await identityRestore.restore(proof)) onRestored()
  })

  const restoreWords = async (): Promise<void> => {
    if (!isBusy() && (await identityRestore.restore())) onRestored()
  }

  const restoreTyped = async (): Promise<void> => {
    if (!isBusy() && (await identityRestore.restore(typed.read()))) onRestored()
  }

  const clear = (): void => {
    words.clear()
    typed.clear()
    scanning.clear()
  }

  return { words, typed, scanning, restoreWords, restoreTyped, clear }
}
