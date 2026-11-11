<script setup lang="ts">
/**
 * A conversation with an agent: the questions, the lines about its work and its answers, and the field a
 * question is written in.
 */
import { computed, nextTick, ref, useTemplateRef, watch } from 'vue'
import { Menu, Prose, type MenuItem } from '@numen/ui'
import { MODEL_OPTIONS, type Agent } from '../model/agent'
import AgentCall from './AgentCall.vue'
import AgentFoot from './AgentFoot.vue'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ agent: Agent }>()

/* --------------------------------- State ---------------------------------- */
const thread = useTemplateRef<HTMLElement>('thread')
const menuAt = ref<{ x: number; y: number } | null>(null)
const draft = computed({
  get: () => props.agent.draft.value,
  set: (text: string) => props.agent.setDraft(text),
})

const items: readonly MenuItem[] = MODEL_OPTIONS.map((one) => ({
  id: one.modelId,
  text: `${one.agentTitle} · ${one.modelTitle}`,
}))

/* --------------------------------- Hooks ---------------------------------- */
watch(
  () => props.agent.turns.value.length,
  async () => {
    await nextTick()
    thread.value?.scrollTo({ top: thread.value.scrollHeight, behavior: 'smooth' })
  },
)

/* -------------------------------- Handlers -------------------------------- */
function onKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' || event.shiftKey) return
  event.preventDefault()
  props.agent.submit(props.agent.draft.value)
}

function onOpenModels(at: { x: number; y: number }) {
  menuAt.value = at
}

function onChooseModel(id: string) {
  props.agent.select(id)
  menuAt.value = null
}
</script>

<template>
  <div class="agent">
    <div ref="thread" class="agent__thread">
      <template v-for="turn in agent.turns.value" :key="turn.id">
        <p v-if="turn.voice === 'prompt'" class="agent__bubble">{{ turn.text }}</p>
        <AgentCall v-else-if="turn.voice === 'doing'" :turn="turn" />
        <Prose v-else class="agent__answer" :text="turn.text" />
      </template>
    </div>

    <div class="agent__scrim" />

    <div class="agent__composer">
      <textarea
        v-model="draft"
        class="agent__field"
        rows="1"
        placeholder="Ask about the vault"
        @keydown="onKeydown"
      />
      <AgentFoot :agent="agent" @open-models="onOpenModels" />
    </div>

    <Menu
      v-if="menuAt"
      :items="items"
      :at="menuAt"
      is-open
      @choose="onChooseModel"
      @dismiss="menuAt = null"
    />
  </div>
</template>

<style scoped>
.agent {
  --room: 7.5rem;

  position: relative;
  block-size: 100%;
  padding-inline: var(--numen-gutter);
  font-family: var(--numen-font-sans);
  font-size: var(--numen-reading-size);
  color: var(--numen-ink);
}

.agent__thread {
  display: flex;
  flex-direction: column;
  gap: var(--numen-turn-gap);
  block-size: 100%;
  padding-block: 1rem calc(var(--room) + 1.75rem);
  overflow-y: auto;
  scrollbar-width: none;
  user-select: text;
  -webkit-user-select: text;
}

.agent__thread::-webkit-scrollbar {
  display: none;
}

.agent__bubble {
  align-self: flex-end;
  max-inline-size: 80%;
  margin: 0;
  padding: 0.5rem 0.75rem;
  border-radius: var(--numen-radius-bubble);
  background: var(--numen-bubble-bg);
  color: var(--numen-bubble-fg);
  white-space: pre-wrap;
}

.agent__answer {
  color: var(--numen-answer-fg);
}

.agent__scrim {
  position: absolute;
  inset-inline: 0;
  inset-block-end: 0;
  block-size: calc(var(--room) + 2.75rem);
  background: linear-gradient(to bottom, transparent 0%, var(--numen-surface) 60%);
  pointer-events: none;
}

.agent__composer {
  position: absolute;
  inset-inline: var(--numen-gutter);
  inset-block-end: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 0.75rem 0.75rem 0.5rem;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius-panel);
  background: var(--numen-raised);
  font-size: var(--numen-text-2);
}

.agent__field {
  min-block-size: 1.75rem;
  max-block-size: 10rem;
  padding: 0;
  border: 0;
  outline: 0;
  resize: none;
  background: transparent;
  color: inherit;
  field-sizing: content;
  font: inherit;
  line-height: var(--numen-line-height);
  user-select: text;
  -webkit-user-select: text;
}
</style>
