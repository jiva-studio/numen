<script setup lang="ts">
/**
 * The title bar item that counts the plugins that run, and the popover that lists them.
 */
import { computed, onBeforeUnmount, ref, useTemplateRef, watch } from 'vue'
import { useMenuDismissal } from '@numen/ui'
import type { PluginProcess } from '../types'
import type { Phase } from '../model/plugins'
import { formatCount, formatSummary, MARK } from '../lib/format'
import PluginsPopover from './PluginsPopover.vue'
import { WORDS as words } from '../words'

/** How often the open popover writes the times again. */
const TICK_MS = 1_000

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    processes: readonly PluginProcess[]
    phase: Phase
    locale?: string | undefined
    /** The time now, in milliseconds since the Unix epoch. */
    clock?: () => number
  }>(),
  { locale: undefined, clock: () => Date.now() },
)

/* --------------------------------- State ---------------------------------- */
const isOpen = ref(false)
const now = ref(props.clock())
const root = useTemplateRef<HTMLElement>('root')
// The popover is placed against the item by CSS and follows it through a resize.
const dismissal = useMenuDismissal(root, onClose, isResize)
let tick: number | undefined

const isReady = computed(() => props.phase === 'ready')
const figure = computed(() =>
  isReady.value ? formatCount(props.processes.length, props.locale) : words.unavailable,
)
const summary = computed(() => {
  if (isReady.value) return formatSummary(props.processes.length, props.locale)
  return props.phase === 'loading' ? words.loadingHint : words.unavailableHint
})

/* --------------------------------- Hooks ---------------------------------- */
watch(isOpen, (open) => {
  if (open) {
    now.value = props.clock()
    tick = window.setInterval(() => (now.value = props.clock()), TICK_MS)
    dismissal.listen()
    return
  }
  stopTick()
  dismissal.release()
})

onBeforeUnmount(stopTick)

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

function stopTick() {
  if (tick !== undefined) window.clearInterval(tick)
  tick = undefined
}
</script>

<template>
  <div ref="root" class="plugin-stats">
    <button
      type="button"
      class="plugin-stats__trigger"
      tabindex="-1"
      :aria-expanded="isOpen"
      aria-haspopup="dialog"
      :title="summary"
      :aria-label="summary"
      @click="onToggle"
    >
      <span class="plugin-stats__icon" aria-hidden="true">{{ MARK }}</span>
      <span class="plugin-stats__figure">{{ figure }}</span>
    </button>

    <PluginsPopover
      v-if="isOpen && isReady"
      :processes="processes"
      :now="now"
      :locale="locale"
      @close="onClose"
    />
  </div>
</template>
