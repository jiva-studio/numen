<script setup lang="ts">
import { computed } from 'vue'
import type { SyncPhase } from '../model/sync'
import { RefreshCw } from '@lucide/vue'
import { WORDS as words } from '../words'
import SyncRing from './SyncRing.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  phase: SyncPhase
  percent?: number | null
}>()

/* --------------------------------- State ---------------------------------- */
const label = computed(() => {
  if (props.phase === 'syncing') {
    return props.percent === undefined || props.percent === null
      ? words.syncingLabel
      : words.syncingPercent(props.percent)
  }
  return props.phase === 'waiting' ? words.statusWaiting : words.syncedLabel
})
const progress = computed(() =>
  props.percent === undefined || props.percent === null ? null : props.percent / 100,
)
const detail = computed(() => {
  if (props.phase === 'syncing') return words.syncingDetail
  return props.phase === 'waiting' ? words.statusWaiting : words.syncedDetail
})
</script>

<template>
  <span v-if="phase !== 'hidden'" class="sync-badge" role="status" :aria-label="detail">
    <span v-if="phase === 'syncing'" class="sync-badge__ring" aria-hidden="true">
      <SyncRing :progress="progress">
        <RefreshCw class="sync-ring__glyph" />
      </SyncRing>
    </span>
    <span
      v-else
      class="sync-badge__dot"
      :class="{ 'sync-badge__dot--waiting': phase === 'waiting' }"
      aria-hidden="true"
    />
    <span class="sync-badge__label">{{ label }}</span>
  </span>
</template>
