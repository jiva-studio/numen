<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
import { Agent } from '../agent'
import type { AgentModelOption } from '../message-composer'
import type { Turn } from '../thread'
import { WORDS as words } from '../words'
import type { AgentTabState } from '../model/useAgentConversation'

const props = defineProps<{ state: AgentTabState }>()

/* -------------------------------- Handlers -------------------------------- */
function onUpdateQuestion(text: string) {
  props.state.setQuestion(text)
}

function onSubmit(text: string) {
  props.state.send(text)
}

function onStop() {
  props.state.stop()
}

function onOpenTurn(turn: Turn) {
  props.state.openTurnSource(turn)
}

function onFollowLink(turn: Turn, href: string, press: MouseEvent) {
  props.state.followLink(turn, href, press)
}

function onSelectModel(option: AgentModelOption) {
  props.state.selectModel(option)
}

function onSelectReasoning(id: string) {
  props.state.selectReasoning(id)
}
</script>

<template>
  <Agent
    :model-value="props.state.userQuestion.value"
    :turns="props.state.turns.value"
    :is-working="props.state.isWorking.value"
    :placeholder="words.ask"
    :send-label="words.send"
    :stop-label="words.stop"
    :options="props.state.modelOptions.value"
    :selected-agent-id="props.state.selectedAgentId.value"
    :selected-model-id="props.state.selectedModelId.value"
    :reasoning-levels="props.state.reasoningLevels.value"
    :selected-reasoning-id="props.state.selectedReasoningId.value"
    :reasoning-name="words.reasoning"
    @update:model-value="onUpdateQuestion"
    @submit="onSubmit"
    @stop="onStop"
    @open="onOpenTurn"
    @follow="onFollowLink"
    @select-model="onSelectModel"
    @select-reasoning="onSelectReasoning"
  >
    <template #silence>{{ props.state.unreachable() || words.nothingSaid }}</template>
    <template #failure="{ turn }">
      {{ turn.voice === 'prompt' ? words.unsent : words.stopped }}
    </template>
  </Agent>
</template>
