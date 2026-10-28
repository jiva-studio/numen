<script setup lang="ts">
import { win } from '../dom'
/**
 * The contents of the book, standing over the page it is read on. A pointer
 * landing outside it and the Escape key both put it away.
 */
import { onBeforeUnmount, useTemplateRef, watch } from 'vue'
import { useOutsidePress } from '@numen/ui'
import { BookContents, type ContentsEntry } from '../book'
import { WORDS as words } from '../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  /** Whether the contents stand over the page. */
  isOpen: boolean
  /** The places in the book worth jumping to. */
  entries: readonly ContentsEntry[]
  /** Where in the book the page being read begins. */
  offset: number
}>()

const emit = defineEmits<{ go: [offset: number]; dismiss: []; escape: [] }>()

/* --------------------------------- State ---------------------------------- */
const panel = useTemplateRef<HTMLElement>('panel')
let detach: (() => void) | null = null

/* --------------------------------- Hooks ---------------------------------- */
watch(
  () => props.isOpen,
  (isOpen) => {
    if (!isOpen) return cleanupWindowListeners()
    win.addEventListener('keydown', onWindowKey, true)
    detach = () => win.removeEventListener('keydown', onWindowKey, true)
  },
)

useOutsidePress(
  panel,
  () => emit('dismiss'),
  () => props.isOpen,
)

onBeforeUnmount(cleanupWindowListeners)

/* -------------------------------- Handlers -------------------------------- */
function onSelectEntry(offset: number) {
  emit('go', offset)
}

function onWindowKey(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  event.preventDefault()
  emit('escape')
}

/* -------------------------------- Helpers --------------------------------- */
function cleanupWindowListeners() {
  detach?.()
  detach = null
}
</script>

<template>
  <Transition name="book-tab__over">
    <aside v-if="props.isOpen" ref="panel" class="book-tab__contents">
      <BookContents
        :entries="props.entries"
        :offset="props.offset"
        :words="words"
        @go="onSelectEntry"
      />
    </aside>
  </Transition>
</template>

<style scoped>
.book-tab__contents {
  position: absolute;
  inset-block-end: 2.5rem;
  inset-inline-start: 0.5rem;
  z-index: 1;
  inline-size: 15rem;
  max-inline-size: calc(100% - 0.5rem);
  block-size: min(26rem, calc(100% - 3rem));
  overflow: hidden;
  border: var(--numen-stroke) solid var(--numen-panel-border);
  border-radius: var(--numen-radius-panel);
  background: var(--numen-panel-bg);
  backdrop-filter: blur(var(--numen-panel-blur));
  box-shadow: var(--numen-panel-shadow);
}

.book-tab__over-enter-active,
.book-tab__over-leave-active {
  transition:
    translate var(--numen-motion) var(--numen-easing),
    opacity var(--numen-motion) var(--numen-easing);
}

.book-tab__over-enter-from,
.book-tab__over-leave-to {
  translate: 0 -0.5rem;
  opacity: 0;
}
</style>
