<script setup lang="ts">
/**
 * The contents of the book, standing over the page it is read on. A pointer
 * landing outside it and the Escape key both put it away.
 */
import { BookContents } from '../../book'
import { useWindowEscape } from '../../model/useWindowEscape'
import type { ContentsEntry } from '../../types'
import { WORDS as words } from '../../words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  isOpen: boolean
  entries: readonly ContentsEntry[]
  offset: number
}>()

const emit = defineEmits<{ go: [offset: number]; dismiss: []; escape: [] }>()

/* --------------------------------- Hooks ---------------------------------- */
useWindowEscape(
  () => props.isOpen,
  () => emit('escape'),
)

/* -------------------------------- Handlers -------------------------------- */
function onSelectEntry(offset: number) {
  emit('go', offset)
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
  --panel-bottom: 2.5rem;
  --panel-edge: 0.5rem;
  --panel-width: 15rem;
  position: absolute;
  inset-block-end: var(--panel-bottom);
  inset-inline-start: var(--panel-edge);
  z-index: 1;
  inline-size: var(--panel-width);
  max-inline-size: calc(100% - var(--panel-edge));
}

.book-tab__over-enter-active,
.book-tab__over-leave-active {
  transition:
    translate var(--numen-motion) var(--numen-easing),
    opacity var(--numen-motion) var(--numen-easing);
}

.book-tab__over-enter-from,
.book-tab__over-leave-to {
  --over-shift: 0.5rem;
  translate: 0 calc(-1 * var(--over-shift));
  opacity: 0;
}
</style>
