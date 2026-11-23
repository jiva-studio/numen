<script setup lang="ts">
/**
 * The title bar item that counts the vectors of the search index, and the popover with its details.
 */
import { computed, onBeforeUnmount, ref, useTemplateRef, watch } from 'vue'
import { Tooltip, isPageMove, useMenuDismissal, useTooltip } from '@numen/ui'
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

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'open'): void
  (event: 'close'): void
}>()

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const tip = useTooltip()
const root = useTemplateRef<HTMLElement>('root')
const dismissal = useMenuDismissal(root, onClose, isPageMove)

const figure = computed(() => {
  return props.stats ? formatVectors(props.stats, props.locale) : words.unavailable
})
const summary = computed(() => {
  if (props.stats) return formatSummary(props.stats, props.locale)
  return props.phase === 'loading' ? words.loadingHint : words.unavailableHint
})

/* --------------------------------- Hooks ---------------------------------- */
watch(isOpen, (open) => {
  if (open) {
    dismissal.listen()
    emit('open')
  } else {
    dismissal.release()
    emit('close')
  }
})

onBeforeUnmount(() => {
  if (isOpen.value) emit('close')
})

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
      tabindex="-1"
      :aria-expanded="isOpen"
      aria-haspopup="dialog"
      :aria-label="summary"
      :aria-describedby="tip.describedBy.value"
      @mouseenter="tip.open"
      @mouseleave="tip.close"
      @click="onToggle"
    >
      <span class="index-stats__icon" aria-hidden="true">{{ MARK }}</span>
      <span class="index-stats__figure">{{ figure }}</span>
    </button>
    <Teleport to="body">
      <Tooltip v-if="tip.at.value && !isOpen" :id="tip.id" :at="tip.at.value">
        {{ summary }}
      </Tooltip>
    </Teleport>

    <StatsPopover v-if="isOpen && stats" :index-stats="stats" :locale="locale" @close="onClose" />
  </div>
</template>
