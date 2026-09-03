<script setup lang="ts">
/**
 * The conversation. What was said sits in a bubble; what came back is prose on
 * the surface; what the agent reached for is a quiet line between them.
 *
 * Scrolls on its own, and stays at the foot while an answer arrives: that is
 * the browser's own scroll anchoring.
 */
import { computed } from 'vue'
import Prose from '../prose/Prose.vue'
import Tool from '../tool/Tool.vue'
import { placeTurns, type PlacedTurn, type Turn } from './model'

const props = defineProps<{
  turns: readonly Turn[]
}>()

const emit = defineEmits<{
  /** A turn the person pressed, which is one that says it opens something. */
  (event: 'open', turn: Turn): void
  /** A link inside a turn was pressed, with the turn it stands in. */
  (event: 'follow', turn: Turn, href: string, press: MouseEvent): void
}>()

const placed = computed(() => placeTurns(props.turns))

/** What the line about a tool in hand is drawn from. */
const toolOf = (entry: PlacedTurn) => ({
  tool: entry.turn.text,
  about: entry.turn.about ?? '',
  aside: entry.turn.aside ?? '',
  working: entry.state === 'arriving',
})
</script>

<template>
  <div
    class="thread numen flex min-h-0 flex-col gap-turn overflow-y-auto overscroll-contain font-sans text-base"
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
          <template v-if="entry.turn.voice === 'doing'">
            <button
              v-if="entry.turn.opens"
              type="button"
              class="thread__opens block w-full cursor-pointer rounded-node text-start outline-none focus-visible:ring-(length:--numen-ring-width) focus-visible:ring-ring"
              @click="emit('open', entry.turn)"
            >
              <Tool v-bind="toolOf(entry)" />
            </button>
            <Tool v-else v-bind="toolOf(entry)" />
          </template>
          <span v-else-if="entry.voice.bubble" class="thread__text">{{ entry.turn.text }}</span>
          <Prose
            v-else
            :text="entry.turn.text"
            :arriving="entry.state === 'arriving'"
            @follow="(href: string, press: MouseEvent) => emit('follow', entry.turn, href, press)"
          />
        </slot>
      </div>

      <p v-if="entry.state === 'failed'" class="thread__failure mt-1 text-small text-alarm">
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

/* Nothing is drawn to scroll with. */
.thread {
  scrollbar-width: none;
}

.thread::-webkit-scrollbar {
  display: none;
}
</style>
