<script setup lang="ts">
/**
 * The card "Synced vaults" of the plugin's settings: the vaults of this device that sync, each with a calm
 * status in words. It only reads: a vault gets in the list from the file tree.
 */
import { computed, onMounted, onUnmounted } from 'vue'
import { Cloud, TriangleAlert } from '@lucide/vue'
import { Button, getCurrentDate, IslandCard, type MenuItem } from '@numen/ui'
import type { VaultsClient } from '../api/vaults'
import { getErrorMessage, useVaults } from '../model/vaults'
import { WORDS as words } from '../words'
import DeleteCloudCopyDialog from './DeleteCloudCopyDialog.vue'
import EndSyncDialog from './EndSyncDialog.vue'
import EmptyBlock from './EmptyBlock.vue'
import RowMenu from './RowMenu.vue'
import VaultRow from './VaultRow.vue'
import VaultRowSkeleton from './VaultRowSkeleton.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  client?: VaultsClient
  now?: () => Date
}>()

/* --------------------------------- State ---------------------------------- */
const model = useVaults({
  client: props.client,
  now: props.now ?? getCurrentDate,
})
const { status, error, warning, rows, skeletonRows, pendingStop, pendingDelete } = model.state
const {
  load,
  watch,
  resolveDeletions,
  askStop,
  cancelStop,
  confirmStop,
  askDelete,
  cancelDelete,
  confirmDelete,
} = model.actions
const STOP_ITEMS: readonly MenuItem[] = [{ id: 'stop', text: words.stopSyncing, group: 'danger' }]
const DELETE_ITEMS: readonly MenuItem[] = [
  { id: 'delete', text: words.deleteCloudItem, group: 'danger' },
]
const hasProblem = computed(() => error.value !== '' && status.value !== 'error')

/* --------------------------------- Hooks ---------------------------------- */
let stopWatching: (() => void) | undefined

onMounted(() => {
  stopWatching = watch()
})

onUnmounted(() => {
  stopWatching?.()
})

/* -------------------------------- Handlers -------------------------------- */
function onResolve(vaultId: string, choice: 'remove' | 'keep'): void {
  resolveDeletions(vaultId, choice).catch(report)
}

function onConfirmStop(deleteCloudCopy: boolean): void {
  confirmStop(deleteCloudCopy).catch(report)
}

function onConfirmDelete(): void {
  confirmDelete().catch(report)
}

function onRetry(): void {
  load().catch(report)
}

/* --------------------------------- Helpers -------------------------------- */
function report(failure: unknown): void {
  error.value = getErrorMessage(failure)
}
</script>

<template>
  <IslandCard class="sync-card" :title="words.vaultsTitle" :aria-label="words.vaultsTitle">
    <ul v-if="status === 'loading'" class="sync-card__list" aria-busy="true">
      <VaultRowSkeleton v-for="index in skeletonRows" :key="index" />
    </ul>
    <EmptyBlock
      v-else-if="status === 'error'"
      role="alert"
      :icon="TriangleAlert"
      :title="words.loadFailedTitle"
      :text="error === '' ? words.loadFailed : error"
    >
      <Button variant="ghost" size="small" @click="onRetry">{{ words.tryAgain }}</Button>
    </EmptyBlock>
    <template v-else>
      <p v-if="warning !== ''" class="sync-card__note" role="status">{{ warning }}</p>
      <p v-if="hasProblem" class="sync-card__note sync-card__note--alarm" role="alert">
        {{ error }}
      </p>
      <EmptyBlock
        v-if="rows.length === 0 && warning === ''"
        :icon="Cloud"
        :title="words.noVaultsTitle"
        :text="words.noVaultsText"
      />
      <ul v-else class="sync-card__list">
        <VaultRow
          v-for="row in rows"
          :key="row.id"
          :name="row.name"
          :kind="row.kind"
          :status="row.status"
          :qualifier="row.qualifier"
          :facts="row.facts"
          :is-transferring="row.isTransferring"
          :progress="row.progress"
        >
          <template #actions>
            <template v-if="row.canResolve">
              <Button variant="outline" size="small" @click="onResolve(row.id, 'remove')">
                {{ words.removeFiles }}
              </Button>
              <Button variant="ghost" size="small" @click="onResolve(row.id, 'keep')">
                {{ words.keepFiles }}
              </Button>
            </template>
            <RowMenu
              v-if="row.isSynced"
              :label="words.rowActions(row.name)"
              :items="STOP_ITEMS"
              @choose="askStop(row)"
            />
            <RowMenu
              v-else
              :label="words.rowActions(row.name)"
              :items="DELETE_ITEMS"
              @choose="askDelete(row)"
            />
          </template>
        </VaultRow>
      </ul>
    </template>
    <EndSyncDialog
      v-if="pendingStop !== null"
      kind="stop"
      :name="pendingStop.name"
      @confirm="onConfirmStop"
      @cancel="cancelStop"
    />
    <DeleteCloudCopyDialog
      v-if="pendingDelete !== null"
      :name="pendingDelete.name"
      @confirm="onConfirmDelete"
      @cancel="cancelDelete"
    />
  </IslandCard>
</template>
