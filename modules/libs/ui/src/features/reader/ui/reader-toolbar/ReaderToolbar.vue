<script setup lang="ts">
/**
 * What a person turns and zooms a document with.
 *
 * It floats over the page, and all the room there is belongs to the page. The
 * page in front and how close it is drawn are the reader's, held here.
 */
import { computed, ref } from 'vue'
import { Button } from '@/shared/ui/button'
import { clampZoom, CLOSEST, FURTHEST, NEARER, READER_WORDS, type ReaderWords } from '../../lib/strip'

withDefaults(
  defineProps<{
    /** How many pages the document has. */
    pageCount?: number
    /** The words they are drawn with. */
    words?: ReaderWords
  }>(),
  {
    pageCount: 0,
    words: () => READER_WORDS,
  },
)

/** Which page is in front, counted from the first. */
const at = defineModel<number>('at', { default: 0 })

/** How close the page is drawn, which is never past either end. */
const zoom = defineModel<number>('zoom', { default: 1, set: clampZoom })

/**
 * What is being typed over the page in front, and nothing while nothing is. A
 * number field hands over a number for what parses as one and the text itself
 * for what does not.
 */
const typing = ref<string | number | null>(null)

/** What is in the field: the page in front, or what is being typed over it. */
const typed = computed({
  get: () => typing.value ?? String(at.value + 1),
  set: (text: string | number) => {
    typing.value = text
  },
})

/**
 * A page asked for by its place in the document. The field shows the page in
 * front, and an empty field asks for nothing.
 */
const turn = () => {
  const asked = typing.value
  typing.value = null
  if (asked === null || (typeof asked === 'string' && !asked.trim())) return

  const page = Number(asked)
  if (Number.isFinite(page)) at.value = Math.round(page) - 1
}
</script>

<template>
  <div class="pointer-events-none absolute inset-x-0 bottom-inset flex justify-center">
    <!-- Standing over the page, so it carries a panel's own ground and lets
         what is behind it through. -->
    <div
      class="pointer-events-auto flex items-center gap-1 rounded-pill border border-panel-rule bg-panel p-1 shadow-panel backdrop-blur-panel"
    >
      <Button
        variant="ghost"
        size="icon-small"
        class="rounded-pill"
        :aria-label="words.back"
        :disabled="at <= 0"
        @click="at -= 1"
      >
        <svg
          viewBox="0 0 16 16"
          class="size-4"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M10 3 5 8l5 5" />
        </svg>
      </Button>
      <Button
        variant="ghost"
        size="icon-small"
        class="rounded-pill"
        :aria-label="words.next"
        :disabled="at >= pageCount - 1"
        @click="at += 1"
      >
        <svg
          viewBox="0 0 16 16"
          class="size-4"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="m6 3 5 5-5 5" />
        </svg>
      </Button>

      <input
        v-model="typed"
        class="reader__at w-10 rounded-node border border-field-rule bg-field px-1 text-center text-small text-ink outline-none ring-numen"
        type="number"
        min="1"
        :max="pageCount"
        :aria-label="words.page"
        @change="turn"
        @keydown.enter="turn"
      />
      <span class="pe-1 text-small text-hushed">/ {{ pageCount }}</span>

      <Button
        variant="ghost"
        size="icon-small"
        class="rounded-pill"
        :aria-label="words.further"
        :disabled="zoom <= FURTHEST"
        @click="zoom /= NEARER"
      >
        <svg
          viewBox="0 0 16 16"
          class="size-4"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
        >
          <path d="M3.5 8h9" />
        </svg>
      </Button>
      <Button
        variant="ghost"
        size="icon-small"
        class="rounded-pill"
        :aria-label="words.closer"
        :disabled="zoom >= CLOSEST"
        @click="zoom *= NEARER"
      >
        <svg
          viewBox="0 0 16 16"
          class="size-4"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
        >
          <path d="M8 3.5v9M3.5 8h9" />
        </svg>
      </Button>
    </div>
  </div>
</template>

<style scoped>
/* The field carries the page and nothing else: a number field's own arrows are
   not drawn. */
.reader__at::-webkit-inner-spin-button,
.reader__at::-webkit-outer-spin-button {
  appearance: none;
  margin: 0;
}

.reader__at {
  appearance: textfield;
}
</style>
