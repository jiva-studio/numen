<script setup lang="ts">
/**
 * An agent talked to: the conversation, and the field it is carried on with.
 *
 * The composer is written over the conversation, and the conversation is held
 * clear of however much room it takes, sitting at its foot when a question is
 * sent. It fills whatever it is put in, and says nothing about where that is.
 */
import { computed, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from 'vue'
import { Thread } from '@/features/thread'
import { MessageComposer } from './message-composer'
import type { Turn } from '@/features/thread'

withDefaults(
  defineProps<{
    turns: readonly Turn[]
    /** An answer is on its way. */
    working?: boolean
    placeholder?: string
    disabled?: boolean
    /** What the disc at the end of the field is called while it sends. */
    sendLabel?: string
    /** What it is called while it stops the answer on its way. */
    stopLabel?: string
  }>(),
  {
    working: false,
    placeholder: 'Write a message',
    disabled: false,
    sendLabel: 'Send',
    stopLabel: 'Stop',
  },
)

const text = defineModel<string>({ default: '' })

const emit = defineEmits<{
  (event: 'submit', text: string): void
  /** Give up on the answer on its way. */
  (event: 'stop'): void
  /** A turn the person pressed, which is one that says it opens something. */
  (event: 'open', turn: Turn): void
  /** A link inside a turn was pressed, with the turn it stands in. */
  (event: 'follow', turn: Turn, href: string, press: MouseEvent): void
}>()

const composer = useTemplateRef<InstanceType<typeof MessageComposer>>('composer')
const thread = useTemplateRef<InstanceType<typeof Thread>>('thread')

defineExpose({ focus: (how?: FocusOptions) => composer.value?.focus(how) })
const room = ref('0px')

/** Sent. The conversation takes up following its foot, where the question is. */
const onSubmit = (text: string) => {
  emit('submit', text)
  thread.value?.toFoot(true)
}

// A field grown taller carries the foot of the conversation up with it.
watch(room, () => thread.value?.toFoot(), { flush: 'post' })

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

const agentStyle = computed(() => ({ '--agent-room': room.value }))

onBeforeUnmount(() => watching?.disconnect())
</script>

<template>
  <div
    class="agent numen flex min-h-0 flex-col font-sans text-base text-ink"
    :style="agentStyle"
  >
    <Thread
      ref="thread"
      class="agent__thread"
      :turns="turns"
      @open="emit('open', $event)"
      @follow="
        (turn: Turn, href: string, press: MouseEvent) => emit('follow', turn, href, press)
      "
    >
      <template #silence><slot name="silence">Nothing said yet</slot></template>
      <template v-if="$slots.turn" #turn="bound"><slot name="turn" v-bind="bound" /></template>
      <template #failure="bound"><slot name="failure" v-bind="bound">Did not send</slot></template>
    </Thread>

    <MessageComposer
      ref="composer"
      v-model="text"
      class="agent__composer"
      :working="working"
      :placeholder="placeholder"
      :disabled="disabled"
      :send-label="sendLabel"
      :stop-label="stopLabel"
      @submit="onSubmit"
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
  --fade: 1.25rem;
  --lift: 0.75rem;
  /* What is left between the last turn and the composer written over it. */
  --breath: 1.75rem;

  position: relative;
  block-size: 100%;
  padding-inline: var(--inset);
}

/* Takes the whole of it and scrolls inside. The words run on to the composer's
   own edge and fade out over the band it stands in. The top is an edge of the
   window and is drawn to it. */
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
