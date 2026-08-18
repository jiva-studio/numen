<script setup lang="ts">
/**
 * An agent talked to: the conversation, and the field it is carried on with.
 *
 * The composer is written over the conversation, and the words pass behind it
 * as they scroll. How much room it takes is measured on every change of size.
 *
 * It fills whatever it is put in, and says nothing about where that is.
 */
import { onBeforeUnmount, onMounted, ref, useTemplateRef } from 'vue'
import Thread from '../thread/Thread.vue'
import Composer from '../composer/Composer.vue'
import type { Turn } from '../thread/model'

withDefaults(
  defineProps<{
    turns: readonly Turn[]
    /** An answer is on its way. */
    working?: boolean
    placeholder?: string
    disabled?: boolean
    /** What the disc at the end of the field is called while it sends. */
    sends?: string
    /** What it is called while it stops the answer on its way. */
    stops?: string
  }>(),
  {
    working: false,
    placeholder: 'Write a message',
    disabled: false,
    sends: 'Send',
    stops: 'Stop',
  },
)

const emit = defineEmits<{
  (event: 'submit', text: string): void
  /** Give up on the answer on its way. */
  (event: 'stop'): void
}>()

const text = defineModel<string>({ default: '' })

const composer = useTemplateRef<InstanceType<typeof Composer>>('composer')
const room = ref('0px')

let watching: ResizeObserver | undefined

onMounted(() => {
  const element = composer.value?.$el as HTMLElement | undefined
  if (!element || typeof ResizeObserver === 'undefined') return
  // The border box: what the conversation has to clear is the whole field,
  // its padding and its edge included.
  watching = new ResizeObserver(([seen]) => {
    if (seen) room.value = `${seen.borderBoxSize?.[0]?.blockSize ?? element.offsetHeight}px`
  })
  watching.observe(element)
})

onBeforeUnmount(() => watching?.disconnect())
</script>

<template>
  <div
    class="agent numen flex min-h-0 flex-col font-sans text-base text-ink"
    :style="{ '--agent-room': room }"
  >
    <Thread class="agent__thread" :turns="turns">
      <template #silence><slot name="silence">Nothing said yet</slot></template>
      <template v-if="$slots.turn" #turn="bound"><slot name="turn" v-bind="bound" /></template>
      <template #failure="bound"><slot name="failure" v-bind="bound">Did not send</slot></template>
    </Thread>

    <Composer
      ref="composer"
      v-model="text"
      class="agent__composer"
      :working="working"
      :placeholder="placeholder"
      :disabled="disabled"
      :sends="sends"
      :stops="stops"
      @submit="emit('submit', $event)"
      @stop="emit('stop')"
    />
  </div>
</template>

<style scoped>
.agent {
  /* What the conversation is kept clear of at the sides, how far it fades
     where it passes behind an edge, and how far the composer stands off the
     foot. */
  --inset: var(--numen-gutter);
  --fade: 20px;
  --lift: 12px;
  /* What is left between the last turn and the composer written over it. */
  --breath: 28px;

  position: relative;
  block-size: 100%;
  padding-inline: var(--inset);
}

/* Takes the whole of it and scrolls inside. The words run on to the composer's
   own edge and fade out over the band it stands in. */
.agent__thread {
  /* The band the composer stands in, up from the foot, and what the last turn
     is held clear of above it. */
  --behind: calc(var(--agent-room) + var(--lift));
  --clear: calc(var(--behind) + var(--lift) + var(--breath));

  flex: 1;
  min-height: 0;
  padding-block-start: var(--fade);
  padding-block-end: var(--clear);
  mask-image: linear-gradient(
    to bottom,
    transparent 0,
    #000 var(--fade),
    #000 calc(100% - var(--behind)),
    transparent calc(100% - var(--behind) + var(--fade))
  );
}

.agent__composer {
  position: absolute;
  inset-inline: var(--inset);
  inset-block-end: var(--lift);
}
</style>
