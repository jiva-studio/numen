<script setup lang="ts">
/**
 * The title bar item of Numen Sync: the badge with the state of all vaults, and the popover that lists
 * every synced vault with its state and, while it syncs, its progress.
 */
import { ref, useTemplateRef, watch } from 'vue'
import { isPageMove, useMenuDismissal } from '@numen/ui'
import type { SyncState } from '../model/sync'
import { WORDS as words } from '../words'
import SyncBadge from './SyncBadge.vue'
import VaultRow from './VaultRow.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: SyncState }>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const root = useTemplateRef<HTMLElement>('root')
const dismissal = useMenuDismissal(root, onClose, isPageMove)

/* --------------------------------- Hooks ---------------------------------- */
watch(isOpen, (open) => (open ? dismissal.listen() : dismissal.release()))

/* -------------------------------- Handlers -------------------------------- */
function onToggle(): void {
  isOpen.value = !isOpen.value
}

function onClose(): void {
  isOpen.value = false
}
</script>

<template>
  <div v-if="props.state.phase.value !== 'hidden'" ref="root" class="sync-indicator">
    <button
      type="button"
      class="sync-indicator__trigger"
      tabindex="-1"
      :aria-expanded="isOpen"
      aria-haspopup="dialog"
      @click="onToggle"
    >
      <SyncBadge :phase="props.state.phase.value" :percent="props.state.percent.value" />
    </button>
    <div v-if="isOpen" class="sync-indicator__popover" role="dialog" :aria-label="words.vaultsTitle">
      <p v-if="props.state.rows.value.length === 0" class="sync-indicator__empty">
        {{ words.noVaultsTitle }}
      </p>
      <ul v-else class="sync-card__list">
        <VaultRow
          v-for="row in props.state.rows.value"
          :key="row.id"
          :name="row.name"
          :kind="row.kind"
          :status="row.status"
          :qualifier="row.qualifier"
          :facts="row.facts"
          :is-transferring="row.isTransferring"
          :progress="row.progress"
        />
      </ul>
    </div>
  </div>
</template>
