/**
 * The vaults kept in the cloud, in the shape the window requests them.
 */
import type { Result } from '@numen/result'
import { formatError } from '@/shared/api/errors'
import type { WindowClients } from '@/shared/api/clients'
import type { CloudVault, VaultPlace } from '../lib/cloudVault'

/** What the window asks of the cloud vaults. A failure carries the sentence the person reads. */
export interface RemoteVaults {
  /** The cloud vaults; none where the cloud is off, signed out or out of reach. */
  list(): Promise<readonly CloudVault[]>
  suggestPath(name: string): Promise<Result<VaultPlace>>
  /** Starts getting the vault into `path` and answers with the id of the local vault. */
  open(remoteId: string, path: string): Promise<Result<string>>
}

interface Failed {
  readonly ok: false
  readonly error: { readonly message: string }
}

const asFailed = (failure: Failed): Result<never> => ({ ok: false, error: failure.error.message })

const guard = async <T>(call: () => Promise<Result<T>>): Promise<Result<T>> => {
  try {
    return await call()
  } catch (error) {
    return { ok: false, error: formatError(error) }
  }
}

/** The cloud vaults over `client`. */
export const createRemoteVaults = (client: WindowClients['remoteVaults']): RemoteVaults => ({
  list: async () => {
    try {
      const answer = await client.listRemoteVaults()
      if (!answer.ok) return []
      return answer.value
    } catch {
      return []
    }
  },
  suggestPath: (name) =>
    guard(async () => {
      const answer = await client.suggestVaultPath(name)
      if (!answer.ok) return asFailed(answer)
      const { folder, path, isTaken } = answer.value
      return { ok: true, value: { folder, path, isTaken } }
    }),
  open: (remoteId, path) =>
    guard(async () => {
      const answer = await client.openRemoteVault(remoteId, path)
      if (!answer.ok) return asFailed(answer)
      return { ok: true, value: answer.value }
    }),
})
