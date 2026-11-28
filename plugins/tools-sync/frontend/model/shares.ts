import { getCurrentScope, onScopeDispose, reactive, ref } from 'vue'
import type {
  FolderSharesClient,
  ReceivedShare,
  ReceivedSharesClient,
  ReceivedState,
  SharedFolder,
} from '../api/shares'
import { folderSharesClient, shareInboxClient } from '../api/wire'
import { WORDS as words } from '../words'
import { getShortFingerprint } from '../lib/fingerprint'
import { createPoller } from './polling'
import { getErrorMessage } from './vaults'

const REFRESH_MS = 2000

export interface ReceivedRow {
  readonly vaultId: string
  readonly name: string
  readonly ownerShort: string
  readonly stateLabel: string
  readonly tone: 'ok' | 'warning' | 'alarm'
}

export interface DeviceRow {
  readonly fingerprint: string
  readonly name: string
}

export interface GivenRow {
  readonly vaultId: string
  readonly folder: string
  readonly isCreating: boolean
  readonly devices: readonly DeviceRow[]
}

const RECEIVED_LABELS: Record<ReceivedState, string> = {
  receiving: words.receiving,
  ready: words.receivedReady,
  ended: words.receivedEnded,
}

const RECEIVED_TONES: Record<ReceivedState, ReceivedRow['tone']> = {
  receiving: 'warning',
  ready: 'ok',
  ended: 'alarm',
}

const getReceivedRow = (share: ReceivedShare): ReceivedRow => ({
  vaultId: share.vaultId,
  name: share.name,
  ownerShort: getShortFingerprint(share.ownerFingerprint),
  stateLabel: RECEIVED_LABELS[share.state],
  tone: RECEIVED_TONES[share.state],
})

const getGivenRow = (share: SharedFolder): GivenRow => ({
  vaultId: share.vaultId,
  folder: share.folder,
  isCreating: share.state === 'creating',
  devices: share.readers
    .filter((one) => one.state === 'active')
    .map((one) => ({
      fingerprint: one.fingerprint,
      name: words.deviceLabel(one.label, getShortFingerprint(one.fingerprint)),
    })),
})

export interface SharesViewOptions {
  readonly given?: FolderSharesClient | undefined
  readonly received?: ReceivedSharesClient | undefined
  readonly refreshMs?: number
}

/** What the page "Shared folders" holds: the folders this identity shares with their devices, and the folders shared with it. */
export function createSharesView(options: SharesViewOptions = {}) {
  const { given = folderSharesClient, received = shareInboxClient } = options
  const gives = ref<readonly GivenRow[]>([])
  const receives = ref<readonly ReceivedRow[]>([])
  const error = ref('')
  const pendingLeave = ref<ReceivedRow | null>(null)
  const poller = createPoller(options.refreshMs ?? REFRESH_MS, fail)

  if (getCurrentScope() !== undefined) onScopeDispose(poller.stop)

  function fail(failure: unknown): void {
    error.value = getErrorMessage(failure)
  }

  async function load(): Promise<void> {
    try {
      const [shared, taken] = await Promise.all([given.listShares(), received.listReceived()])
      gives.value = shared.map(getGivenRow)
      receives.value = taken.map(getReceivedRow)
      error.value = ''
      const isBusy =
        shared.some((one) => one.state === 'creating') ||
        taken.some((one) => one.state === 'receiving')
      if (isBusy) poller.runAgain(load)
    } catch (failure) {
      fail(failure)
    }
  }

  async function change(work: () => Promise<void>): Promise<void> {
    try {
      await work()
      await load()
    } catch (failure) {
      fail(failure)
    }
  }

  return reactive({
    gives,
    receives,
    error,
    load,
    fail,
    removeAccess: (row: GivenRow, device: DeviceRow): Promise<void> =>
      change(() =>
        given.removeReader({
          vaultId: row.vaultId,
          folder: row.folder,
          fingerprint: device.fingerprint,
        }),
      ),
    stopSharing: (row: GivenRow): Promise<void> =>
      change(() => given.stopShare({ vaultId: row.vaultId, folder: row.folder })),
    pendingLeave,
    askLeave: (row: ReceivedRow): void => {
      pendingLeave.value = row
    },
    cancelLeave: (): void => {
      pendingLeave.value = null
    },
    confirmLeave: (): Promise<void> => {
      const row = pendingLeave.value
      pendingLeave.value = null
      if (row === null) return Promise.resolve()
      return change(() => received.leaveShare({ vaultId: row.vaultId }))
    },
    stop: poller.stop,
  })
}

export type SharesView = ReturnType<typeof createSharesView>
