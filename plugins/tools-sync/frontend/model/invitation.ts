import { getCurrentScope, onScopeDispose, reactive, ref, shallowRef } from 'vue'
import type {
  InvitationState,
  SharePerson,
  ShareInvitationClient,
  ShareRequest,
} from '../api/invitations'
import { getErrorMessage } from './vaults'

const REFRESH_MS = 2000

export interface InvitationViewOptions {
  readonly client: ShareInvitationClient
  readonly target: { readonly vaultId: string; readonly folder: string }
  readonly close: () => void
  readonly refreshMs?: number
}

type InvitationPhase = 'starting' | InvitationState

/** What the dialog "Share..." holds: the link to send, who waits for an answer, and who was allowed. */
export function createInvitationView(options: InvitationViewOptions) {
  const { client, target, close, refreshMs = REFRESH_MS } = options
  const link = ref('')
  const phase = ref<InvitationPhase>('starting')
  const request = shallowRef<ShareRequest | undefined>(undefined)
  const people = shallowRef<readonly SharePerson[]>([])
  const error = ref('')
  const isBusy = ref(false)
  let timer: ReturnType<typeof setTimeout> | undefined

  const stop = (): void => {
    if (timer !== undefined) clearTimeout(timer)
    timer = undefined
  }
  if (getCurrentScope() !== undefined) onScopeDispose(stop)

  async function refresh(): Promise<void> {
    stop()
    try {
      const status = await client.getInvitationStatus(target)
      phase.value = status.state
      request.value = status.requests.at(0)
      people.value = status.people
    } catch (failure) {
      error.value = getErrorMessage(failure)
    }
    if (phase.value === 'waiting') {
      timer = setTimeout(() => {
        refresh().catch(fail)
      }, refreshMs)
    }
  }

  function fail(failure: unknown): void {
    error.value = getErrorMessage(failure)
  }

  async function start(): Promise<void> {
    try {
      const started = await client.startShareInvitation(target)
      link.value = started.link
      phase.value = 'waiting'
    } catch (failure) {
      fail(failure)
      return
    }
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

  return reactive({
    target,
    link,
    phase,
    request,
    people,
    error,
    isBusy,
    start,
    fail,
    approve: (): Promise<void> =>
      answer(async () => {
        if (request.value !== undefined) await client.approveShareRequest(request.value.requestId)
      }),
    decline: (): Promise<void> =>
      answer(async () => {
        if (request.value !== undefined) await client.declineShareRequest(request.value.requestId)
      }),
    copyLink: (): Promise<void> => globalThis.navigator.clipboard.writeText(link.value),
    stop,
    close,
  })
}

export type InvitationView = ReturnType<typeof createInvitationView>
