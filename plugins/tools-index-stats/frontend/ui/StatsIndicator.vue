<script setup lang="ts">
/**
 * The title bar item that counts the vectors of the search index, and the popover with its details.
 */
import { computed, ref, useTemplateRef, watch } from 'vue'
import { useMenuDismissal } from '@numen/ui'
import type { Stats } from '../api/stats'
import type { Phase } from '../model/stats'
import { formatSummary, formatVectors, MARK } from '../lib/format'
import StatsPopover from './StatsPopover.vue'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  stats: Stats | null
  phase: Phase
  locale?: string | undefined
}>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const root = useTemplateRef<HTMLElement>('root')
// The popover is placed against the item by CSS and follows it through a resize.
const dismissal = useMenuDismissal(root, onClose, isResize)

const figure = computed(() => {
  return props.stats ? formatVectors(props.stats, props.locale) : words.unavailable
})
const summary = computed(() => {
  if (props.stats) return formatSummary(props.stats, props.locale)
  return props.phase === 'loading' ? words.loadingHint : words.unavailableHint
})

/* --------------------------------- Hooks ---------------------------------- */
watch(isOpen, (open) => (open ? dismissal.listen() : dismissal.release()))

/* -------------------------------- Handlers -------------------------------- */
function onToggle() {
  isOpen.value = !isOpen.value
}

function onClose() {
  isOpen.value = false
}

function isResize(event: Event): boolean {
  return event.type === 'resize'
}
</script>

<template>
  <div ref="root" class="index-stats">
    <button
      type="button"
      class="index-stats__trigger"
      tabindex="-1"
      :aria-expanded="isOpen"
      aria-haspopup="dialog"
      :title="summary"
      :aria-label="summary"
      @click="onToggle"
    >
      <span class="index-stats__icon" aria-hidden="true">{{ MARK }}</span>
      <span class="index-stats__figure">{{ figure }}</span>
    </button>

    <StatsPopover v-if="isOpen && stats" :stats="stats" :locale="locale" @close="onClose" />
  </div>
</template>
