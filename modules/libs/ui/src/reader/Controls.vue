<script setup lang="ts">
/**
 * What a person turns and zooms a document with, standing over the page.
 *
 * It floats: a row of its own would take a strip of the room off every document
 * for the whole time one is open, and what the room is for is the page.
 */
import { ref, watch } from 'vue'
import { Button } from '@/components/ui/button'
import { CLOSEST, FURTHEST, NEARER } from './strip'

const props = withDefaults(
  defineProps<{
    /** How many pages the document has. */
    pages?: number
    /** Which page is in front, counted from the first. */
    at?: number
    /** How close the page is drawn. What it may be is the reader's rule. */
    zoom?: number
    /** What turning back a page is called, and turning on. */
    back?: string
    next?: string
    /** What the field the page is typed in is called. */
    page?: string
    /** What drawing the page larger is called, and smaller. */
    closer?: string
    further?: string
  }>(),
  {
    pages: 0,
    at: 0,
    zoom: 1,
    back: 'Previous page',
    next: 'Next page',
    page: 'Page',
    closer: 'Closer',
    further: 'Further',
  },
)

const emit = defineEmits<{
  /** The page to turn to, counted from the first. */
  (event: 'go', page: number): void
  /** Draw the page this many times closer. */
  (event: 'draw', how: number): void
}>()

/** What is typed in the field, which follows the page in front. */
const typed = ref(String(props.at + 1))
watch(
  () => props.at,
  (page) => {
    typed.value = String(page + 1)
  },
)

/** A page asked for by its place in the document. The field shows the page in front. */
const turn = () => {
  const page = Number(typed.value)
  if (Number.isFinite(page)) emit('go', Math.round(page) - 1)
  typed.value = String(props.at + 1)
}
</script>

<template>
  <div class="reader__controls pointer-events-none absolute inset-x-0 bottom-inset flex justify-center">
    <!-- Standing over the page, so it carries a panel's own ground and lets
         what is behind it through. -->
    <div
      class="reader__pill pointer-events-auto flex items-center gap-1 rounded-full border border-panel-rule bg-panel p-1 shadow-panel backdrop-blur-panel"
    >
      <Button
        variant="ghost"
        size="icon-small"
        class="rounded-full"
        :aria-label="back"
        :disabled="at <= 0"
        @click="emit('go', at - 1)"
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
        class="rounded-full"
        :aria-label="next"
        :disabled="at >= pages - 1"
        @click="emit('go', at + 1)"
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
        class="reader__at w-10 rounded-node border border-field-rule bg-field px-1 text-center text-small text-ink outline-none focus-visible:ring-(length:--numen-ring-width) focus-visible:ring-ring"
        type="number"
        min="1"
        :max="pages"
        :aria-label="page"
        @change="turn"
        @keydown.enter="turn"
      />
      <span class="pe-1 text-small text-hushed">/ {{ pages }}</span>

      <Button
        variant="ghost"
        size="icon-small"
        class="rounded-full"
        :aria-label="further"
        :disabled="zoom <= FURTHEST"
        @click="emit('draw', 1 / NEARER)"
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
        class="rounded-full"
        :aria-label="closer"
        :disabled="zoom >= CLOSEST"
        @click="emit('draw', NEARER)"
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
