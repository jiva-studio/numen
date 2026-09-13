<script setup lang="ts">
/**
 * What a person draws the page closer and further with, a step at a time. Each
 * end is offered no press once it is reached.
 */
import { Button } from '@/shared/ui/button'
import { CLOSEST, FURTHEST, NEARER, READER_WORDS, type ReaderWords } from '../../../lib/strip'

withDefaults(
  defineProps<{
    /** The words they are drawn with. */
    words?: ReaderWords
  }>(),
  { words: () => READER_WORDS },
)

/** How close the page is drawn. */
const zoom = defineModel<number>('zoom', { default: 1 })
</script>

<template>
  <div class="flex items-center gap-1">
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
</template>
