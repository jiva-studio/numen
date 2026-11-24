/** The calls of the invitation of a folder: the sharer starts it, the person who opens the link asks to read, the sharer allows it. */

export type InvitationState = 'waiting' | 'approved' | 'declined' | 'expired'

export interface ShareRequest {
  readonly requestId: string
  readonly deviceName: string
  readonly fingerprint: string
  readonly folderName: string
  readonly code: string
}

export interface SharePerson {
  readonly deviceName: string
  readonly fingerprint: string
}

interface FolderTarget {
  readonly vaultId: string
  readonly folder: string
}

interface StartedInvitation {
  readonly link: string
  readonly expiresAt: string
}

interface InvitationStatus {
  readonly state: InvitationState
  readonly requests: readonly ShareRequest[]
  readonly people: readonly SharePerson[]
}

/** What the dialog "Share..." asks of the plugin's backend. */
export interface ShareInvitationClient {
  startShareInvitation(target: FolderTarget): Promise<StartedInvitation>
  getInvitationStatus(target: FolderTarget): Promise<InvitationStatus>
  approveShareRequest(requestId: string): Promise<void>
  declineShareRequest(requestId: string): Promise<void>
  revokeShareInvitation(target: FolderTarget): Promise<void>
}
