<script setup lang="ts">
/**
 * The conversation. What was said sits in a bubble; what came back is text on
 * the surface.
 *
 * Scrolls on its own, and stays at the foot while an answer arrives — that is
 * the browser's scroll anchoring, so nothing here measures or moves it. What
 * a turn's text is made of belongs to whoever renders the thread: the default
 * keeps the line breaks and does nothing else with it.
 */
import { computed } from 'vue'
import { placeTurns, type Turn } from './model'

const props = defineProps<{
  turns: readonly Turn[]
}>()

const placed = computed(() => placeTurns(props.turns))
</script>

<template>
  <div
    class="numen flex min-h-0 flex-col gap-turn overflow-y-auto overscroll-contain font-sans text-base"
  >
    <p v-if="!placed.length" class="m-auto text-hushed">
      <slot name="silence">Nothing said yet</slot>
    </p>

    <div
      v-for="entry in placed"
      :key="entry.turn.id"
      class="thread__turn flex flex-col"
      :class="entry.voice.against === 'end' ? 'items-end' : 'items-stretch'"
      :data-voice="entry.turn.voice"
      :data-state="entry.state"
    >
      <div
        class="thread__body min-w-0"
        :class="
          entry.voice.bubble
            ? 'max-w-(--numen-bubble-measure) rounded-bubble bg-bubble px-3 py-2 text-bubble-ink'
            : 'text-answer-ink'
        "
      >
        <slot name="turn" :turn="entry.turn" :state="entry.state">
          <span class="thread__text">{{ entry.turn.text }}</span>
        </slot>
        <span v-if="entry.caret" class="thread__caret" />
      </div>

      <p v-if="entry.state === 'failed'" class="mt-1 text-small text-alarm">
        <slot name="failure" :turn="entry.turn">Did not send</slot>
      </p>
    </div>
  </div>
</template>

<style scoped>
/* A line break that was typed is a line break that is read, and a word with
   nothing to break at is broken where it reaches the edge. */
.thread__text {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

/* Sits on the last line of what has arrived, so it moves with the text. */
.thread__caret {
  display: inline-block;
  inline-size: 0.5em;
  block-size: 1em;
  margin-inline-start: 0.15em;
  vertical-align: text-bottom;
  background: currentColor;
  animation: thread-caret 1s steps(2, start) infinite;
}

@keyframes thread-caret {
  50% {
    opacity: 0;
  }
}
</style>
