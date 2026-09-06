<script setup lang="ts">
/**
 * A book tab: the document of the book the person is standing in, and the list
 * of what the book divides into beside it.
 *
 * What the book could not be read as is said where the text would be.
 */
import { ref } from 'vue'
import { BookContents, BookReader } from '@numen/ui'
import { WORDS as words } from './words'
import type { BookTabState } from './kind'

const props = defineProps<{ state: BookTabState }>()

/** Whether the list of what the book divides into stands beside the text. */
const listing = ref(false)
</script>

<template>
  <div class="book-tab">
    <aside v-if="listing" class="book-tab__contents">
      <BookContents
        :entries="props.state.contents.value"
        :at="props.state.at.value"
        :words="words"
        @go="(at: number) => void props.state.go(at)"
      />
    </aside>

    <div class="book-tab__reading">
      <button
        type="button"
        class="book-tab__list"
        :aria-label="listing ? words.hides : words.shows"
        :aria-pressed="listing"
        @click="listing = !listing"
      >
        {{ words.contents }}
      </button>

      <p v-if="props.state.trouble.value" class="book-tab__silence">
        {{ props.state.trouble.value }}
      </p>
      <BookReader
        v-else
        :ref="(reader: unknown) => props.state.drew(reader)"
        :markup="props.state.markup.value"
        :span="props.state.reading.value"
        :book="props.state.span.value"
        :at="props.state.at.value"
        :marked="props.state.marked.value"
        :also="props.state.also.value"
        :words="words"
        @go="(at: number) => void props.state.go(at)"
      />
    </div>
  </div>
</template>

<style scoped>
.book-tab {
  display: flex;
  block-size: 100%;
  min-block-size: 0;
}

.book-tab__contents {
  flex: 0 0 auto;
  inline-size: 15rem;
  min-block-size: 0;
  border-inline-end: var(--numen-stroke) solid var(--numen-rule);
}

.book-tab__reading {
  position: relative;
  flex: 1 1 auto;
  min-inline-size: 0;
  block-size: 100%;
}

/* Standing over the text, so it carries a panel's own ground and lets what is
   behind it through. */
.book-tab__list {
  position: absolute;
  inset-block-start: 0.25rem;
  inset-inline-start: 0.25rem;
  z-index: 1;
  padding: 0.25rem 0.5rem;
  border: var(--numen-stroke) solid var(--numen-panel-border);
  border-radius: var(--numen-radius-tight);
  background: var(--numen-panel-bg);
  backdrop-filter: blur(var(--numen-panel-blur));
  font-size: var(--numen-text-1);
}

.book-tab__list:focus-visible {
  outline: var(--numen-ring-width) solid var(--numen-ring);
}

.book-tab__silence {
  padding: var(--numen-inset);
  font-size: var(--numen-text-1);
  color: var(--numen-hushed);
}
</style>
