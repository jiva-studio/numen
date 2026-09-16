<script setup lang="ts">
/** One notice as a card: the line of work it draws, and the way to put it away. */
import { useTemplateRef } from 'vue'
import { Activity } from './activity'
import { tallyOf, type Notice } from '../lib/notice'

defineProps<{
  /** The notice this card stands for. */
  one: Notice
  /** What the way to dismiss it is called. */
  dismiss: string
  /** How long the count has left, in the words it is read in. */
  left: string
}>()

const emit = defineEmits<{
  /** A person pressed the way away. */
  (event: 'dismiss'): void
}>()

/** The way away, which whoever draws the card keeps the keyboard on. */
const way = useTemplateRef<HTMLElement>('way')

defineExpose({ way })
</script>

<template>
  <article class="notice" :data-tone="one.tone ?? 'plain'">
    <Activity
      class="notice__work"
      :text="one.text"
      :about="one.about ?? ''"
      :tally="tallyOf(one)"
      :working="one.working ?? false"
      :left="left"
      :tone="one.tone ?? 'plain'"
    />
    <button
      ref="way"
      type="button"
      class="notice__away ring-numen outline-none"
      :aria-label="`${dismiss}: ${one.text}`"
      @click="emit('dismiss')"
    >
      <svg viewBox="0 0 12 12" aria-hidden="true" focusable="false">
        <path d="M3 3 L9 9 M9 3 L3 9" />
      </svg>
    </button>
  </article>
</template>

<style scoped>
.notice__work {
  min-inline-size: 0;
  flex: 1;
}

.notice__away {
  flex: none;
  display: grid;
  place-items: center;
  inline-size: 1.25rem;
  block-size: 1.25rem;
  border-radius: var(--numen-radius-pill);
  color: inherit;
  opacity: 0.5;
}

.notice__away:hover,
.notice__away:focus-visible {
  opacity: 1;
  background: color-mix(in oklab, currentColor 12%, transparent);
}

.notice__away svg {
  inline-size: 0.75rem;
  block-size: 0.75rem;
  stroke: currentColor;
  stroke-width: 1.5;
  stroke-linecap: round;
  fill: none;
}
</style>
