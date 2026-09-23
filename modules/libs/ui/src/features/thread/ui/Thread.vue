<script setup lang="ts">
/**
 * The conversation. What was said sits in a bubble; what came back is prose on
 * the surface; what the agent reached for is a quiet line between them.
 *
 * Scrolls on its own, and follows the foot: a turn that arrives is brought
 * into view, and reading further up leaves it where it was read until the foot
 * is reached again.
 */
import { computed, onMounted, ref, useTemplateRef, watch } from 'vue'
import { ThreadTurn } from './thread-turn'
import { atFoot, footOf } from '../lib/foot'
import { placeTurns, type PlacedTurn, type Turn } from '../lib/turn'

const props = defineProps<{
  turns: readonly Turn[]
}>()

const emit = defineEmits<{
  /** A turn the person pressed, which is one that says it opens something. */
  (event: 'open', turn: Turn): void
  /** A link inside a turn was pressed, with the turn it stands in. */
  (event: 'follow', turn: Turn, href: string, press: MouseEvent): void
}>()

defineSlots<{
  /** What is said when nothing has been said yet. */
  silence?(): unknown
  /** How a turn is drawn, where the caller draws it itself. */
  turn?(props: { turn: Turn; state: PlacedTurn['state'] }): unknown
  /** What is said about a turn that never sent. */
  failure?(props: { turn: Turn }): unknown
}>()

const placed = computed(() => placeTurns(props.turns))

const area = useTemplateRef<HTMLElement>('area')

/** Whether the foot is followed. Scrolling away from it stops that. */
const follows = ref(true)

const onScroll = () => {
  if (area.value) follows.value = atFoot(area.value)
}

/** Brings the foot into view. Asked to, it takes the following up again. */
const toFoot = (again = false) => {
  if (again) follows.value = true
  if (!follows.value || !area.value) return
  area.value.scrollTop = footOf(area.value)
}

watch(
  () => props.turns,
  () => toFoot(),
  { deep: true, flush: 'post' },
)

onMounted(() => toFoot())

defineExpose({ toFoot })
</script>

<template>
  <div
    ref="area"
    class="thread numen gap-turn flex min-h-0 flex-col overflow-y-auto overscroll-contain font-sans text-base"
    @scroll="onScroll"
  >
    <p v-if="!placed.length" class="text-hushed m-auto">
      <slot name="silence">Nothing said yet</slot>
    </p>

    <ThreadTurn
      v-for="entry in placed"
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
