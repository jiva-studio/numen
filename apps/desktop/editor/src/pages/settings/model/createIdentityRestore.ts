/**
 * The restore of the identity on this device from the words or sheets the person entered.
 */
import type { Result } from '@numen/result'
import type { KitProof } from '@/entities/identity'
import type { InstallationIdentity } from '../types'

type Run = <T>(call: () => Promise<Result<T, string>>) => Promise<Result<T, string>>

export function createIdentityRestore(
  getIdentity: () => InstallationIdentity,
  run: Run,
  readProof: () => KitProof | undefined,
) {
  /** Whether the identity was restored; nothing is asked of the core until the input is a whole proof. */
  const restore = async (
    proof: KitProof | undefined = readProof(),
    signal?: AbortSignal,
  ): Promise<boolean> => {
    const given = proof
    if (given === undefined) return false
    const answer = await run(() => getIdentity().recovery.restore(given, signal))
    return answer.ok
  }

  return { restore }
}
