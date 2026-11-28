/**
 * The vaults a plugin keeps remotely: listed next to the local vaults, and opened in a new local folder
 * with one call.
 */
import { Code, ConnectError, createClient, type Client, type Transport } from '@connectrpc/connect'
import { type RemoteVault as ProtoRemoteVault, RemoteVaultService } from '@numen/protocol'
import { asFailure, asValue, type Failure, type Result } from '@numen/result'

/** A vault a plugin keeps remotely. */
export interface RemoteVault {
  readonly remoteId: string
  readonly name: string
  readonly sizeBytes: number
  readonly vaultId: string
}

/** Where a new vault of a name goes. */
export interface VaultPathSuggestion {
  readonly folder: string
  readonly path: string
  readonly isTaken: boolean
}

/** Why a call of the remote vaults failed. */
export type RemoteVaultErrorCode = 'invalidPath' | 'unavailable' | 'failed'

/** A failure with the reason the core gives. */
export type RemoteVaultFailure = Failure<RemoteVaultErrorCode>

/** The calls of RemoteVaultService the remote vaults are managed through. */
export type RemoteVaultCalls = Pick<
  Client<typeof RemoteVaultService>,
  'listRemoteVaults' | 'suggestVaultPath' | 'openRemoteVault'
>

/** The remote vaults of the application. */
export interface RemoteVaultRpc {
  /** Lists the remote vaults. A provider that is off or not installed gives an empty list. */
  listRemoteVaults(): Promise<Result<readonly RemoteVault[], RemoteVaultFailure>>
  /** Where a vault of `name` goes: in `folder` when given, else in the folder for new vaults. */
  suggestVaultPath(
    name: string,
    folder?: string,
  ): Promise<Result<VaultPathSuggestion, RemoteVaultFailure>>
  /** Makes the folder at the path, registers it as a vault, connects the remote vault and returns the vault id. */
  openRemoteVault(remoteId: string, path: string): Promise<Result<string, RemoteVaultFailure>>
}

const readVault = (vault: ProtoRemoteVault): RemoteVault => ({
  remoteId: vault.remoteId,
  name: vault.name,
  sizeBytes: Number(vault.sizeBytes),
  vaultId: vault.vaultId,
})

const readFailure = (thrown: unknown): RemoteVaultFailure => {
  const failure = ConnectError.from(thrown)
  switch (failure.code) {
    case Code.InvalidArgument:
      return { error: 'invalidPath', message: failure.rawMessage }
    case Code.FailedPrecondition:
    case Code.Unavailable:
      return { error: 'unavailable', message: failure.rawMessage }
    default:
      return { error: 'failed', message: failure.rawMessage }
  }
}

const answer = async <T>(call: () => Promise<T>): Promise<Result<T, RemoteVaultFailure>> => {
  try {
    return asValue(await call())
  } catch (thrown) {
    return asFailure(readFailure(thrown))
  }
}

/** The remote vaults managed through a RemoteVaultService client. */
export const createRemoteVaultClient = (calls: RemoteVaultCalls): RemoteVaultRpc => ({
  listRemoteVaults: async () =>
    answer(async () => (await calls.listRemoteVaults({})).vaults.map(readVault)),
  suggestVaultPath: async (name, folder) =>
    answer(async () => {
      const suggestion = await calls.suggestVaultPath({ name, folder: folder ?? '' })
      return { folder: suggestion.folder, path: suggestion.path, isTaken: suggestion.isTaken }
    }),
  openRemoteVault: async (remoteId, path) =>
    answer(async () => (await calls.openRemoteVault({ remoteId, path })).vaultId),
})

/** The remote vaults, called over the transport. */
export const createRemoteVaultRpc = (transport: Transport): RemoteVaultRpc =>
  createRemoteVaultClient(createClient(RemoteVaultService, transport))
