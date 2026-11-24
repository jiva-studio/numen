/** The words of the questions about where the keys are kept. */
// Stryker disable StringLiteral: interface copy
export const WORDS = {
  chooseTitle: 'No system keychain on this computer',
  chooseHint:
    'Numen keeps your keys in the system keychain. This computer has none, so they must be kept in a file.',
  choiceLabel: 'How to keep your keys',
  passphraseTitle: 'Protect with a passphrase (recommended)',
  passphraseRule:
    'You need this passphrase every time you start Numen. If you lose it, restore from your recovery words.',
  plaintextTitle: 'Store in a plain file (not recommended)',
  plaintextWarning: 'Anyone who can read your files can read your keys.',
  passphraseField: 'Passphrase',
  confirmField: 'Repeat the passphrase',
  mismatch: 'The two passphrases are not the same.',
  continue: 'Continue',
  saving: 'Saving…',
  cancel: 'Cancel',
  unlockTitle: 'Enter your passphrase',
  unlockHint: 'Numen needs it to open your keys.',
  unlockAction: 'Unlock',
  unlocking: 'Unlocking…',
  notNow: 'Not now',
  rowTitle: 'Where keys are kept',
  rowChange: 'Change',
  rowUnlock: 'Enter passphrase',
  rowKeychain: 'In the system keychain of this computer.',
  rowUnset: 'Not chosen yet. Numen asks when it first needs to keep a key.',
  rowPassphrase: 'In a file protected by your passphrase.',
  rowLocked: 'In a file protected by your passphrase. Enter it to open your keys.',
  rowPlaintext: 'In a plain file. Anyone who can read your files can read your keys.',
}
