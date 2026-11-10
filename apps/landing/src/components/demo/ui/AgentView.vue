<script setup lang="ts">
/**
 * A conversation with an agent: the questions, the lines about its work and its answers, and the field a
 * question is written in.
 */
import { computed, nextTick, ref, useTemplateRef, watch } from 'vue'
import { ArrowUp, ChevronDown } from '@lucide/vue'
import { Menu, Prose, type MenuItem } from '@numen/ui'
import { MODEL_OPTIONS, type Agent, type Turn } from '../model/agent'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{ agent: Agent }>()

const emit = defineEmits<{ open: [turn: Turn] }>()

/* --------------------------------- State ---------------------------------- */
const thread = useTemplateRef<HTMLElement>('thread')
const field = useTemplateRef<HTMLTextAreaElement>('field')
const selector = useTemplateRef<HTMLElement>('selector')
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
watch(props.agent.focusCount, () => field.value?.focus())

/* -------------------------------- Handlers -------------------------------- */
function onKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' || event.shiftKey) return
  event.preventDefault()
  props.agent.submit(props.agent.draft.value)
}

function onOpenSelector() {
  const box = selector.value?.getBoundingClientRect()
  if (box) menuAt.value = { x: box.left, y: box.bottom }
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
        <button v-else-if="turn.voice === 'doing'" type="button" class="agent__call" @click="emit('open', turn)">
          <span class="agent__mark" />
          <span class="agent__tool">{{ turn.text }}</span>
          <span class="agent__subject">{{ turn.subject }}</span>
          <span class="agent__aside">{{ turn.aside }}</span>
        </button>
        <Prose v-else class="agent__answer" :text="turn.text" />
      </template>
    </div>

    <div class="agent__scrim" />

    <div class="agent__composer">
      <textarea
        ref="field"
        v-model="draft"
        class="agent__field"
        rows="1"
        placeholder="Ask about the vault"
        @keydown="onKeydown"
      />
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
          @click="agent.submit(agent.draft.value)"
        >
          <ArrowUp class="agent__arrow" />
        </button>
      </div>
    </div>

    <Menu v-if="menuAt" :items="items" :at="menuAt" is-open @choose="onChooseModel" @dismiss="menuAt = null" />
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

.agent__call {
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
  inline-size: 100%;
  padding: 0;
  border: 0;
  background: none;
  color: var(--numen-hushed);
  font: inherit;
  text-align: start;
  cursor: pointer;
}

.agent__mark {
  flex: none;
  inline-size: 0.45em;
  block-size: 0.45em;
  border-radius: var(--numen-radius-pill);
  background: currentColor;
  opacity: 0.4;
}

.agent__tool {
  color: var(--numen-ink);
  opacity: 0.7;
  white-space: nowrap;
}

.agent__subject {
  min-inline-size: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent__aside {
  margin-inline-start: auto;
  white-space: nowrap;
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
