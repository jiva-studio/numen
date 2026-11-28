/** Whether the passphrase and its repeat are the same and not empty. */
export const isPassphraseSet = (passphrase: string, confirmation: string): boolean =>
  passphrase !== '' && passphrase === confirmation

/** Whether the repeat has started and differs from the passphrase. */
export const isPassphraseMismatch = (passphrase: string, confirmation: string): boolean =>
  confirmation !== '' && passphrase !== confirmation
