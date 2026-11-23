/**
 * The JSON messages of the sharing methods the plugin's backend serves, and the client that calls them.
 * Keys and meaning match the Go messages in backend/api.
 */
import type { Api, ApiError } from '@numen/plugin-sdk'

type ShareState = 'notShared' | 'creating' | 'active'
type ReaderState = 'active' | 'removed'

export interface ShareReader {
  readonly label: string
  readonly fingerprint: string
  readonly state: ReaderState
}

export interface Share {
  readonly state: ShareState
  readonly readers: readonly ShareReader[]
  readonly error?: string
}

interface ReaderCode {
  readonly code: string
  readonly fingerprint: string
}

interface ReaderCodeInfo {
  readonly fingerprint: string
  readonly label?: string
}

interface InvitationInfo {
  readonly shareName: string
  readonly ownerFingerprint: string
  readonly isForThisAccount: boolean
  readonly alreadyReceived: boolean
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

interface AddReaderRequest extends FolderRequest {
  readonly code: string
  readonly label: string
  readonly confirmChange?: boolean
}

interface RemoveReaderRequest extends FolderRequest {
  readonly fingerprint: string
}

interface AddedReader {
  readonly invitation: string
  readonly fingerprint: string
}

/** The calls of the owner of a folder: the share of the folder and the readers of it. */
export interface FolderSharesClient {
  readReaderCode(code: string): Promise<ReaderCodeInfo>
  getShare(request: FolderRequest): Promise<Share>
  addReader(request: AddReaderRequest): Promise<AddedReader>
  removeReader(request: RemoveReaderRequest): Promise<void>
  stopShare(request: FolderRequest): Promise<void>
}

/** The calls of a reader: the code of this account, the invitation and the folders received. */
export interface ReceivedSharesClient {
  getReaderCode(): Promise<ReaderCode>
  readInvitation(text: string): Promise<InvitationInfo>
  acceptInvitation(text: string): Promise<void>
  listReceived(): Promise<readonly ReceivedShare[]>
}

type Empty = Record<string, never>

const createFailure = (error: ApiError): Error => new Error(error.message)

async function call<Req, Res>(api: Api, method: string, request: Req): Promise<Res> {
  const result = await api.call<Req, Res>(method, request)
  if (!result.ok) throw createFailure(result.error)
  return result.value
}

export function createFolderSharesClient(api: Api): FolderSharesClient {
  return {
    readReaderCode: (code) =>
      call<{ code: string }, ReaderCodeInfo>(api, 'ReadReaderCode', { code }),
    getShare: (request) => call<FolderRequest, Share>(api, 'GetShare', request),
    addReader: (request) => call<AddReaderRequest, AddedReader>(api, 'AddReader', request),
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
    getReaderCode: () => call<Empty, ReaderCode>(api, 'GetReaderCode', {}),
    readInvitation: (text) =>
      call<{ text: string }, InvitationInfo>(api, 'ReadInvitation', { text }),
    acceptInvitation: async (text) => {
      await call<{ text: string }, Empty>(api, 'AcceptInvitation', { text })
    },
    listReceived: async () =>
      (await call<Empty, { shares: readonly ReceivedShare[] }>(api, 'ListReceived', {})).shares,
  }
}
