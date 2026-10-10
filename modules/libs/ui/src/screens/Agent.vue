<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
import { computed, onBeforeUnmount, onMounted, ref, useTemplateRef, watch } from 'vue'
import { browserViewport } from '@/shared/lib/viewport'
import { Thread } from '@/features/thread'
import { MessageComposer, type AgentModelOption } from './message-composer'
import type { Turn } from '@/features/thread'

withDefaults(
  defineProps<{
    turns: readonly Turn[]
    /** An answer is on its way. */
    isWorking?: boolean
    placeholder?: string
    disabled?: boolean
    /** What the disc at the end of the field is called while it sends. */
    sendLabel?: string
    /** What it is called while it stops the answer on its way. */
    stopLabel?: string
    options?: readonly AgentModelOption[]
    selectedAgentId?: string
    selectedModelId?: string
  }>(),
  {
    isWorking: false,
    placeholder: 'Write a message',
    disabled: false,
    sendLabel: 'Send',
    stopLabel: 'Stop',
    options: () => [],
    selectedAgentId: '',
    selectedModelId: '',
  },
)

const text = defineModel<string>({ default: '' })

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'submit', text: string): void
  /** Give up on the answer on its way. */
  (event: 'stop'): void
  /** A turn the person pressed, which is one that says it opens something. */
  (event: 'open', turn: Turn): void
  /** A link inside a turn was pressed, with the turn it stands in. */
  (event: 'follow', turn: Turn, href: string, press: MouseEvent): void
  /** A model/agent was selected from the dropdown. */
  (event: 'select-model', option: AgentModelOption): void
}>()

/* --------------------------------- State ---------------------------------- */
const composer = useTemplateRef<InstanceType<typeof MessageComposer>>('composer')
const thread = useTemplateRef<InstanceType<typeof Thread>>('thread')
const room = ref('0px')

defineExpose({ focus: (how?: FocusOptions) => composer.value?.focus(how) })

/* --------------------------------- Hooks ---------------------------------- */
/** What stops the watching, held from the moment it begins. */
let stopWatching: (() => void) | null = null

onMounted(() => {
  const element = composer.value?.$el as HTMLElement | undefined
  if (!element) return
  // What the conversation has to clear is the whole field, its padding and its
  // edge included.
  stopWatching = browserViewport.watchWhole(element, (size) => {
    room.value = `${size.height}px`
  })
})

// A field grown taller carries the foot of the conversation up with it.
watch(room, () => thread.value?.toFoot(), { flush: 'post' })

onBeforeUnmount(() => stopWatching?.())

/* -------------------------------- Handlers -------------------------------- */
/** Sent. The conversation takes up following its foot, where the question is. */
function onSubmit(text: string) {
  emit('submit', text)
  thread.value?.toFoot(true)
}

function onSelectModel(option: AgentModelOption) {
  emit('select-model', option)
}

/* -------------------------------- Helpers --------------------------------- */
const agentStyle = computed(() => ({ '--agent-room': room.value }))
</script>

<template>
  <div class="agent numen text-ink flex min-h-0 flex-col font-sans text-base" :style="agentStyle">
    <Thread
      ref="thread"
      class="agent__thread"
      :turns="turns"
      @open="emit('open', $event)"
      @follow="(turn: Turn, href: string, press: MouseEvent) => emit('follow', turn, href, press)"
    >
      <template #silence><slot name="silence">Nothing said yet</slot></template>
      <template v-if="$slots.turn" #turn="bound"><slot name="turn" v-bind="bound" /></template>
      <template #failure="bound"><slot name="failure" v-bind="bound">Did not send</slot></template>
    </Thread>

    <div class="agent__scrim" aria-hidden="true" />

    <MessageComposer
      ref="composer"
      v-model="text"
      class="agent__composer"
      :is-working="isWorking"
      :placeholder="placeholder"
      :disabled="disabled"
      :send-label="sendLabel"
      :stop-label="stopLabel"
      :options="options"
      :selected-agent-id="selectedAgentId"
      :selected-model-id="selectedModelId"
      @submit="onSubmit"
      @stop="emit('stop')"
      @select-model="onSelectModel"
    >
      <template v-if="$slots.selector" #selector><slot name="selector" /></template>
      <template v-if="$slots.glyph" #glyph><slot name="glyph" /></template>
    </MessageComposer>
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

.agent__thread {
  --behind: calc(var(--agent-room) + var(--lift));
  --clear: calc(var(--behind) + 1rem);

  flex: 1;
  min-height: 0;
  padding-block-start: 1rem;
  padding-block-end: var(--clear);
}

.agent__scrim {
  position: absolute;
  inset-inline: 0;
  inset-block-end: 0;
  block-size: calc(var(--agent-room) + var(--lift) + 2rem);
  pointer-events: none;
  background: linear-gradient(
    to bottom,
    transparent 0%,
    var(--numen-surface) 60%,
    var(--numen-surface) 100%
  );
  z-index: 1;
}

.agent__composer {
  position: absolute;
  inset-inline: var(--inset);
  inset-block-end: var(--lift);
  z-index: 2;
}
</style>
