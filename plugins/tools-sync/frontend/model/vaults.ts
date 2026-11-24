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
import { WORDS as words } from '../words'

const MS_PER_MINUTE = 60_000
const REFRESH_MS = 15_000

type RowTone = 'ok' | 'warning' | 'alarm'

export interface VaultRow {
  readonly id: string
  readonly name: string
  readonly status: string
  readonly tone: RowTone
  readonly detail: string
  readonly canResolve: boolean
}

interface UseVaultsOptions {
  readonly client?: VaultsClient | undefined
  readonly now: () => Date
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
  isServerAway(error) ? '' : error instanceof Error ? error.message : String(error)

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

const getRow = (vault: Vault, now: Date, isReachable: boolean): VaultRow => {
  const base = { id: vault.id, name: vault.name }
  const paused = getPausedLabel(vault.pausedDeletions)
  if (paused !== '') {
    return { ...base, status: paused, tone: 'warning', detail: '', canResolve: true }
  }
  const synced = getSyncedLabel(vault, now)
  if (vault.conflictCount > 0) {
    const detail = words.conflicts(vault.conflictCount)
    return { ...base, status: words.statusProblem, tone: 'alarm', detail, canResolve: false }
  }
  if (!isReachable) {
    const status = words.statusWaiting
    return { ...base, status, tone: 'warning', detail: synced, canResolve: false }
  }
  return { ...base, status: words.statusUpToDate, tone: 'ok', detail: synced, canResolve: false }
}

export function useVaults(options: UseVaultsOptions) {
  const { client = vaultsClient, now } = options
  const vaults = shallowRef<readonly Vault[]>([])
  const remote = shallowRef<readonly RemoteVault[]>([])
  const status = ref<VaultsStatus>('loading')
  const error = ref('')
  const hasIdentity = ref(true)
  const isReachable = ref(true)
  const pendingConfirm = shallowRef<PendingConfirm | null>(null)

  const rows = computed(() =>
    vaults.value
      .filter((vault) => vault.state === 'connected')
      .map((vault) => getRow(vault, now(), isReachable.value)),
  )
  const freeVaults = computed(() => vaults.value.filter((vault) => vault.state === 'notConnected'))
  const cloudVaults = computed(() => remote.value.filter((one) => !one.isConnected))
  const canGetFromCloud = computed(
    () => hasIdentity.value && isReachable.value && cloudVaults.value.length > 0,
  )
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
      freeVaults,
      cloudVaults,
      canGetFromCloud,
      pendingConfirm,
    },
    actions: {
      load,
      watch,
      connectExisting,
      resolveDeletions,
      confirmConnect,
      cancelConnect,
    },
  }
}
