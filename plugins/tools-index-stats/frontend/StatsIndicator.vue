<script setup lang="ts">
/**
 * The status bar item that counts the vectors of the search index, and the popover with its details.
 */
import { computed, ref, useTemplateRef, watch } from 'vue'
import { useMenuDismissal } from '@numen/ui'
import type { Stats } from './api/stats'
import type { Phase } from './model/stats'
import { formatLabel, MARK } from './lib/format'
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

const label = computed(() =>
  props.stats ? formatLabel(props.stats, props.locale) : `${MARK} ${words.unavailable}`,
)
const isShown = computed(() => props.phase === 'unavailable' || props.stats !== null)

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
  <div v-if="isShown" ref="root" class="index-stats">
    <button
      type="button"
      class="index-stats__trigger"
      :aria-expanded="isOpen"
      aria-haspopup="dialog"
      :title="words.triggerHint"
      @click="onToggle"
    >
      <span class="index-stats__label">{{ label }}</span>
    </button>

    <StatsPopover v-if="isOpen && stats" :stats="stats" :locale="locale" @close="onClose" />
  </div>
</template>
