<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
import Agent from './Agent.vue'
import type { AgentModelOption } from '../message-composer'
import type { Turn } from '../types'
import { WORDS as words } from '../words'
import type { AgentTabState } from '../model/useAgentConversation'

const props = defineProps<{ state: AgentTabState }>()

/* -------------------------------- Handlers -------------------------------- */
function onUpdateQuestion(text: string) {
  props.state.draft.setQuestion(text)
}

function onSubmit(text: string) {
  props.state.draft.send(text)
}

function onStop() {
  props.state.conversation.stop()
}

function onOpenTurn(turn: Turn) {
  props.state.draft.openTurnSource(turn)
}

function onFollowLink(turn: Turn, href: string, press: MouseEvent) {
  props.state.addresses.followLink(turn, href, press)
}

function onSelectModel(option: AgentModelOption) {
  props.state.agents?.selectModel(option)
}

function onSelectReasoning(id: string) {
  props.state.agents?.selectReasoningLevel(id)
}

function onOpenChoices() {
  props.state.agents?.listChoices().catch(() => undefined)
}
</script>

<template>
  <Agent
    :model-value="props.state.draft.userQuestion.value"
    :turns="props.state.addresses.turns.value"
    :is-answer-pending="props.state.conversation.isAnswerPending.value"
    :placeholder="words.ask"
    :send-label="words.send"
    :stop-label="words.stop"
    :options="props.state.agents?.offered.options.value ?? []"
    :selected-agent-id="props.state.agents?.selected.agentId.value ?? ''"
    :selected-model-id="props.state.agents?.selected.modelId.value ?? ''"
    :reasoning-levels="props.state.agents?.offered.reasoningLevels.value ?? []"
    :selected-reasoning-id="props.state.agents?.selected.reasoningId.value ?? ''"
    :reasoning-name="words.reasoning"
    :is-agent-missing="props.state.agents?.offered.isAgentMissing.value ?? false"
    :no-agent-label="words.noAgent"
    :no-agent-message="words.noAgentMessage"
    :open-settings-label="words.openSettings"
    @update:model-value="onUpdateQuestion"
    @submit="onSubmit"
    @stop="onStop"
    @open="onOpenTurn"
    @follow="onFollowLink"
    @select-model="onSelectModel"
    @select-reasoning="onSelectReasoning"
    @open-choices="onOpenChoices"
    @open-settings="props.state.openSettings()"
  >
    <template #silence>{{ words.noTranscript }}</template>
    <template #failure="{ turn }">
      <button
        v-if="turn.offersSettings"
        type="button"
        class="agent-tab__settings cursor-pointer"
        @click="props.state.openSettings()"
      >
        {{ words.openSettings }}
      </button>
      <template v-else>{{ turn.voice === 'prompt' ? words.unsent : words.stopped }}</template>
    </template>
  </Agent>
</template>

<style scoped>
.agent-tab__settings {
  --underline-offset: 0.2em;
  padding: 0;
  background: none;
  border: 0;
  color: inherit;
  font: inherit;
  text-decoration: underline dotted;
  text-underline-offset: var(--underline-offset);
}
</style>
