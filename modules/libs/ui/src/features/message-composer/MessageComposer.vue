<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
import { computed, useTemplateRef } from 'vue'
import { Textarea } from '@/shared/ui/textarea'
import { Disc } from './disc'
import { AgentModelSelector, type AgentModelOption } from './model-selector'
import { COMPOSER_STATES, composerState, keyIntent, getMessage } from './state'

const props = withDefaults(
  defineProps<{
    placeholder?: string
    /** An answer is being written; the disc stops it. */
    isWorking?: boolean
    disabled?: boolean
    /** What the disc is called while it sends. */
    sendLabel?: string
    /** What it is called while it stops. */
    stopLabel?: string
    options?: readonly AgentModelOption[]
    selectedAgentId?: string
    selectedModelId?: string
  }>(),
  {
    placeholder: 'Write a message',
    isWorking: false,
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
  /** Sent. Carries what was written, with the whitespace around it gone. */
  (event: 'submit', text: string): void
  /** Give up on the answer on its way. */
  (event: 'stop'): void
  /** A model/agent was selected from the dropdown. */
  (event: 'select-model', option: AgentModelOption): void
}>()

/* --------------------------------- State ---------------------------------- */
const field = useTemplateRef<InstanceType<typeof Textarea>>('field')

const state = computed(() => composerState(text.value, props.isWorking))
const descriptor = computed(() => COMPOSER_STATES[state.value])

/** Nothing to say, or turned off. */
const barred = computed(() => props.disabled || !descriptor.value.canAct)

/* -------------------------------- Handlers -------------------------------- */
function act() {
  if (barred.value) return
  if (descriptor.value.action === 'stop') emit('stop')
  else emit('submit', getMessage(text.value))
}

/** Enter sends, and Shift+Enter inserts a newline. While an answer is on its way it does nothing. */
function onKeydown(event: KeyboardEvent) {
  if (keyIntent(event) !== 'submit') return
  event.preventDefault()
  if (descriptor.value.action === 'send') act()
}

function onSelectModel(option: AgentModelOption) {
  emit('select-model', option)
}

defineExpose({ focus: (how?: FocusOptions) => field.value?.focus(how) })
</script>

<template>
  <div class="composer numen text-ink flex flex-col font-sans transition-colors">
    <!-- Multi-line expanding text input -->
    <div class="composer__grow grid min-w-0">
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
        :disabled="disabled"
        @keydown="onKeydown"
      />
    </div>

    <!-- Integrated controls row: flat model selector pill on bottom-left, send/stop disc on bottom-right -->
    <div class="composer__toolbar flex min-w-0 items-center justify-between gap-2 pt-2">
      <AgentModelSelector
        v-if="options && options.length > 0"
        :options="options"
        :selected-agent-id="selectedAgentId"
        :selected-model-id="selectedModelId"
        :disabled="disabled || isWorking"
        @select="onSelectModel"
      />
      <div v-else class="min-w-0 flex-1" />

      <Disc
        class="composer__action shrink-0"
        :action="descriptor.action"
        :disabled="barred"
        :send-label="sendLabel"
        :stop-label="stopLabel"
        @press="act"
      />
    </div>
  </div>
</template>

<style scoped>
.composer {
  /* Lines of typing it grows to before it scrolls. */
  --lines: 10;

  position: relative;
  min-block-size: var(--numen-field-min);
  padding: 0.75rem 0.75rem 0.5rem;
  background-color: var(--numen-raised);
  border: 1px solid var(--numen-rule);
  border-radius: var(--numen-radius-panel);
  box-shadow: none;
  font-size: var(--numen-text-2);
}

.composer__grow {
  flex: 1;
  min-height: 1.75rem;
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
  visibility: hidden;
  overflow: hidden;
  max-block-size: calc((var(--lines) - 1) * var(--numen-line-height) * 1em + 2rem);
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
