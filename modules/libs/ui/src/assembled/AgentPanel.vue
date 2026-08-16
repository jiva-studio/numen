<script setup lang="ts">
/**
 * The panel an agent is talked to in: the conversation, and the field it is
 * carried on with.
 *
 * The composer is written over the conversation, and the words pass behind it
 * as they scroll. How much room it takes is measured on every change of size.
 */
import { onBeforeUnmount, onMounted, ref, useTemplateRef } from 'vue'
import Panel from '../panel/Panel.vue'
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
  }>(),
  { working: false, placeholder: 'Write a message', disabled: false },
)

const emit = defineEmits<{
  (event: 'submit', text: string): void
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
  <Panel class="agent-panel" :style="{ '--agent-panel-room': room }">
    <Thread class="agent-panel__thread" :turns="turns">
      <template #silence><slot name="silence">Nothing said yet</slot></template>
      <template v-if="$slots.turn" #turn="bound"><slot name="turn" v-bind="bound" /></template>
      <template #failure="bound"><slot name="failure" v-bind="bound">Did not send</slot></template>
    </Thread>

    <Composer
      ref="composer"
      v-model="text"
      class="agent-panel__composer"
      :working="working"
      :placeholder="placeholder"
      :disabled="disabled"
      @submit="emit('submit', $event)"
    />
  </Panel>
</template>

<style scoped>
.agent-panel {
  /* What the conversation is kept clear of at the sides, how far it fades
     where it passes behind an edge, and how far the composer stands off the
     foot. */
  --inset: 20px;
  --fade: 20px;
  --lift: 12px;
  /* What is left between the last turn and the composer written over it. */
  --breath: 28px;

  position: relative;
  padding-block: 0;
  padding-inline: var(--inset);
}

/* Takes the whole panel and scrolls inside it. What it is clear of at the
   foot is the composer's own height. */
.agent-panel__thread {
  --clear: calc(var(--agent-panel-room) + var(--lift) * 2 + var(--breath));

  flex: 1;
  min-height: 0;
  padding-block-start: var(--fade);
  padding-block-end: var(--clear);
  mask-image: linear-gradient(
    to bottom,
    transparent 0,
    #000 var(--fade),
    #000 calc(100% - var(--clear)),
    transparent calc(100% - var(--clear) + var(--fade))
  );
}

.agent-panel__composer {
  position: absolute;
  inset-inline: var(--inset);
  inset-block-end: var(--lift);
  box-shadow: var(--numen-panel-shadow);
}
</style>
