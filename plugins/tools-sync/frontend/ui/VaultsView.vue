<script setup lang="ts">
/**
 * The card "Synced vaults" of the plugin's settings: the vaults of this device that sync, each with a calm
 * status in words. A vault gets in the list from the file tree; the header offers only to get a vault of the
 * account that this device does not hold.
 */
import { computed, onMounted, onUnmounted } from 'vue'
import { Cloud, RefreshCw } from '@lucide/vue'
import { Button, ErrorState, getCurrentDate, IslandCard, LoadingIndicator } from '@numen/ui'
import type { VaultsClient } from '../api/vaults'
import { getErrorMessage, useVaults } from '../model/vaults'
import { WORDS as words } from '../words'
import CloudVaultDialog from './CloudVaultDialog.vue'
import EmptyBlock from './EmptyBlock.vue'
import HeaderAction from './HeaderAction.vue'
import SyncRow from './SyncRow.vue'
import SyncStatus from './SyncStatus.vue'

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
const { status, error, warning, rows, freeVaults, cloudVaults, canGetFromCloud } = model.state
const { isCloudOpen, pendingConfirm } = model.state
const { load, watch, openCloud, closeCloud, connectExisting, resolveDeletions } = model.actions
const hasProblem = computed(() => error.value !== '' && status.value !== 'error')
const question = computed(() => pendingConfirm.value?.text ?? '')

/* --------------------------------- Hooks ---------------------------------- */
let stopWatching: (() => void) | undefined

onMounted(() => {
  stopWatching = watch()
})

onUnmounted(() => {
  stopWatching?.()
})

/* -------------------------------- Handlers -------------------------------- */
function onChoose(vaultId: string, remoteId: string): void {
  connectExisting(vaultId, remoteId).catch(report)
}

function onConfirm(): void {
  model.actions.confirmConnect().catch(report)
}

function onResolve(vaultId: string, choice: 'remove' | 'keep'): void {
  resolveDeletions(vaultId, choice).catch(report)
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
    <template v-if="canGetFromCloud" #aside>
      <HeaderAction @click="openCloud">
        <Cloud :size="14" aria-hidden="true" />
        {{ words.getFromCloud }}
      </HeaderAction>
    </template>
    <p class="sync-card__lead">{{ words.vaultsText }}</p>
    <div v-if="status === 'loading'" class="sync-card__state">
      <LoadingIndicator :label="words.loading" />
    </div>
    <div v-else-if="status === 'error'" class="sync-card__state">
      <ErrorState :message="error === '' ? words.loadFailed : error">
        <Button variant="outline" @click="onRetry">
          <RefreshCw :size="14" aria-hidden="true" />
          {{ words.tryAgain }}
        </Button>
      </ErrorState>
    </div>
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
        <SyncRow v-for="row in rows" :key="row.id" :name="row.name">
          <SyncStatus :text="row.status" :tone="row.tone" />
          <span v-if="row.detail !== ''" class="sync-row__detail">{{ row.detail }}</span>
          <template v-if="row.canResolve" #actions>
            <Button variant="outline" size="small" @click="onResolve(row.id, 'remove')">
              {{ words.removeFiles }}
            </Button>
            <Button variant="ghost" size="small" @click="onResolve(row.id, 'keep')">
              {{ words.keepFiles }}
            </Button>
          </template>
        </SyncRow>
      </ul>
    </template>
    <CloudVaultDialog
      v-if="isCloudOpen"
      :cloud-vaults="cloudVaults"
      :free-vaults="freeVaults"
      :question="question"
      @choose="onChoose"
      @confirm="onConfirm"
      @back="model.actions.cancelConnect"
      @cancel="closeCloud"
    />
  </IslandCard>
</template>
