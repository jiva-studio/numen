<script setup lang="ts">
/**
 * the surface; what the agent reached for is a quiet line between them.
 *
 * Scrolls on its own, and follows the foot: a turn that arrives is brought
 * into view, and reading further up leaves it where it was read until the foot
 * is reached again.
 */
import { computed, onMounted, ref, useTemplateRef, watch } from 'vue'
import { ThreadTurn } from './thread-turn'
import { atFoot, getFoot } from '../lib/foot'
import { placeTurns, type PlacedTurn, type Turn } from '../lib/turn'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  turns: readonly Turn[]
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** A turn the user pressed that is openable. */
  (event: 'open', turn: Turn): void
  (event: 'follow', turn: Turn, href: string, press: MouseEvent): void
}>()

defineSlots<{
  /** Content shown when there are no turns yet. */
  silence?(): unknown
  /** How a turn is drawn, where the caller draws it itself. */
  turn?(props: { turn: Turn; state: PlacedTurn['state'] }): unknown
  /** Content shown for a turn that failed to send. */
  failure?(props: { turn: Turn }): unknown
}>()

/* --------------------------------- State ---------------------------------- */
const placedTurns = computed(() => placeTurns(props.turns))

const area = useTemplateRef<HTMLElement>('area')

/** Whether the foot is followed. Scrolling away from it stops that. */
const follows = ref(true)

/* --------------------------------- Hooks ---------------------------------- */
watch(
  () => {
    const turns = props.turns
    const last = turns[turns.length - 1]
    return [turns.length, last?.id, last?.text, last?.state, last?.aside]
  },
  () => toFoot(),
  { flush: 'post' },
)

onMounted(() => toFoot())

/* -------------------------------- Handlers -------------------------------- */
function onScroll() {
  if (area.value) follows.value = atFoot(area.value)
}

/* -------------------------------- Helpers --------------------------------- */
/** Scrolls the foot into view. With `again` set, it resumes following new content. */
function toFoot(again = false) {
  if (again) follows.value = true
  if (!follows.value || !area.value) return
  area.value.scrollTop = getFoot(area.value)
}

defineExpose({ toFoot })
</script>

<template>
  <div
    ref="area"
    class="thread numen gap-turn flex min-h-0 flex-col overflow-y-auto overscroll-contain font-sans text-base"
    @scroll="onScroll"
  >
    <p v-if="!placedTurns.length" class="text-hushed m-auto">
      <slot name="silence">No messages yet</slot>
    </p>

    <ThreadTurn
      v-for="entry in placedTurns"
      :key="entry.turn.id"
      :entry="entry"
      @open="emit('open', $event)"
      @follow="(turn, href, press) => emit('follow', turn, href, press)"
    >
      <template v-if="$slots.turn" #turn="bound">
        <slot name="turn" v-bind="bound" />
      </template>

      <template v-if="$slots.failure" #failure="bound">
        <slot name="failure" v-bind="bound" />
      </template>
    </ThreadTurn>
  </div>
</template>

<style scoped>
/* Nothing is drawn to scroll with. */
.thread {
  /* How much of the width one bubble may take before it wraps. */
  --measure: 80%;

  scrollbar-width: none;
}

.thread::-webkit-scrollbar {
  display: none;
}
</style>
