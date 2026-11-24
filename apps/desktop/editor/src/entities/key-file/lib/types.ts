import type { Failure } from '@numen/result'

/** How the keys of this computer are kept where it has no system keychain. */
export type KeyFileMode = 'unset' | 'plaintext' | 'passphrase'

/** How the keys are kept now. */
export interface KeyFileState {
  readonly isInUse: boolean
  readonly mode: KeyFileMode
  readonly isChoiceNeeded: boolean
  readonly isLocked: boolean
  readonly warning: string
}

/** A refusal of the core with the reason it gives. */
export type KeyFileFailure = Failure<
  'wrongPassphrase' | 'emptyPassphrase' | 'saveFailed' | 'invalidMode'
>

/** What the person is asked: how to keep the keys, or the passphrase that opens them. */
export type KeyFileRequest = 'choose' | 'unlock'
