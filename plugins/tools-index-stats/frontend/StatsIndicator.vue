<script setup lang="ts">
/**
 * The status bar item that counts the vectors of the search index, and the popover with its details.
 */
import { computed, ref, useTemplateRef, watch } from 'vue'
import { useMenuDismissal } from '@numen/ui'
import type { Stats } from './api/stats'
import type { Phase } from './model/stats'
import { formatSummary, formatVectors, isPending, MARK } from './lib/format'
import StatsPopover from './StatsPopover.vue'
import { WORDS as words } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  stats: Stats | null
  phase: Phase
  /** The locale numbers and times are written in; the locale of the window when empty. */
  locale?: string | undefined
}>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const root = useTemplateRef<HTMLElement>('root')
const dismissal = useMenuDismissal(root, onClose)

const figure = computed(() => {
  if (props.stats) return formatVectors(props.stats, props.locale)
  return props.phase === 'loading' ? words.loading : words.unavailable
})
const isIndexing = computed(() => props.stats !== null && isPending(props.stats))
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
</script>

<template>
  <div ref="root" class="index-stats">
    <button
      type="button"
      class="index-stats__trigger"
      :aria-expanded="isOpen"
      aria-haspopup="dialog"
      :title="summary"
      :aria-label="summary"
      @click="onToggle"
    >
      <span class="index-stats__label">
        <span
          class="index-stats__icon"
          :class="{ 'index-stats__icon--indexing': isIndexing }"
          aria-hidden="true"
        >
          {{ MARK }}
        </span>
        {{ figure }}
      </span>
    </button>

    <StatsPopover v-if="isOpen && stats" :stats="stats" :locale="locale" @close="onClose" />
  </div>
</template>
