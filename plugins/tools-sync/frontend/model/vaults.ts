import { computed, ref, shallowRef } from 'vue'
import type { RemoteVault, Vault, VaultsClient, VaultState } from '../api/vaults'
import { vaultsClient } from '../api/wire'
import { WORDS as words } from '../words'

const MS_PER_MINUTE = 60_000

export interface VaultRow {
  readonly id: string
  readonly name: string
  readonly path: string
  readonly stateLabel: string
  readonly syncedLabel: string
  readonly conflictLabel: string
  readonly canConnect: boolean
}

export interface UseVaultsOptions {
  readonly client?: VaultsClient | undefined
  readonly now: () => Date
}

export interface PendingConfirm {
  readonly vaultId: string
  readonly remoteId: string
  readonly text: string
}

export type VaultsStatus = 'loading' | 'ready' | 'error'

const STATE_LABELS: Record<VaultState, string> = {
  notConnected: words.stateNotConnected,
  connected: words.stateConnected,
  copy: words.stateCopy,
}

export const getErrorMessage = (error: unknown): string =>
  error instanceof Error ? error.message : String(error)

const getSyncedLabel = (vault: Vault, now: Date): string => {
  if (vault.state !== 'connected' || vault.lastSyncedAt === '') return ''
  const minutes = Math.floor((now.getTime() - Date.parse(vault.lastSyncedAt)) / MS_PER_MINUTE)
  return minutes < 1 ? words.syncedJustNow : words.syncedMinutesAgo(minutes)
}

const getRow = (vault: Vault, now: Date): VaultRow => ({
  id: vault.id,
  name: vault.name,
  path: vault.path,
  stateLabel: STATE_LABELS[vault.state],
  syncedLabel: getSyncedLabel(vault, now),
  conflictLabel: vault.conflictCount > 0 ? words.conflicts(vault.conflictCount) : '',
  canConnect: vault.state === 'notConnected',
})

export function useVaults(options: UseVaultsOptions) {
  const { client = vaultsClient, now } = options
  const vaults = shallowRef<readonly Vault[]>([])
  const remote = shallowRef<readonly RemoteVault[]>([])
  const status = ref<VaultsStatus>('loading')
  const error = ref('')
  const hasIdentity = ref(true)
  const openFor = ref('')
  const pendingConfirm = shallowRef<PendingConfirm | null>(null)

  const rows = computed(() => vaults.value.map((vault) => getRow(vault, now())))
  const warning = computed(() => (hasIdentity.value ? '' : words.noIdentity))

  async function read(): Promise<void> {
    const state = await client.getStatus()
    hasIdentity.value = state.hasIdentity
    vaults.value = state.hasIdentity ? (await client.listVaults()).vaults : []
  }

  async function load(): Promise<void> {
    try {
      await read()
      status.value = 'ready'
    } catch (failure) {
      status.value = 'error'
      error.value = getErrorMessage(failure)
    }
  }

  async function openRemote(vaultId: string): Promise<void> {
    try {
      remote.value = (await client.listRemoteVaults()).vaults
      openFor.value = vaultId
      error.value = ''
    } catch (failure) {
      error.value = getErrorMessage(failure)
    }
  }

  function closeRemote(): void {
    openFor.value = ''
  }

  async function connect(request: Parameters<VaultsClient['connectVault']>[0]): Promise<void> {
    try {
      await client.connectVault(request)
      error.value = ''
      closeRemote()
      await read()
    } catch (failure) {
      error.value = getErrorMessage(failure)
    }
  }

  function confirmConnect(): Promise<void> {
    return Promise.reject(new Error('not implemented'))
  }

  function cancelConnect(): void {
    throw new Error('not implemented')
  }

  const connectNew = (vaultId: string): Promise<void> => connect({ vaultId, mode: 'new' })
  const connectExisting = (vaultId: string, remoteId: string): Promise<void> =>
    connect({ vaultId, mode: 'existing', remoteId })

  return {
    state: { status, error, hasIdentity, warning, rows, openFor, remote, pendingConfirm },
    actions: {
      load,
      openRemote,
      closeRemote,
      connectNew,
      connectExisting,
      confirmConnect,
      cancelConnect,
    },
  }
}
