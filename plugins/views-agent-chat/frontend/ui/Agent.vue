<script setup lang="ts">
import { computed, useTemplateRef, watch } from 'vue'
import { useWholeViewport } from '@numen/ui'
import { Thread } from '../thread'
import { MessageComposer, type AgentModelOption } from '../message-composer'
import type { Turn } from '../types'
import type { AgentProps } from './Agent.types'

/* --------------------------------- Props ---------------------------------- */
withDefaults(defineProps<AgentProps>(), {
  isAnswerPending: false,
  placeholder: 'Write a message',
  isDisabled: false,
  sendLabel: 'Send',
  stopLabel: 'Stop',
  options: () => [],
  selectedAgentId: '',
  selectedModelId: '',
  reasoningLevels: () => [],
  selectedReasoningId: '',
  reasoningName: 'Reasoning',
  isAgentMissing: false,
  noAgentLabel: 'No agent',
  noAgentMessage: 'No agent is set up.',
  openSettingsLabel: 'Open settings',
})

const text = defineModel<string>({ default: '' })

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'submit', text: string): void
  /** Give up on the answer on its way. */
  (event: 'stop'): void
  /** A turn that the user activated, and that opens something. */
  (event: 'open', turn: Turn): void
  (event: 'follow', turn: Turn, href: string, press: MouseEvent): void
  /** A model/agent was selected from the dropdown. */
  (event: 'select-model', option: AgentModelOption): void
  /** A depth of reasoning was chosen from the dropdown, by the name of the level. */
  (event: 'select-reasoning', id: string): void
  /** The list of models or of reasoning levels opened. */
  (event: 'open-choices'): void
  (event: 'open-settings'): void
}>()

defineExpose({ focus: (how?: Parameters<HTMLElement['focus']>[0]) => composer.value?.focus(how) })

/* --------------------------------- State ---------------------------------- */
const composer = useTemplateRef<InstanceType<typeof MessageComposer>>('composer')
const thread = useTemplateRef<InstanceType<typeof Thread>>('thread')
const composerElement = computed(() => (composer.value?.$el as HTMLElement | undefined) ?? null)

// What the conversation has to clear is the whole field, its padding and its
// edge included.
const { viewport: whole } = useWholeViewport(composerElement)
const room = computed(() => `${whole.value.height}px`)

/* -------------------------------- Helpers --------------------------------- */
const agentStyle = computed(() => ({ '--agent-room': room.value }))

/* --------------------------------- Hooks ---------------------------------- */
// A field grown taller carries the foot of the conversation up with it.
watch(room, () => thread.value?.toFoot(), { flush: 'post' })

/* -------------------------------- Handlers -------------------------------- */
function onSubmit(text: string) {
  emit('submit', text)
  thread.value?.toFoot(true)
}

function onSelectModel(option: AgentModelOption) {
  emit('select-model', option)
}

function onSelectReasoning(id: string) {
  emit('select-reasoning', id)
}
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
      <template #silence><slot name="silence">No messages yet</slot></template>
      <template v-if="$slots.turn" #turn="bound"><slot name="turn" v-bind="bound" /></template>
      <template #failure="bound"><slot name="failure" v-bind="bound">Not sent</slot></template>
    </Thread>

    <div class="agent__scrim" aria-hidden="true" />

    <MessageComposer
      ref="composer"
      v-model="text"
      class="agent__composer"
      :is-answer-pending="isAnswerPending"
      :placeholder="placeholder"
      :is-disabled="isDisabled"
      :send-label="sendLabel"
      :stop-label="stopLabel"
      :options="options"
      :selected-agent-id="selectedAgentId"
      :selected-model-id="selectedModelId"
      :reasoning-levels="reasoningLevels"
      :selected-reasoning-id="selectedReasoningId"
      :reasoning-name="reasoningName"
      :is-agent-missing="isAgentMissing"
      :no-agent-label="noAgentLabel"
      :no-agent-message="noAgentMessage"
      :open-settings-label="openSettingsLabel"
      @submit="onSubmit"
      @stop="emit('stop')"
      @select-model="onSelectModel"
      @select-reasoning="onSelectReasoning"
      @open-choices="emit('open-choices')"
      @open-settings="emit('open-settings')"
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
  --gutter: 1.75rem;

  position: relative;
  block-size: 100%;
  padding-inline: var(--inset);
}

.agent__thread {
  --behind: calc(var(--agent-room) + var(--lift));
  --clear: calc(var(--behind) + 1rem);
  --thread-pad-start: 1rem;

  flex: 1;
  min-height: 0;
  padding-block-start: var(--thread-pad-start);
  padding-block-end: var(--clear);
}

.agent__scrim {
  --scrim-extra: 2rem;
  position: absolute;
  inset-inline: 0;
  inset-block-end: 0;
  block-size: calc(var(--agent-room) + var(--lift) + var(--scrim-extra));
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
