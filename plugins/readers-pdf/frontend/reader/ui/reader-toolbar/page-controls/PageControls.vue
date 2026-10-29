<script setup lang="ts">
/**
 * The controls that change the page: a step in either direction, and an
 * editable field holding the current page.
 */
import { computed, ref } from 'vue'
import { Button } from '@numen/ui'
import { READER_WORDS, type ReaderWords } from '../../../lib/strip'

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
const currentPage = defineModel<number>('at', { default: 0 })

/**
 * The text being edited in the page field, empty when idle. A number field
 * yields a number for input that parses as one and the text itself otherwise.
 */
const typing = ref<string | number | null>(null)

/** What is in the field: the page in front, or what is being typed over it. */
const inputValue = computed({
  get: () => typing.value ?? String(currentPage.value + 1),
  set: (text: string | number) => {
    typing.value = text
  },
})

/**
 * Goes to a page by its position in the document. The field shows the current
 * page, and an empty field requests nothing.
 */
const turn = () => {
  const rawInput = typing.value
  typing.value = null
  if (rawInput === null || (typeof rawInput === 'string' && !rawInput.trim())) return

  const page = Number(rawInput)
  if (Number.isFinite(page)) currentPage.value = Math.round(page) - 1
}
</script>

<template>
  <div class="flex items-center gap-1">
    <Button
      variant="ghost"
      size="icon-small"
      class="rounded-pill"
      :aria-label="words.back"
      :disabled="currentPage <= 0"
      @click="currentPage -= 1"
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
      :disabled="currentPage >= pageCount - 1"
      @click="currentPage += 1"
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
      v-model="inputValue"
      class="reader__at rounded-node border-field-rule bg-field text-small text-ink ring-numen w-10 border px-1 text-center outline-none"
      type="number"
      min="1"
      :max="pageCount"
      :aria-label="words.page"
      @change="turn"
      @keydown.enter="turn"
    />
    <span class="text-small text-hushed pe-1">/ {{ pageCount }}</span>
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
