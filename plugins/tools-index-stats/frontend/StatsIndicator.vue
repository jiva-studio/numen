<script setup lang="ts">
/**
 * The title bar item that counts the vectors of the search index, and the popover with its details.
 */
import { computed, ref } from 'vue'
import { BarChip } from '@numen/ui'
import type { Stats } from './api/stats'
import type { Phase } from './model/stats'
import { formatSummary, formatVectors, MARK } from './lib/format'
import StatsPopover from './StatsPopover.vue'
import { WORDS as words } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  stats: Stats | null
  phase: Phase
  locale?: string | undefined
}>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)

const figure = computed(() => {
  return props.stats ? formatVectors(props.stats, props.locale) : words.unavailable
})
const summary = computed(() => {
  if (props.stats) return formatSummary(props.stats, props.locale)
  return props.phase === 'loading' ? words.loadingHint : words.unavailableHint
})

/* -------------------------------- Handlers -------------------------------- */
function onClose() {
  isOpen.value = false
}
</script>

<template>
  <BarChip v-model:open="isOpen" :label="summary" :tooltip="summary" :popover-label="words.title">
    <span class="index-stats__icon" aria-hidden="true">{{ MARK }}</span>
    <span class="index-stats__figure">{{ figure }}</span>
    <template v-if="stats" #popover>
      <StatsPopover :stats="stats" :locale="locale" @close="onClose" />
    </template>
  </BarChip>
</template>
