import { getCurrentScope, onScopeDispose, reactive, ref, shallowRef } from 'vue'
import type {
  AcceptInvitationClient,
  InvitationPreview,
  RequestPhase,
  RequestStatus,
} from '../api/invitations'
import { getErrorMessage } from './vaults'
import { createPoller } from './polling'

const REFRESH_MS = 1000

export interface AcceptanceViewOptions {
  readonly client: AcceptInvitationClient
  readonly close: () => void
  readonly refreshMs?: number
}

export type AcceptPhase = 'link' | RequestPhase

const ENDED: readonly RequestPhase[] = ['received', 'declined', 'used', 'expired', 'failed']

const isEnded = (phase: AcceptPhase): boolean => phase !== 'link' && ENDED.includes(phase)

/** What the dialog "Accept a shared folder" holds: the link pasted, what it offers and how the request stands. */
export function createAcceptanceView(options: AcceptanceViewOptions) {
  const { client, close, refreshMs = REFRESH_MS } = options
  const text = ref('')
  const preview = shallowRef<InvitationPreview | undefined>(undefined)
  const previewError = ref('')
  const phase = ref<AcceptPhase>('link')
  const code = ref('')
  const reason = ref('')
  const error = ref('')
  const poller = createPoller(refreshMs, fail)
  let seq = 0

  if (getCurrentScope() !== undefined) onScopeDispose(poller.stop)

  function fail(failure: unknown): void {
    error.value = getErrorMessage(failure)
  }

  async function setLink(value: string): Promise<void> {
    text.value = value
    const mine = ++seq
    preview.value = undefined
    previewError.value = ''
    if (value.trim() === '') return
    try {
      const found = await client.readSharePreview(value.trim())
      if (mine === seq) preview.value = found
    } catch (failure) {
      if (mine === seq) previewError.value = getErrorMessage(failure)
    }
  }

  function take(status: RequestStatus): void {
    phase.value = status.phase
    code.value = status.code ?? ''
    reason.value = status.error ?? ''
  }

  async function follow(requestId: string): Promise<void> {
    try {
      take(await client.getShareInvitationStatus(requestId))
    } catch (failure) {
      fail(failure)
    }
    if (!isEnded(phase.value)) poller.runAgain(() => follow(requestId))
  }

  async function accept(): Promise<void> {
    error.value = ''
    phase.value = 'connecting'
    try {
      const sent = await client.acceptShareInvitation({ link: text.value.trim(), deviceName: '' })
      await follow(sent.requestId)
    } catch (failure) {
      fail(failure)
      phase.value = 'link'
    }
  }

  return reactive({
    text,
    preview,
    previewError,
    phase,
    code,
    reason,
    error,
    setLink,
    accept,
    fail,
    stop: poller.stop,
    close,
  })
}

export type AcceptanceView = ReturnType<typeof createAcceptanceView>
