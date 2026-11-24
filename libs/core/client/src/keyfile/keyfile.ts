/**
 * Where a computer without a system keychain keeps the keys of the account: in a file, protected by a
 * passphrase or left as plain text. The passphrase crosses only in the calls that take it.
 */
import type { Client } from '@connectrpc/connect'
import {
  KeyFileErrorCode as ProtoKeyFileErrorCode,
  KeyFileMode as ProtoKeyFileMode,
  type KeyFileService,
} from '@numen/protocol'
import { asFailure, asValue, type Failure, type Result } from '@numen/result'

/** How the file keeps the keys. */
export type KeyFileMode = 'unset' | 'plaintext' | 'passphrase'

/** How the keys are kept now. */
export interface KeyFileState {
  readonly isInUse: boolean
  readonly mode: KeyFileMode
  readonly isChoiceNeeded: boolean
  readonly isLocked: boolean
  readonly warning: string
}

/** Why a call of the key file was refused. */
type KeyFileErrorCode = 'wrongPassphrase' | 'emptyPassphrase' | 'saveFailed' | 'invalidMode'

/** A served refusal with the reason the core gives. */
export type KeyFileFailure = Failure<KeyFileErrorCode>

/** The calls of KeyFileService the key file is managed through. */
export type KeyFileRpc = Pick<
  Client<typeof KeyFileService>,
  'getKeyFileState' | 'chooseKeyFile' | 'unlockKeyFile'
>

/** The key file of this computer. */
export interface KeyFileClient {
  getState(): Promise<KeyFileState>
  choose(mode: 'plaintext' | 'passphrase', passphrase: string): Promise<Result<undefined, KeyFileFailure>>
  unlock(passphrase: string): Promise<Result<undefined, KeyFileFailure>>
}

const readMode = (mode: ProtoKeyFileMode): KeyFileMode => {
  switch (mode) {
    case ProtoKeyFileMode.PLAINTEXT:
      return 'plaintext'
    case ProtoKeyFileMode.PASSPHRASE:
      return 'passphrase'
    case ProtoKeyFileMode.UNSET:
    case ProtoKeyFileMode.UNSPECIFIED:
      return 'unset'
  }
}

const writeMode = (mode: 'plaintext' | 'passphrase'): ProtoKeyFileMode =>
  mode === 'plaintext' ? ProtoKeyFileMode.PLAINTEXT : ProtoKeyFileMode.PASSPHRASE

const readErrorCode = (code: ProtoKeyFileErrorCode): KeyFileErrorCode | undefined => {
  switch (code) {
    case ProtoKeyFileErrorCode.UNSPECIFIED:
      return undefined
    case ProtoKeyFileErrorCode.WRONG_PASSPHRASE:
      return 'wrongPassphrase'
    case ProtoKeyFileErrorCode.EMPTY_PASSPHRASE:
      return 'emptyPassphrase'
    case ProtoKeyFileErrorCode.SAVE_FAILED:
      return 'saveFailed'
    case ProtoKeyFileErrorCode.INVALID_MODE:
      return 'invalidMode'
  }
}

const readAnswer = (answer: { error: ProtoKeyFileErrorCode }): Result<undefined, KeyFileFailure> => {
  const error = readErrorCode(answer.error)
  return error === undefined ? asValue(undefined) : asFailure({ error, message: error })
}

/** The key file managed through a KeyFileService client. */
export const createKeyFileClient = (rpc: KeyFileRpc): KeyFileClient => ({
  getState: async () => {
    const state = await rpc.getKeyFileState({})
    return {
      isInUse: state.isInUse,
      mode: readMode(state.mode),
      isChoiceNeeded: state.isChoiceNeeded,
      isLocked: state.isLocked,
      warning: state.warning,
    }
  },
  choose: async (mode, passphrase) =>
    readAnswer(await rpc.chooseKeyFile({ mode: writeMode(mode), passphrase })),
  unlock: async (passphrase) => readAnswer(await rpc.unlockKeyFile({ passphrase })),
})
