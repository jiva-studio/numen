/** What the key file store reports when the core refuses a call or a read fails. */
import type { KeyFileFailure } from './lib/types'

const FAILURE_TEXTS: Readonly<Record<KeyFileFailure['error'], string>> = {
  wrongPassphrase: 'This passphrase does not open your keys. Try again.',
  emptyPassphrase: 'Enter a passphrase.',
  saveFailed: 'The choice could not be saved. Try again.',
  invalidMode: 'Choose one of the two ways to keep your keys.',
}

export const KEY_FILE_WORDS = {
  loadFailed: 'Where your keys are kept could not be read.',
  describeFailure: (failure: KeyFileFailure): string => FAILURE_TEXTS[failure.error],
}
