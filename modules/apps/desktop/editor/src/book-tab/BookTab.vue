<script setup lang="ts">
/**
 * A book tab: the document of the book the person is standing in, and the list
 * of what the book divides into, which comes over it.
 *
 * What the book could not be read as is said in the window's own notices.
 */
import { onBeforeUnmount, ref, useTemplateRef, watch, watchEffect } from 'vue'
import { Book, BookContents } from '@numen/ui'
import { ListTree } from '@lucide/vue'
import { WORDS as words } from './words'
import type { BookHandle, BookTabState } from './kind'

const props = defineProps<{ state: BookTabState }>()

/** Whether the list of what the book divides into stands over the text. */
const listing = ref(false)

/**
 * The book as it is drawn, handed to what the tab holds. The tab reads the keys
 * and the book turns the page, so a tab holding no book answers every key with
 * "not mine" and the arrows turn nothing at all.
 */
const book = useTemplateRef<BookHandle>('book')
watchEffect(() => props.state.holdsBook(book.value))

const panel = useTemplateRef<HTMLElement>('panel')
const way = useTemplateRef<HTMLElement>('way')

/** A press that landed neither in the list nor on the way into it. */
const onOutsidePointerDown = (event: Event) => {
  const target = event.target
  if (!(target instanceof Node)) return
  if (panel.value?.contains(target) || way.value?.contains(target)) return
  listing.value = false
}

const onWindowKey = (event: KeyboardEvent) => {
  if (event.key !== 'Escape') return
  event.preventDefault()
  listing.value = false
  way.value?.focus()
}

/**
 * A key struck at the book itself. The window hands one to the tab of the pane
 * the person is in, and several panes are drawn at once: a book they are
 * looking at and reading from holds the keyboard, whichever pane the layout
 * calls the one they are in.
 */
const onPageTurnKey = (event: KeyboardEvent) => {
  if (props.state.pressed(event)) event.preventDefault()
}

/** What the open list installed on the window, if anything. */
let detach: (() => void) | null = null

const leave = () => {
  detach?.()
  detach = null
}

watch(listing, (open) => {
  if (!open) return leave()
  // Escape is read before the panel the tab stands in sees it.
  window.addEventListener('pointerdown', onOutsidePointerDown, true)
  window.addEventListener('keydown', onWindowKey, true)
  detach = () => {
    window.removeEventListener('pointerdown', onOutsidePointerDown, true)
    window.removeEventListener('keydown', onWindowKey, true)
  }
})

onBeforeUnmount(leave)

/** A place chosen in the list: the book is turned to it and the list goes. */
const onSelectEntry = (at: number) => {
  listing.value = false
  void props.state.go(at)
}
</script>

<template>
  <div
    :ref="(held: unknown) => props.state.holdsTab(held as HTMLElement | null)"
    class="book-tab"
    tabindex="-1"
    @keydown="onPageTurnKey"
  >
    <div class="book-tab__reading">
      <Transition name="book-tab__over">
        <aside v-if="listing" ref="panel" class="book-tab__contents">
          <BookContents
            :entries="props.state.contents.value"
            :at="props.state.at.value"
            :words="words"
            @go="chose"
          />
        </aside>
      </Transition>

      <Book
        ref="book"
        :markup="props.state.markup.value"
        :path="props.state.drawn.value"
        :span="props.state.reading.value"
        :book="props.state.span.value"
        :at="props.state.at.value"
        :highlights="props.state.highlights.value"
        :elsewhere="props.state.elsewhere.value"
        :chapter="props.state.chapter.value"
        :words="words"
        @moved="(at: number) => void props.state.go(at)"
        @followed="(path: string) => void props.state.follow(path)"
      >
        <template #way>
          <button
            ref="way"
            type="button"
            class="book-tab__list"
            :aria-label="listing ? words.hides : words.shows"
            :aria-pressed="listing"
            :aria-expanded="listing"
            @click="listing = !listing"
          >
            <ListTree class="size-4" aria-hidden="true" />
          </button>
        </template>
      </Book>
    </div>
  </div>
</template>

<style scoped>
.book-tab {
  block-size: 100%;
  min-block-size: 0;
}

/* It holds the keyboard so that nothing else reads the arrows over it, and
   nothing is drawn around it for holding one. */
.book-tab:focus,
.book-tab:focus-visible {
  outline: none;
}

.book-tab__reading {
  position: relative;
  block-size: 100%;
  min-inline-size: 0;
}

/* It stands on the line under the text and is drawn as what it is: a mark to
   press, in the colour the line is set in. */
.book-tab__list {
  display: grid;
  place-items: center;
  color: inherit;
}

.book-tab__list:focus-visible {
  outline: none;
}

/* The list comes down over the text and takes no room from it, so the columns
   behind it stand where they stood. */
.book-tab__contents {
  position: absolute;
  inset-block-end: 2.5rem;
  inset-inline-start: 0.5rem;
  z-index: 1;
  inline-size: 15rem;
  max-inline-size: calc(100% - 0.5rem);
  /* A height of its own, so a book of a thousand names scrolls inside it. */
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
