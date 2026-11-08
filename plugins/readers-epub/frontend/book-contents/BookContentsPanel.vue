<script setup lang="ts">
/**
 * The contents of the book, standing over the page it is read on. A pointer
 * landing outside it and the Escape key both put it away.
 */
import { onBeforeUnmount, watch } from 'vue'
import { win } from '@numen/ui'
import { BookContents, type ContentsEntry } from '../book'
import { WORDS as words } from '../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  isOpen: boolean
  entries: readonly ContentsEntry[]
  offset: number
}>()

const emit = defineEmits<{ go: [offset: number]; dismiss: []; escape: [] }>()

/* --------------------------------- State ---------------------------------- */
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
    <aside v-if="props.isOpen" class="book-tab__contents">
      <BookContents
        :entries="props.entries"
        :at="props.offset"
        :words="words"
        @go="onSelectEntry"
        @close="emit('dismiss')"
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
