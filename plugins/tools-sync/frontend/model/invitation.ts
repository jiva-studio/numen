import { getCurrentScope, onScopeDispose, reactive, ref, shallowRef } from 'vue'
import type { FolderSharesClient } from '../api/shares'
import type { FolderTarget, ShareInvitationClient, ShareRequest } from '../api/invitations'
import { getErrorMessage } from './vaults'
import { createPoller } from './polling'
import { getTimeLeftText } from '../lib/expiry'

const REFRESH_MS = 2000

export interface InvitationViewOptions {
  readonly invitations: ShareInvitationClient
  readonly shares: FolderSharesClient
  readonly target: FolderTarget
  readonly vaultName?: string
  readonly close: () => void
  readonly now: () => Date
  readonly refreshMs?: number
}

export interface SharePerson {
  readonly deviceName: string
  readonly fingerprint: string
}

/** What the dialog "Share..." holds: the link to send, the device that asks to read, and the devices that read. */
export function createInvitationView(options: InvitationViewOptions) {
  const {
    invitations,
    shares,
    target,
    close,
    now,
    vaultName = '',
    refreshMs = REFRESH_MS,
  } = options
  const name = target.folder.split('/').at(-1) || vaultName
  const id = ref('')
  const link = ref('')
  const expiresAt = ref('')
  const isStarting = ref(true)
  const isLinkAlive = ref(false)
  const request = shallowRef<ShareRequest | undefined>(undefined)
  const people = shallowRef<readonly SharePerson[]>([])
  const error = ref('')
  const isBusy = ref(false)
  const poller = createPoller(refreshMs, fail)

  if (getCurrentScope() !== undefined) onScopeDispose(poller.stop)

  function fail(failure: unknown): void {
    error.value = getErrorMessage(failure)
  }

  const isTarget = (one: FolderTarget): boolean =>
    one.vaultId === target.vaultId && one.folder === target.folder

  async function refresh(): Promise<void> {
    try {
      const [alive, requests, shared] = await Promise.all([
        invitations.listShareInvitations(),
        invitations.listShareRequests(),
        shares.listShares(),
      ])
      isLinkAlive.value = alive.some((one) => one.id === id.value)
      request.value = requests.find(isTarget)
      people.value = (shared.find(isTarget)?.readers ?? [])
        .filter((one) => one.state === 'active')
        .map((one) => ({ deviceName: one.label, fingerprint: one.fingerprint }))
      error.value = ''
      poller.countSuccess()
    } catch (failure) {
      fail(failure)
      if (poller.countFailure()) return
    }
    if (isLinkAlive.value || request.value !== undefined) poller.runAgain(refresh)
  }

  async function start(): Promise<void> {
    try {
      const started = await invitations.startShareInvitation(target)
      id.value = started.id
      link.value = started.link
      expiresAt.value = started.expiresAt
    } catch (failure) {
      fail(failure)
      isStarting.value = false
      return
    }
    isStarting.value = false
    await refresh()
  }

  async function answer(work: () => Promise<void>): Promise<void> {
    isBusy.value = true
    try {
      await work()
      await refresh()
    } catch (failure) {
      fail(failure)
    } finally {
      isBusy.value = false
    }
  }

  const answerRequest = (reply: (requestId: string) => Promise<void>): Promise<void> =>
    answer(async () => {
      if (request.value !== undefined) await reply(request.value.requestId)
    })

  return reactive({
    target,
    name,
    link,
    expiresAt,
    isStarting,
    isLinkAlive,
    request,
    people,
    error,
    isBusy,
    start,
    fail,
    approve: (): Promise<void> => answerRequest((one) => invitations.approveShareRequest(one)),
    decline: (): Promise<void> => answerRequest((one) => invitations.declineShareRequest(one)),
    revoke: (): Promise<void> => answer(() => invitations.revokeShareInvitation(id.value)),
    removeAccess: (person: SharePerson): Promise<void> =>
      answer(() => shares.removeReader({ ...target, fingerprint: person.fingerprint })),
    getTimeLeft: (): string => getTimeLeftText(expiresAt.value, now().getTime()),
    copyLink: (): Promise<void> => globalThis.navigator.clipboard.writeText(link.value),
    stop: poller.stop,
    close,
  })
}

export type InvitationView = ReturnType<typeof createInvitationView>
