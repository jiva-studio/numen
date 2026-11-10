<script setup lang="ts">
/** The row under the field of a question: the model it goes to, and the button that sends it. */
import { useTemplateRef } from 'vue'
import { ArrowUp, ChevronDown } from '@lucide/vue'
import { measureRect } from '@numen/ui'
import type { Agent } from '../model/agent'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ agent: Agent }>()

const emit = defineEmits<{ 'open-models': [at: { x: number; y: number }] }>()

/* --------------------------------- State ---------------------------------- */
const selector = useTemplateRef<HTMLElement>('selector')

/* -------------------------------- Handlers -------------------------------- */
function onOpenSelector() {
  const box = measureRect(selector.value)
  if (box) emit('open-models', { x: box.left, y: box.bottom })
}

function onSend() {
  props.agent.submit(props.agent.draft.value)
}
</script>

<template>
  <div class="agent__foot">
    <button ref="selector" type="button" class="agent__model" @click="onOpenSelector">
      {{ agent.selected.value.agentTitle }} · {{ agent.selected.value.modelTitle }}
      <ChevronDown class="agent__chevron" />
    </button>
    <button
      type="button"
      class="agent__send"
      :disabled="!agent.draft.value.trim() || agent.isWorking.value"
      aria-label="Send"
      @click="onSend"
    >
      <ArrowUp class="agent__arrow" />
    </button>
  </div>
</template>

<style scoped>
.agent__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.agent__model {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.35rem 0.75rem;
  border: 0;
  border-radius: var(--numen-radius-pill);
  background: var(--numen-bubble-bg);
  color: inherit;
  font: inherit;
  font-size: var(--numen-text-1);
  font-weight: 500;
  cursor: pointer;
}

.agent__chevron {
  inline-size: 0.875rem;
  block-size: 0.875rem;
  opacity: 0.6;
}

.agent__send {
  display: grid;
  place-items: center;
  inline-size: 1.75rem;
  block-size: 1.75rem;
  padding: 0;
  border: 0;
  border-radius: var(--numen-radius-pill);
  background: var(--numen-accent);
  color: var(--numen-accent-ink);
  cursor: pointer;
}

.agent__send:disabled {
  opacity: 0.45;
  cursor: default;
}

.agent__arrow {
  inline-size: 1rem;
  block-size: 1rem;
}
</style>
