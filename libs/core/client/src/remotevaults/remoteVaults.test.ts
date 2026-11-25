import { describe, expect, it } from 'vitest'
import { create } from '@bufbuild/protobuf'
import { Code, ConnectError } from '@connectrpc/connect'
import {
  OpenRemoteVaultResponseSchema,
  RemoteVaultSchema,
  RemoteVaultServiceListRemoteVaultsResponseSchema,
  SuggestVaultPathResponseSchema,
} from '@numen/protocol'
import { createRemoteVaultClient, type RemoteVaultCalls } from './remoteVaults'

const createCalls = (overrides: Partial<RemoteVaultCalls> = {}): RemoteVaultCalls => ({
  listRemoteVaults: () =>
    Promise.resolve(
      create(RemoteVaultServiceListRemoteVaultsResponseSchema, {
        vaults: [
          create(RemoteVaultSchema, { remoteId: 'r-1', name: 'Notes', sizeBytes: 2048n, vaultId: 'v-1' }),
          create(RemoteVaultSchema, { remoteId: 'r-2', name: 'Books' }),
        ],
      }),
    ),
  suggestVaultPath: () =>
    Promise.resolve(
      create(SuggestVaultPathResponseSchema, { folder: '/docs', path: '/docs/Notes 2', isTaken: true }),
    ),
  openRemoteVault: () => Promise.resolve(create(OpenRemoteVaultResponseSchema, { vaultId: 'v-9' })),
  ...overrides,
})

describe('createRemoteVaultClient', () => {
  it('lists the remote vaults with their sizes and the vaults they are connected to', async () => {
    const result = await createRemoteVaultClient(createCalls()).listRemoteVaults()

    expect(result).toEqual({
      ok: true,
      value: [
        { remoteId: 'r-1', name: 'Notes', sizeBytes: 2048, vaultId: 'v-1' },
        { remoteId: 'r-2', name: 'Books', sizeBytes: 0, vaultId: '' },
      ],
    })
  })

  it('reads the suggested folder, path and whether the name is taken', async () => {
    const sent: unknown[] = []
    const client = createRemoteVaultClient(
      createCalls({
        suggestVaultPath: (request) => {
          sent.push(request)
          return Promise.resolve(
            create(SuggestVaultPathResponseSchema, {
              folder: '/docs',
              path: '/docs/Notes 2',
              isTaken: true,
            }),
          )
        },
      }),
    )

    const result = await client.suggestVaultPath('Notes')

    expect(sent).toEqual([{ name: 'Notes' }])
    expect(result).toEqual({
      ok: true,
      value: { folder: '/docs', path: '/docs/Notes 2', isTaken: true },
    })
  })

  it('sends the remote vault with the path and returns the vault id', async () => {
    const sent: unknown[] = []
    const client = createRemoteVaultClient(
      createCalls({
        openRemoteVault: (request) => {
          sent.push(request)
          return Promise.resolve(create(OpenRemoteVaultResponseSchema, { vaultId: 'v-9' }))
        },
      }),
    )

    const result = await client.openRemoteVault('r-1', '/docs/Notes')

    expect(sent).toEqual([{ remoteId: 'r-1', path: '/docs/Notes' }])
    expect(result).toEqual({ ok: true, value: 'v-9' })
  })

  it('returns a path the core refuses as an invalid path with its message', async () => {
    const client = createRemoteVaultClient(
      createCalls({
        openRemoteVault: () =>
          Promise.reject(new ConnectError('/docs/Notes is not empty', Code.InvalidArgument)),
      }),
    )

    const result = await client.openRemoteVault('r-1', '/docs/Notes')

    expect(result).toEqual({
      ok: false,
      error: { error: 'invalidPath', message: '/docs/Notes is not empty' },
    })
  })

  it('returns a provider that cannot answer as unavailable and any other failure as failed', async () => {
    const unavailable = createRemoteVaultClient(
      createCalls({
        openRemoteVault: () =>
          Promise.reject(new ConnectError('Sign in to open the vault', Code.FailedPrecondition)),
      }),
    )
    const broken = createRemoteVaultClient(
      createCalls({
        openRemoteVault: () => Promise.reject(new ConnectError('boom', Code.Internal)),
      }),
    )

    expect(await unavailable.openRemoteVault('r-1', '/p')).toEqual({
      ok: false,
      error: { error: 'unavailable', message: 'Sign in to open the vault' },
    })
    expect(await broken.openRemoteVault('r-1', '/p')).toEqual({
      ok: false,
      error: { error: 'failed', message: 'boom' },
    })
  })
})
