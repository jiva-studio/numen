export type { KeyFileFailure, KeyFileMode, KeyFileState } from '@numen/core-client'

/** What the person is asked: how to keep the keys, or the passphrase that opens them. */
export type KeyFileRequest = 'choose' | 'unlock'
