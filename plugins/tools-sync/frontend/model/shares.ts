import { computed, getCurrentScope, onScopeDispose, reactive, ref } from 'vue'
import type {
  ReceivedShare,
  ReceivedState,
  Share,
  SharesClient,
  ShareReader,
} from '../api/shares'
import { sharesClient } from '../api/wire'
import { WORDS as words } from '../words'
import { getErrorMessage } from './vaults'

const REFRESH_MS = 2000

export interface ShareTarget {
  readonly vaultId: string
  readonly folder: string
}

export interface ReceivedRow {
  readonly vaultId: string
  readonly name: string
  readonly ownerFingerprint: string
  readonly stateLabel: string
}

const RECEIVED_LABELS: Record<ReceivedState, string> = {
  receiving: words.receiving,
  ready: words.receivedReady,
  ended: words.receivedEnded,
}

export const getReceivedRow = (share: ReceivedShare): ReceivedRow => ({
  vaultId: share.vaultId,
  name: share.name,
  ownerFingerprint: share.ownerFingerprint,
  stateLabel: RECEIVED_LABELS[share.state],
})

/** Whether a refusal is the warning the person must confirm before the reader is added. */
export const isIdentityWarning = (message: string): boolean => message === words.otherIdentity

/** A timer that runs one task again after a pause and is stopped with the scope that made it. */
function createRepeater(delayMs: number) {
  let timer: ReturnType<typeof setTimeout> | undefined
  const stop = (): void => {
    if (timer !== undefined) clearTimeout(timer)
    timer = undefined
  }
  const again = (task: () => Promise<void>): void => {
    stop()
    timer = setTimeout(() => void task(), delayMs)
  }
  if (getCurrentScope() !== undefined) onScopeDispose(stop)
  return { stop, again }
}

/** The text of a field that is read by the backend on every change; an answer of an older text is dropped. */
function createReadField<Info>(read: (text: string) => Promise<Info>) {
  const text = ref('')
  const info = ref<Info | undefined>(undefined)
  const error = ref('')
  let seq = 0

  async function set(value: string): Promise<void> {
    text.value = value
    const mine = ++seq
    info.value = undefined
    error.value = ''
    if (value.trim() === '') return
    try {
      const result = await read(value.trim())
      if (mine === seq) info.value = result
    } catch (failure) {
      if (mine === seq) error.value = getErrorMessage(failure)
    }
  }

  function clear(): void {
    seq++
    text.value = ''
    info.value = undefined
    error.value = ''
  }

  return { text, info, error, set, clear }
}

export interface ShareViewOptions {
  readonly client?: SharesClient | undefined
  readonly target: ShareTarget
  readonly close: () => void
  readonly refreshMs?: number
}

const EMPTY_SHARE: Share = { state: 'notShared', readers: [] }

/** What the dialog "Share..." of a folder holds: the share, the code being read and the invitation given. */
export function createShareView(options: ShareViewOptions) {
  const { client = sharesClient, target, close, refreshMs = REFRESH_MS } = options
  const share = ref<Share>(EMPTY_SHARE)
  const label = ref('')
  const invitation = ref('')
  const error = ref('')
  const isBusy = ref(false)
  const field = createReadField((text) => client.readReaderCode(text))
  const repeater = createRepeater(refreshMs)

  async function load(): Promise<void> {
    repeater.stop()
    try {
      share.value = await client.getShare(target)
      if (share.value.error !== undefined) error.value = share.value.error
    } catch (failure) {
      error.value = getErrorMessage(failure)
    }
    if (share.value.state === 'creating') repeater.again(load)
  }

  async function setCode(text: string): Promise<void> {
    await field.set(text)
    if (label.value === '') label.value = field.info.value?.label ?? ''
  }

  async function guard(work: () => Promise<void>): Promise<void> {
    isBusy.value = true
    error.value = ''
    try {
      await work()
    } catch (failure) {
      error.value = getErrorMessage(failure)
    } finally {
      isBusy.value = false
    }
  }

  const add = (confirmChange: boolean): Promise<void> =>
    guard(async () => {
      const code = field.text.value.trim()
      const added = await client.addReader({ ...target, code, label: label.value.trim(), confirmChange })
      invitation.value = added.invitation
      field.clear()
      label.value = ''
      await load()
    })

  const removeReader = (reader: ShareReader): Promise<void> =>
    guard(async () => {
      await client.removeReader({ ...target, fingerprint: reader.fingerprint })
      await load()
    })

  const stopShare = (): Promise<void> =>
    guard(async () => {
      await client.stopShare(target)
      invitation.value = ''
      await load()
    })

  return reactive({
    target,
    share,
    code: field.text,
    fingerprint: computed(() => field.info.value?.fingerprint ?? ''),
    codeError: field.error,
    label,
    invitation,
    error,
    isBusy,
    load,
    setCode,
    setLabel: (value: string): void => {
      label.value = value
    },
    fail: (failure: unknown): void => {
      error.value = getErrorMessage(failure)
    },
    add,
    removeReader,
    stopShare,
    stop: repeater.stop,
    close,
  })
}

export type ShareView = ReturnType<typeof createShareView>

export interface ReceivedViewOptions {
  readonly client?: SharesClient | undefined
  readonly refreshMs?: number
}

/** What the page "Shares" holds: the code of this account, the invitation being received and the shares received. */
export function createReceivedView(options: ReceivedViewOptions = {}) {
  const { client = sharesClient, refreshMs = REFRESH_MS } = options
  const code = ref('')
  const fingerprint = ref('')
  const error = ref('')
  const rows = ref<readonly ReceivedRow[]>([])
  const isBusy = ref(false)
  const invitation = createReadField((text) => client.readInvitation(text))
  const repeater = createRepeater(refreshMs)

  async function refresh(): Promise<void> {
    repeater.stop()
    try {
      const shares = await client.listReceived()
      rows.value = shares.map(getReceivedRow)
      if (shares.some((one) => one.state === 'receiving')) repeater.again(refresh)
    } catch (failure) {
      error.value = getErrorMessage(failure)
    }
  }

  async function load(): Promise<void> {
    try {
      const own = await client.getReaderCode()
      code.value = own.code
      fingerprint.value = own.fingerprint
    } catch (failure) {
      error.value = getErrorMessage(failure)
    }
    await refresh()
  }

  async function receive(): Promise<void> {
    isBusy.value = true
    error.value = ''
    try {
      await client.acceptInvitation(invitation.text.value.trim())
      invitation.clear()
      await refresh()
    } catch (failure) {
      error.value = getErrorMessage(failure)
    } finally {
      isBusy.value = false
    }
  }

  return reactive({
    code,
    fingerprint,
    text: invitation.text,
    shareName: computed(() => invitation.info.value?.shareName ?? ''),
    ownerFingerprint: computed(() => invitation.info.value?.ownerFingerprint ?? ''),
    invitationError: invitation.error,
    error,
    rows,
    isBusy,
    load,
    setText: invitation.set,
    fail: (failure: unknown): void => {
      error.value = getErrorMessage(failure)
    },
    receive,
    stop: repeater.stop,
  })
}

export type ReceivedView = ReturnType<typeof createReceivedView>
