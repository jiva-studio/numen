<script setup lang="ts">
/**
 * The title bar item of Numen Sync: the badge with the state of all vaults, and the popover that lists
 * every synced vault with its state and, while it syncs, its progress.
 */
import { ref, useTemplateRef, watch } from 'vue'
import { Tooltip, isPageMove, useMenuDismissal, useTooltip } from '@numen/ui'
import type { SyncState } from '../model/sync'
import { WORDS as words } from '../words'
import SyncBadge from './SyncBadge.vue'
import VaultRow from './VaultRow.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: SyncState }>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const tip = useTooltip()
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
    <div
      v-if="isOpen"
      class="sync-indicator__popover"
      role="dialog"
      :aria-label="words.vaultsTitle"
    >
      <div class="sync-indicator__header">
        <span class="sync-indicator__title">{{ words.vaultsTitle }}</span>
        <button
          type="button"
          class="sync-indicator__close-btn"
          :aria-label="words.close"
          @mouseenter="tip.open($event, words.close)"
          @mouseleave="tip.close"
          @focus="tip.open($event, words.close)"
          @blur="tip.close"
          @click="onClose"
        >
          ✕
        </button>
      </div>
      <div class="sync-indicator__body">
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
      <Teleport to="body">
        <Tooltip v-if="tip.at.value && tip.text.value" :id="tip.id" :at="tip.at.value">
          {{ tip.text.value }}
        </Tooltip>
      </Teleport>
    </div>
  </div>
</template>
