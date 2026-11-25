/**
 * The JSON messages of the invitation methods the plugin's backend serves and the clients that call them: the
 * sharer starts a link and answers the requests it brings, the person who opens the link asks to read.
 * Keys and meaning match the Go messages in backend/api.
 */
import type { Api } from '@numen/plugin-sdk'
import { call, type Empty } from './call'

export interface FolderTarget {
  readonly vaultId: string
  readonly folder: string
}

interface StartedInvitation {
  readonly id: string
  readonly link: string
  readonly expiresAt: string
}

interface ShareInvitation {
  readonly id: string
  readonly folder: string
  readonly expiresAt: string
  readonly isRequested: boolean
}

export interface ShareRequest {
  readonly requestId: string
  readonly vaultId: string
  readonly folder: string
  readonly deviceName: string
  readonly fingerprint: string
  readonly code: string
}

export interface InvitationPreview {
  readonly folder: string
  readonly ownerFingerprint: string
  readonly expiresAt: string
  readonly isReadOnly: boolean
}

interface AcceptRequest {
  readonly link: string
  readonly deviceName: string
}

interface SentRequest {
  readonly requestId: string
  readonly folder: string
  readonly ownerFingerprint: string
}

export type RequestPhase =
  'connecting' | 'waiting' | 'received' | 'declined' | 'used' | 'expired' | 'failed'

export interface RequestStatus {
  readonly phase: RequestPhase
  readonly code?: string
  readonly folder: string
  readonly ownerFingerprint: string
  readonly expiresAt: string
  readonly error?: string
}

/** The calls of the sharer: the link of a folder and the requests that come through it. */
export interface ShareInvitationClient {
  startShareInvitation(target: FolderTarget): Promise<StartedInvitation>
  listShareInvitations(): Promise<readonly ShareInvitation[]>
  revokeShareInvitation(id: string): Promise<void>
  listShareRequests(): Promise<readonly ShareRequest[]>
  approveShareRequest(requestId: string): Promise<void>
  declineShareRequest(requestId: string): Promise<void>
}

/** The calls of the person who got a link: what it offers, the request to read and how it stands. */
export interface AcceptInvitationClient {
  readSharePreview(link: string): Promise<InvitationPreview>
  acceptShareInvitation(request: AcceptRequest): Promise<SentRequest>
  getShareInvitationStatus(requestId: string): Promise<RequestStatus>
}

export function createShareInvitationClient(api: Api): ShareInvitationClient {
  return {
    startShareInvitation: (target) =>
      call<FolderTarget, StartedInvitation>(api, 'StartShareInvitation', target),
    listShareInvitations: async () =>
      (
        await call<Empty, { invitations: readonly ShareInvitation[] }>(
          api,
          'ListShareInvitations',
          {},
        )
      ).invitations,
    revokeShareInvitation: async (id) => {
      await call<{ id: string }, Empty>(api, 'RevokeShareInvitation', { id })
    },
    listShareRequests: async () =>
      (await call<Empty, { requests: readonly ShareRequest[] }>(api, 'ListShareRequests', {}))
        .requests,
    approveShareRequest: async (requestId) => {
      await call<{ requestId: string }, Empty>(api, 'ApproveShareRequest', { requestId })
    },
    declineShareRequest: async (requestId) => {
      await call<{ requestId: string }, Empty>(api, 'DeclineShareRequest', { requestId })
    },
  }
}

export function createAcceptInvitationClient(api: Api): AcceptInvitationClient {
  return {
    readSharePreview: (link) =>
      call<{ link: string }, InvitationPreview>(api, 'PreviewShareInvitation', { link }),
    acceptShareInvitation: (request) =>
      call<AcceptRequest, SentRequest>(api, 'AcceptShareInvitation', request),
    getShareInvitationStatus: (requestId) =>
      call<{ requestId: string }, RequestStatus>(api, 'GetShareInvitationStatus', { requestId }),
  }
}
