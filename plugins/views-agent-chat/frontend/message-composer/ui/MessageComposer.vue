<script setup lang="ts">
import { computed, useTemplateRef } from 'vue'
import { Textarea } from './textarea'
import { Disc } from './disc'
import { AgentModelSelector } from './model-selector'
import type { AgentModelOption } from '../model/options'
import { ReasoningSelector } from './reasoning-selector'
import { COMPOSER_STATES, getComposerState, keyIntent, getMessage } from '../model/state'
import type { MessageComposerProps } from './MessageComposer.types'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<MessageComposerProps>(), {
  placeholder: 'Write a message',
  isAnswerPending: false,
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
  /** Sent. Carries what was written, with the whitespace around it gone. */
  (event: 'submit', text: string): void
  /** Give up on the answer on its way. */
  (event: 'stop'): void
  /** A model/agent was selected from the dropdown. */
  (event: 'select-model', option: AgentModelOption): void
  /** A depth of reasoning was chosen from the dropdown, by the name of the level. */
  (event: 'select-reasoning', id: string): void
  /** The list of models or of reasoning levels opened. */
  (event: 'open-choices'): void
  /** The link to the settings was pressed. */
  (event: 'open-settings'): void
}>()

defineExpose({ focus: (how?: Parameters<HTMLElement['focus']>[0]) => field.value?.focus(how) })

/* --------------------------------- State ---------------------------------- */
const field = useTemplateRef<InstanceType<typeof Textarea>>('field')

const state = computed(() => getComposerState(text.value, props.isAnswerPending))
const descriptor = computed(() => COMPOSER_STATES[state.value])

/** Nothing to say, or turned off. */
const barred = computed(() => props.isDisabled || props.isAgentMissing || !descriptor.value.canAct)

/* -------------------------------- Handlers -------------------------------- */
function onAct() {
  if (barred.value) return
  if (descriptor.value.action === 'stop') emit('stop')
  else emit('submit', getMessage(text.value))
}

function onKeydown(event: KeyboardEvent) {
  if (keyIntent(event) !== 'submit') return
  event.preventDefault()
  if (descriptor.value.action === 'send') onAct()
}

function onSelectModel(option: AgentModelOption) {
  emit('select-model', option)
}

function onSelectReasoning(id: string) {
  emit('select-reasoning', id)
}
</script>

<template>
  <div class="composer numen text-ink flex flex-col font-sans transition-colors">
    <!-- Multi-line expanding text input -->
    <p v-if="isAgentMissing" class="composer__empty text-hushed">
      {{ noAgentMessage }}
      <button type="button" class="composer__link cursor-pointer" @click="emit('open-settings')">
        {{ openSettingsLabel }}
      </button>
    </p>
    <div v-else class="composer__grow grid min-w-0">
      <div class="composer__mirror">{{ text }}&nbsp;</div>
      <div v-if="!text" class="composer__placeholder text-hushed" aria-hidden="true">
        {{ placeholder }}
      </div>
      <Textarea
        ref="field"
        v-model="text"
        rows="1"
        class="composer__field resize-none overflow-y-auto rounded-none border-0 bg-transparent p-0 placeholder:text-transparent"
        :placeholder="placeholder"
        :aria-label="placeholder"
        :disabled="isDisabled"
        @keydown="onKeydown"
      />
    </div>

    <!-- Integrated controls row: flat model selector pill on bottom-left, send/stop disc on bottom-right -->
    <div class="composer__toolbar flex min-w-0 items-center justify-between gap-2 pt-2">
      <div class="composer__selectors flex min-w-0 flex-1 items-center gap-1.5">
        <AgentModelSelector
          v-if="isAgentMissing || (options && options.length > 0)"
          :options="options"
          :selected-agent-id="selectedAgentId"
          :selected-model-id="selectedModelId"
          :empty-label="noAgentLabel"
          :is-disabled="isDisabled || isAnswerPending"
          @select="onSelectModel"
          @open="emit('open-choices')"
        />
        <ReasoningSelector
          v-if="reasoningLevels.length > 0"
          :levels="reasoningLevels"
          :selected-id="selectedReasoningId"
          :name="reasoningName"
          :is-disabled="isDisabled || isAnswerPending"
          @select="onSelectReasoning"
          @open="emit('open-choices')"
        />
      </div>

      <Disc
        class="composer__action shrink-0"
        :action="descriptor.action"
        :is-disabled="barred"
        :send-label="sendLabel"
        :stop-label="stopLabel"
        @press="onAct"
      />
    </div>
  </div>
</template>

<style scoped>
.composer {
  /* Lines of typing it grows to before it scrolls. */
  --lines: 10;
  --pad-top: 0.75rem;
  --pad-inline: 0.75rem;
  --pad-bottom: 0.5rem;
  --line-min: 1.75rem;

  position: relative;
  min-block-size: var(--numen-field-min);
  padding: var(--pad-top) var(--pad-inline) var(--pad-bottom);
  background-color: var(--numen-raised);
  border: 1px solid var(--numen-rule);
  border-radius: var(--numen-radius-panel);
  box-shadow: none;
  font-size: var(--numen-text-2);
}

.composer__empty {
  flex: 1;
  margin: 0;
  min-height: var(--line-min);
  line-height: var(--numen-line-height);
}

.composer__link {
  padding: 0;
  background: none;
  border: 0;
  color: inherit;
  font: inherit;
  text-decoration: underline dotted;
  --underline-offset: 0.2em;
  text-underline-offset: var(--underline-offset);
}

.composer__grow {
  flex: 1;
  min-height: var(--line-min);
}

.composer__grow > * {
  grid-area: 1 / 1;
  padding: 0;
  font: inherit;
  line-height: var(--numen-line-height);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.composer__mirror {
  --mirror-pad: 2rem;
  --line-unit: 1em;
  visibility: hidden;
  overflow: hidden;
  max-block-size: calc(
    (var(--lines) - 1) * var(--numen-line-height) * var(--line-unit) + var(--mirror-pad)
  );
}

.composer__placeholder {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  pointer-events: none;
}

.composer__field {
  scrollbar-width: none;
}

.composer__field:focus-visible {
  box-shadow: none;
}

.composer__field::-webkit-scrollbar {
  display: none;
}
</style>
