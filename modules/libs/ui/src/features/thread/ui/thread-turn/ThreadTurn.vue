<script setup lang="ts">
/**
 * One turn of the conversation. What was said sits in a bubble; what came back
 * is prose on the surface; what the agent reached for is a quiet line.
 *
 * A turn that never sent says so under itself.
 */
import { computed } from 'vue'
import { Prose } from '@/shared/ui/prose'
import { ToolCall } from '../tool-call'
import type { PlacedTurn, Turn } from '../../lib/turn'

const props = defineProps<{
  entry: PlacedTurn
}>()

const emit = defineEmits<{
  /** The turn pressed, which is one that says it opens something. */
  (event: 'open', turn: Turn): void
  /** A link inside the turn was pressed. */
  (event: 'follow', turn: Turn, href: string, press: MouseEvent): void
}>()

defineSlots<{
  /** How the turn is drawn, where the caller draws it itself. */
  turn?(props: { turn: Turn; state: PlacedTurn['state'] }): unknown
  /** What is said about a turn that never sent. */
  failure?(props: { turn: Turn }): unknown
}>()

/** Whether the turn is a line about work. */
const isDoing = computed(() => props.entry.turn.voice === 'doing')

/** A line about work the person can press, which opens what it was working on. */
const opens = computed(() => isDoing.value && props.entry.turn.canOpen === true)

/** What the line about a tool in hand is drawn from. */
const call = computed(() => ({
  tool: props.entry.turn.text,
  subject: props.entry.turn.subject ?? '',
  aside: props.entry.turn.aside ?? '',
  working: props.entry.state === 'arriving',
}))
</script>

<template>
  <div
    class="thread__turn flex flex-col"
    :class="entry.voice.against === 'end' ? 'items-end' : 'items-stretch'"
    :data-voice="entry.turn.voice"
    :data-state="entry.state"
  >
    <div
      class="thread__body min-w-0"
      :class="
        entry.voice.bubble
          ? 'rounded-bubble bg-bubble text-bubble-ink max-w-(--measure) px-3 py-2'
          : 'text-answer-ink'
      "
    >
      <slot v-if="$slots.turn" name="turn" :turn="entry.turn" :state="entry.state" />
      <button
        v-else-if="opens"
        type="button"
        class="thread__opens rounded-node ring-numen block w-full cursor-pointer text-start outline-none"
        @click="emit('open', entry.turn)"
      >
        <ToolCall v-bind="call" />
      </button>
      <ToolCall v-else-if="isDoing" v-bind="call" />
      <span v-else-if="entry.voice.bubble" class="thread__text">{{ entry.turn.text }}</span>
      <Prose
        v-else
        :text="entry.turn.text"
        :is-arriving="entry.state === 'arriving'"
        :unresolved="entry.turn.unresolved ?? []"
        @follow="(href: string, press: MouseEvent) => emit('follow', entry.turn, href, press)"
      />
    </div>

    <p v-if="entry.state === 'failed'" class="thread__failure text-small text-alarm mt-1">
      <slot name="failure" :turn="entry.turn">Did not send</slot>
    </p>
  </div>
</template>

<style scoped>
/* A line break that was typed is a line break that is read, and a word with
   nothing to break at is broken where it reaches the edge. */
.thread__text {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
</style>
