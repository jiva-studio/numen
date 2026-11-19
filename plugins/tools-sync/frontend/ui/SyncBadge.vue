<script setup lang="ts">
import { computed } from 'vue'
import type { SyncPhase } from '../model/sync'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  phase: SyncPhase
}>()

/* --------------------------------- State ---------------------------------- */
const label = computed(() => (props.phase === 'syncing' ? words.syncingLabel : words.syncedLabel))
const detail = computed(() =>
  props.phase === 'syncing' ? words.syncingDetail : words.syncedDetail,
)
</script>

<template>
  <span v-if="phase !== 'hidden'" class="sync-badge" role="status" :aria-label="detail">
    <span v-if="phase === 'syncing'" class="sync-badge__spinner" aria-hidden="true" />
    <span v-else class="sync-badge__dot" aria-hidden="true" />
    <span class="sync-badge__label">{{ label }}</span>
  </span>
</template>
