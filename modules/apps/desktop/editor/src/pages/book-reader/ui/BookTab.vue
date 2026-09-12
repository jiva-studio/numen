<script setup lang="ts">
/**
 * Book tab view rendering reflowable book text and contents navigation.
 */
import { onBeforeUnmount, ref, useTemplateRef, watch, watchEffect } from 'vue'
import { BookContents } from '@numen/ui'
import BookContent from './BookContent.vue'
import { WORDS as words } from '../words'
import type { BookTabState } from '../model/useBookTab'

// --- Props & Emits ---
const props = defineProps<{ state: BookTabState }>()

// --- State ---
const isListingOpen = ref(false)
const bookContent = useTemplateRef<InstanceType<typeof BookContent>>('bookContent')
watchEffect(() => props.state.setBookHandle(bookContent.value ?? null))

const panel = useTemplateRef<HTMLElement>('panel')
let detach: (() => void) | null = null

// --- Handlers ---
function onOutsidePointerDown(event: Event) {
  const target = event.target
  if (!(target instanceof Node)) return
  if (panel.value?.contains(target)) return
  isListingOpen.value = false
}

function onWindowKey(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  event.preventDefault()
  isListingOpen.value = false
  bookContent.value?.focusWay()
}

function onPageTurnKey(event: KeyboardEvent) {
  if (props.state.handleKeyPress(event)) event.preventDefault()
}

function onSelectEntry(offset: number) {
  isListingOpen.value = false
  void props.state.goToOffset(offset)
}

function onToggleListing() {
  isListingOpen.value = !isListingOpen.value
}

function onMove(offset: number) {
  void props.state.goToOffset(offset)
}

function onFollow(targetPath: string) {
  void props.state.followLink(targetPath)
}

// --- Helpers ---
function cleanupWindowListeners() {
  detach?.()
  detach = null
}

watch(isListingOpen, (open) => {
  if (!open) return cleanupWindowListeners()
  window.addEventListener('pointerdown', onOutsidePointerDown, true)
  window.addEventListener('keydown', onWindowKey, true)
  detach = () => {
    window.removeEventListener('pointerdown', onOutsidePointerDown, true)
    window.removeEventListener('keydown', onWindowKey, true)
  }
})

onBeforeUnmount(cleanupWindowListeners)
</script>

<template>
  <div
    :ref="(held: unknown) => props.state.setTabElement(held as HTMLElement | null)"
    class="book-tab"
    tabindex="-1"
    @keydown="onPageTurnKey"
  >
    <div class="book-tab__reading">
      <Transition name="book-tab__over">
        <aside v-if="isListingOpen" ref="panel" class="book-tab__contents">
          <BookContents
            :entries="props.state.contents.value"
            :at="props.state.offset.value"
            :words="words"
            @go="onSelectEntry"
          />
        </aside>
      </Transition>

      <BookContent
        ref="bookContent"
        :markup="props.state.markup.value"
        :path="props.state.drawn.value"
        :reading="props.state.reading.value"
        :book-span="props.state.span.value"
        :offset="props.state.offset.value"
        :highlights="props.state.highlights.value"
        :other-highlights="props.state.otherHighlights.value"
        :chapter="props.state.chapter.value"
        :is-listing-open="isListingOpen"
        @move="onMove"
        @follow="onFollow"
        @toggle-listing="onToggleListing"
      />
    </div>
  </div>
</template>

<style scoped>
.book-tab {
  block-size: 100%;
  min-block-size: 0;
}

.book-tab:focus,
.book-tab:focus-visible {
  outline: none;
}

.book-tab__reading {
  position: relative;
  block-size: 100%;
  min-inline-size: 0;
}

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
