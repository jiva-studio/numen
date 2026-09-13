<script setup lang="ts">
/**
 * What a person turns the document with: a step either way, and the page in
 * front in a field they can type over.
 */
import { computed, ref } from 'vue'
import { Button } from '@/shared/ui/button'
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
const at = defineModel<number>('at', { default: 0 })

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
  <div class="flex items-center gap-1">
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
