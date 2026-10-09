<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
import { Agent, type Turn, type AgentModelOption } from '@numen/ui'
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
    @update:model-value="onUpdateQuestion"
    @submit="onSubmit"
    @stop="onStop"
    @open="onOpenTurn"
    @follow="onFollowLink"
    @select-model="onSelectModel"
  >
    <template #silence>{{ props.state.unreachable() || words.nothingSaid }}</template>
    <template #failure="{ turn }">
      {{ turn.voice === 'asked' ? words.unsent : words.stopped }}
    </template>
  </Agent>
</template>
