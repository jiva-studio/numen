/**
 * The JSON messages of the sharing methods the plugin's backend serves, and the clients that call them.
 * Keys and meaning match the Go messages in backend/api.
 */
import type { Api } from '@numen/plugin-sdk'
import { call, type Empty } from './call'

type ShareState = 'notShared' | 'creating' | 'active'

interface ShareReader {
  readonly label: string
  readonly fingerprint: string
  readonly state: 'active' | 'removed'
}

export interface SharedFolder {
  readonly vaultId: string
  readonly folder: string
  readonly state: ShareState
  readonly readers: readonly ShareReader[]
}

export type ReceivedState = 'receiving' | 'ready' | 'ended'

export interface ReceivedShare {
  readonly vaultId: string
  readonly name: string
  readonly ownerFingerprint: string
  readonly state: ReceivedState
  readonly lastSyncedAt: string
}

interface FolderRequest {
  readonly vaultId: string
  readonly folder: string
}

interface LeaveShareRequest {
  readonly vaultId: string
}

interface RemoveReaderRequest extends FolderRequest {
  readonly fingerprint: string
}

/** The calls of the owner of a folder: the folders shared and the readers of them. */
export interface FolderSharesClient {
  listShares(): Promise<readonly SharedFolder[]>
  removeReader(request: RemoveReaderRequest): Promise<void>
  stopShare(request: FolderRequest): Promise<void>
}

/** The calls of a reader: the folders received. */
export interface ReceivedSharesClient {
  listReceived(): Promise<readonly ReceivedShare[]>
  leaveShare(request: LeaveShareRequest): Promise<void>
}

export function createFolderSharesClient(api: Api): FolderSharesClient {
  return {
    listShares: async () =>
      (await call<Empty, { shares: readonly SharedFolder[] }>(api, 'ListShares', {})).shares,
    removeReader: async (request) => {
      await call<RemoveReaderRequest, Empty>(api, 'RemoveReader', request)
    },
    stopShare: async (request) => {
      await call<FolderRequest, Empty>(api, 'StopShare', request)
    },
  }
}

export function createReceivedSharesClient(api: Api): ReceivedSharesClient {
  return {
    listReceived: async () =>
      (await call<Empty, { shares: readonly ReceivedShare[] }>(api, 'ListReceived', {})).shares,
    leaveShare: async (request) => {
      await call<LeaveShareRequest, Empty>(api, 'LeaveShare', request)
    },
  }
}
