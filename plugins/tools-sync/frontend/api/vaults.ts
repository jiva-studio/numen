/**
 * The JSON messages of the vaults service the plugin's backend serves, and the client that calls it.
 * Keys and meaning match the Go messages in backend/api; testdata/api holds the fixtures both sides
 * are tested against.
 */
import type { Api, ApiError } from '@numen/plugin-sdk'

import { REQUIREMENT_UNMET } from './call'

export const SERVICE = 'md.numen.tools.sync'

export type VaultState = 'notConnected' | 'connected' | 'copy' | 'received'

type DeletionsDirection = 'send' | 'apply'

export type DeletionsChoice = 'remove' | 'keep'

export interface PausedDeletions {
  readonly count: number
  readonly tracked: number
  readonly direction: DeletionsDirection
}

export interface Vault {
  readonly id: string
  readonly name: string
  readonly path: string
  readonly state: VaultState
  readonly lastSyncedAt: string
  readonly conflictCount: number
  readonly pausedDeletions?: PausedDeletions | null
}

export interface RemoteVault {
  readonly remoteId: string
  readonly name: string
  readonly isConnected: boolean
}

interface StatusItem {
  readonly key: string
  readonly text: string
}

type Empty = Record<string, never>

interface ListVaultsResponse {
  readonly vaults: readonly Vault[]
}

interface ListRemoteVaultsResponse {
  readonly vaults: readonly RemoteVault[]
}

interface ConnectVaultRequest {
  readonly vaultId: string
  readonly mode: 'new' | 'existing'
  readonly remoteId?: string
}

interface PreviewConnectRequest {
  readonly vaultId: string
  readonly remoteId: string
}

interface PreviewConnectResponse {
  readonly differCount: number
}

interface ResolveDeletionsRequest {
  readonly vaultId: string
  readonly choice: DeletionsChoice
}

interface StopSyncingVaultRequest {
  readonly vaultId: string
}

interface GetStatusResponse {
  readonly hasIdentity: boolean
  readonly isReachable: boolean
  readonly items: readonly StatusItem[]
}

export interface VaultsClient {
  listVaults(): Promise<ListVaultsResponse>
  listRemoteVaults(): Promise<ListRemoteVaultsResponse>
  getConnectPreview(request: PreviewConnectRequest): Promise<PreviewConnectResponse>
  connectVault(request: ConnectVaultRequest): Promise<void>
  resolveDeletions(request: ResolveDeletionsRequest): Promise<void>
  getStatus(): Promise<GetStatusResponse>
  stopSyncingVault(request: StopSyncingVaultRequest): Promise<void>
}

const NO_IDENTITY: GetStatusResponse = { hasIdentity: false, isReachable: false, items: [] }

const createFailure = (error: ApiError): Error => new Error(error.message)

export function createVaultsClient(api: Api): VaultsClient {
  async function call<Req, Res>(method: string, request: Req, ifNoIdentity?: Res): Promise<Res> {
    const result = await api.call<Req, Res>(method, request)
    if (!result.ok && ifNoIdentity !== undefined && result.error.code === REQUIREMENT_UNMET) {
      return ifNoIdentity
    }
    if (!result.ok) throw createFailure(result.error)
    return result.value
  }

  return {
    listVaults: () => call<Empty, ListVaultsResponse>('ListVaults', {}),
    listRemoteVaults: () => call<Empty, ListRemoteVaultsResponse>('ListRemoteVaults', {}),
    getConnectPreview: (request) =>
      call<PreviewConnectRequest, PreviewConnectResponse>('PreviewConnect', request),
    connectVault: async (request) => {
      await call<ConnectVaultRequest, Empty>('ConnectVault', request)
    },
    resolveDeletions: async (request) => {
      await call<ResolveDeletionsRequest, Empty>('ResolveDeletions', request)
    },
    getStatus: () => call<Empty, GetStatusResponse>('GetStatus', {}, NO_IDENTITY),
    stopSyncingVault: async (request) => {
      await call<StopSyncingVaultRequest, Empty>('StopSyncingVault', request)
    },
  }
}
