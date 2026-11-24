import type { KeyFileState } from '@/entities/key-file'
import { WORDS as words } from '../words'

/** The line that says where the keys are kept and how the file is protected. */
export function describeKeyFile(state: KeyFileState): string {
  if (!state.isInUse) return words.rowKeychain
  if (state.mode === 'plaintext') return words.rowPlaintext
  if (state.mode === 'passphrase') return state.isLocked ? words.rowLocked : words.rowPassphrase
  return words.rowUnset
}
