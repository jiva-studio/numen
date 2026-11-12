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
  props.state.models.selectModel(option)
}

function onSelectReasoning(id: string) {
  props.state.reasoning.selectReasoningLevel(id)
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
    :options="props.state.models.modelOptions.value"
    :selected-agent-id="props.state.models.selectedAgentId.value"
    :selected-model-id="props.state.models.selectedModelId.value"
    :reasoning-levels="props.state.reasoning.reasoningLevels.value"
    :selected-reasoning-id="props.state.reasoning.selectedReasoningId.value"
    :reasoning-name="words.reasoning"
    :is-agent-missing="props.state.agent.isAgentMissing.value"
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
    @open-settings="props.state.agent.openSettings()"
  >
    <template #silence>{{ props.state.agent.unreachable() || words.noTranscript }}</template>
    <template #failure="{ turn }">
      {{ turn.voice === 'prompt' ? words.unsent : words.stopped }}
    </template>
  </Agent>
</template>
