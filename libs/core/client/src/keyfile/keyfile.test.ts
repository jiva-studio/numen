import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import {
  ChooseKeyFileResponseSchema,
  GetKeyFileStateResponseSchema,
  ErrorCode,
  KeyFileMode,
  UnlockKeyFileResponseSchema,
} from '@numen/protocol'
import { createKeyFileClient, type KeyFileRpc } from './keyfile'

const createRpc = (overrides: Partial<KeyFileRpc> = {}): KeyFileRpc => ({
  getKeyFileState: () =>
    Promise.resolve(
      create(GetKeyFileStateResponseSchema, {
        isInUse: true,
        mode: KeyFileMode.UNSET,
        isChoiceNeeded: true,
        warning: 'No system keychain.',
      }),
    ),
  chooseKeyFile: () => Promise.resolve(create(ChooseKeyFileResponseSchema, {})),
  unlockKeyFile: () => Promise.resolve(create(UnlockKeyFileResponseSchema, {})),
  ...overrides,
})

describe('createKeyFileClient', () => {
  it('reads the state of the key file', async () => {
    const state = await createKeyFileClient(createRpc()).getState()

    expect(state).toEqual({
      isInUse: true,
      mode: 'unset',
      isChoiceNeeded: true,
      isLocked: false,
      warning: 'No system keychain.',
    })
  })

  it('sends the chosen mode with its passphrase', async () => {
    const sent: unknown[] = []
    const client = createKeyFileClient(
      createRpc({
        chooseKeyFile: (request) => {
          sent.push(request)
          return Promise.resolve(create(ChooseKeyFileResponseSchema, {}))
        },
      }),
    )

    const result = await client.choose('passphrase', 'moon river')

    expect(result.ok).toBe(true)
    expect(sent).toEqual([{ mode: KeyFileMode.PASSPHRASE, passphrase: 'moon river' }])
  })

  it('returns a wrong passphrase as a failure', async () => {
    const client = createKeyFileClient(
      createRpc({
        unlockKeyFile: () =>
          Promise.resolve(
            create(UnlockKeyFileResponseSchema, { error: ErrorCode.WRONG_PASSPHRASE }),
          ),
      }),
    )

    const result = await client.unlock('wrong')

    expect(result.ok).toBe(false)
    expect(!result.ok && result.error.error).toBe('wrongPassphrase')
  })
})
