import { formatErrorMessage } from '@numen/plugin-sdk'
import { computed, ref, shallowRef } from 'vue'
import type {
  DeletionsChoice,
  PausedDeletions,
  RemoteVault,
  Vault,
  VaultsClient,
} from '../api/vaults'
import { getFailureCode } from '../api/subscription'
import { vaultsClient } from '../api/wire'
import { formatCloudSize } from '../lib/cloudSize'
import { WORDS as words } from '../words'

const MS_PER_MINUTE = 60_000
const REFRESH_MS = 15_000
const MAX_SKELETON_ROWS = 5

const knownRowCount = ref(1)

type RowTone = 'ok' | 'warning' | 'alarm'

export type VaultRowKind = 'ok' | 'syncing' | 'off' | 'conflict'

export interface VaultRow {
  readonly id: string
  readonly name: string
  readonly status: string
  readonly tone: RowTone
  readonly kind: VaultRowKind
  readonly detail: string
  readonly canResolve: boolean
  readonly remoteId: string
  readonly isSynced: boolean
  readonly size: string
  readonly isTransferring: boolean
  readonly progress: number | null
}

interface UseVaultsOptions {
  readonly client?: VaultsClient | undefined
  readonly now: () => Date
  readonly onConnected?: (() => void) | undefined
}

export interface PendingStop {
  readonly vaultId: string
  readonly name: string
}

export interface PendingDelete {
  readonly remoteId: string
  readonly name: string
}

export interface PendingConfirm {
  readonly vaultId: string
  readonly remoteId: string
  readonly text: string
}

type VaultsStatus = 'loading' | 'ready' | 'error'

const isServerAway = (failure: unknown): boolean => getFailureCode(failure) === 'unavailable'

/** The text of a failure; empty for an unreachable server, which the subscription card says. */
export const getErrorMessage = (error: unknown): string =>
  isServerAway(error) ? '' : formatErrorMessage(error)

const getSyncedLabel = (vault: Vault, now: Date): string => {
  if (vault.lastSyncedAt === '') return ''
  const minutes = Math.floor((now.getTime() - Date.parse(vault.lastSyncedAt)) / MS_PER_MINUTE)
  return minutes < 1 ? words.syncedJustNow : words.syncedMinutesAgo(minutes)
}

const getPausedLabel = (paused: PausedDeletions | null | undefined): string => {
  if (paused === null || paused === undefined) return ''
  return paused.direction === 'apply'
    ? words.deletionsPausedApply(paused.count, paused.tracked)
    : words.deletionsPausedSend(paused.count, paused.tracked)
}

const getSize = (cloud: readonly RemoteVault[], remoteId: string): string => {
  const one = cloud.find((vault) => vault.remoteId === remoteId)
  return one === undefined ? '' : words.sizeInCloud(formatCloudSize(one.sizeBytes))
}

const getProgress = (synced: number, total: number): number =>
  total > 0 ? Math.min(Math.max(synced / total, 0), 1) : 0

/** The share of bytes done over the vaults that sync now, in whole percents. */
export function getSyncPercent(vaults: readonly Vault[]): number {
  const active = vaults.filter((vault) => vault.syncing === true)
  const total = active.reduce((sum, vault) => sum + (vault.totalBytes ?? 0), 0)
  const synced = active.reduce((sum, vault) => sum + (vault.syncedBytes ?? 0), 0)
  return Math.round(getProgress(synced, total) * 100)
}

const getCloudRow = (one: RemoteVault): VaultRow => ({
  id: one.remoteId,
  name: one.name,
  status: words.notSyncing,
  tone: 'warning',
  kind: 'off',
  detail: '',
  canResolve: false,
  remoteId: one.remoteId,
  isSynced: false,
  size: words.sizeInCloud(formatCloudSize(one.sizeBytes)),
  isTransferring: false,
  progress: null,
})

export const getRow = (
  vault: Vault,
  now: Date,
  isReachable: boolean,
  cloud: readonly RemoteVault[],
): VaultRow => {
  const base = {
    id: vault.id,
    name: vault.name,
    remoteId: vault.remoteId,
    isSynced: true,
    size: getSize(cloud, vault.remoteId),
    isTransferring: false,
    progress: null,
  }
  const paused = getPausedLabel(vault.pausedDeletions)
  if (paused !== '') {
    return { ...base, status: paused, tone: 'warning', kind: 'syncing', detail: '', canResolve: true }
  }
  const synced = getSyncedLabel(vault, now)
  if (vault.conflictCount > 0) {
    const detail = words.conflicts(vault.conflictCount)
    const status = words.statusProblem
    return { ...base, status, tone: 'alarm', kind: 'conflict', detail, canResolve: false }
  }
  if (!isReachable) {
    const status = words.statusWaiting
    return { ...base, status, tone: 'warning', kind: 'syncing', detail: synced, canResolve: false }
  }
  if (vault.syncing === true) {
    const synced = vault.syncedBytes ?? 0
    const total = vault.totalBytes ?? 0
    const size = words.syncProgress(formatCloudSize(synced), formatCloudSize(total))
    const progress = total > 0 ? getProgress(synced, total) : null
    const status = words.statusSyncing
    return { ...base, status, tone: 'warning', kind: 'syncing', detail: '', canResolve: false, size, isTransferring: true, progress }
  }
  const status = words.statusUpToDate
  return { ...base, status, tone: 'ok', kind: 'ok', detail: synced, canResolve: false }
}

export function useVaults(options: UseVaultsOptions) {
  const { client = vaultsClient, now, onConnected } = options
  const vaults = shallowRef<readonly Vault[]>([])
  const remote = shallowRef<readonly RemoteVault[]>([])
  const status = ref<VaultsStatus>('loading')
  const error = ref('')
  const hasIdentity = ref(true)
  const isReachable = ref(true)
  const pendingConfirm = shallowRef<PendingConfirm | null>(null)
  const pendingStop = shallowRef<PendingStop | null>(null)
  const pendingDelete = shallowRef<PendingDelete | null>(null)

  const cloudVaults = computed(() => remote.value.filter((one) => !one.isConnected))
  const rows = computed(() => [
    ...vaults.value
      .filter((vault) => vault.state === 'connected')
      .map((vault) => getRow(vault, now(), isReachable.value, remote.value)),
    ...cloudVaults.value.map(getCloudRow),
  ])
  const freeVaults = computed(() => vaults.value.filter((vault) => vault.state === 'notConnected'))
  const canGetFromCloud = computed(
    () => hasIdentity.value && isReachable.value && cloudVaults.value.length > 0,
  )
  const skeletonRows = computed(() => knownRowCount.value)
  const warning = computed(() => (hasIdentity.value ? '' : words.noIdentity))

  async function readCloud(): Promise<readonly RemoteVault[]> {
    try {
      return (await client.listRemoteVaults()).vaults
    } catch {
      return []
    }
  }

  async function read(): Promise<void> {
    const state = await client.getStatus()
    hasIdentity.value = state.hasIdentity
    isReachable.value = state.isReachable
    vaults.value = state.hasIdentity ? (await client.listVaults()).vaults : []
    remote.value = state.hasIdentity && state.isReachable ? await readCloud() : []
  }

  async function load(): Promise<void> {
    try {
      await read()
      knownRowCount.value = Math.min(Math.max(rows.value.length, 1), MAX_SKELETON_ROWS)
      error.value = ''
      status.value = 'ready'
    } catch (failure) {
      status.value = 'error'
      if (isServerAway(failure)) isReachable.value = false
      error.value = getErrorMessage(failure)
    }
  }

  function watch(): () => void {
    const refresh = (): void => {
      load().catch(() => undefined)
    }
    refresh()
    const timer = setInterval(refresh, REFRESH_MS)
    return () => clearInterval(timer)
  }

  async function connect(request: Parameters<VaultsClient['connectVault']>[0]): Promise<void> {
    try {
      await client.connectVault(request)
      error.value = ''
      pendingConfirm.value = null
      await read()
      onConnected?.()
    } catch (failure) {
      if (isServerAway(failure)) isReachable.value = false
      error.value = getErrorMessage(failure)
    }
  }

  async function resolveDeletions(vaultId: string, choice: DeletionsChoice): Promise<void> {
    try {
      await client.resolveDeletions({ vaultId, choice })
      error.value = ''
      await read()
    } catch (failure) {
      error.value = getErrorMessage(failure)
    }
  }

  function askStop(row: VaultRow): void {
    pendingStop.value = { vaultId: row.id, name: row.name }
  }

  function cancelStop(): void {
    pendingStop.value = null
  }

  async function confirmStop(deleteCloudCopy: boolean): Promise<void> {
    const pending = pendingStop.value
    if (pending === null) return
    pendingStop.value = null
    try {
      await client.stopSyncingVault({ vaultId: pending.vaultId, deleteCloudCopy })
      error.value = ''
      await read()
    } catch (failure) {
      error.value = getErrorMessage(failure)
    }
  }

  function askDelete(row: VaultRow): void {
    pendingDelete.value = { remoteId: row.remoteId, name: row.name }
  }

  function cancelDelete(): void {
    pendingDelete.value = null
  }

  async function confirmDelete(): Promise<void> {
    const pending = pendingDelete.value
    if (pending === null) return
    pendingDelete.value = null
    try {
      await client.deleteCloudVault({ remoteId: pending.remoteId })
      error.value = ''
      await read()
    } catch (failure) {
      error.value = getErrorMessage(failure)
    }
  }

  async function confirmConnect(): Promise<void> {
    const pending = pendingConfirm.value
    if (pending === null) return
    pendingConfirm.value = null
    await connect({ vaultId: pending.vaultId, mode: 'existing', remoteId: pending.remoteId })
  }

  function cancelConnect(): void {
    pendingConfirm.value = null
  }

  async function connectExisting(vaultId: string, remoteId: string): Promise<void> {
    try {
      const { differCount } = await client.getConnectPreview({ vaultId, remoteId })
      error.value = ''
      if (differCount === 0) {
        await connect({ vaultId, mode: 'existing', remoteId })
        return
      }
      pendingConfirm.value = { vaultId, remoteId, text: words.notesDiffer(differCount) }
    } catch (failure) {
      if (isServerAway(failure)) isReachable.value = false
      error.value = getErrorMessage(failure)
    }
  }

  return {
    state: {
      status,
      error,
      isReachable,
      warning,
      rows,
      skeletonRows,
      freeVaults,
      cloudVaults,
      canGetFromCloud,
      pendingConfirm,
      pendingStop,
      pendingDelete,
    },
    actions: {
      load,
      watch,
      connectExisting,
      resolveDeletions,
      confirmConnect,
      cancelConnect,
      askStop,
      cancelStop,
      confirmStop,
      askDelete,
      cancelDelete,
      confirmDelete,
    },
  }
}
