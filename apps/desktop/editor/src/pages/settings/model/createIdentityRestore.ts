/**
 * The restore of the identity on this device from the words or sheets the person typed.
 */
import type { Ref } from 'vue'
import type { Result } from '@numen/result'
import type { KitProof } from '@/entities/identity'
import type { InstallationIdentity } from '../types'
import { WORDS as words } from '../words'

type Run = <T>(call: () => Promise<Result<T, string>>) => Promise<Result<T, string>>

export function createIdentityRestore(
  getIdentity: () => InstallationIdentity,
  run: Run,
  readProof: () => KitProof | undefined,
  problem: Ref<string>,
) {
  /** Whether the identity was restored. */
  const restore = async (): Promise<boolean> => {
    const given = readProof()
    if (given === undefined) {
      problem.value = words.kitOneMoreSheet
      return false
    }
    const answer = await run(() => getIdentity().restore(given))
    return answer.ok
  }

  return { restore }
}
