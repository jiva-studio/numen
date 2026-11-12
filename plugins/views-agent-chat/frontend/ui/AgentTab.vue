<script setup lang="ts">
/* --------------------------------- Props ---------------------------------- */
import Agent from './Agent.vue'
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
</script>

<template>
  <Agent
    :model-value="props.state.draft.userQuestion.value"
    :turns="props.state.addresses.turns.value"
    :is-answer-pending="props.state.conversation.isAnswerPending.value"
    :placeholder="words.ask"
    :send-label="words.send"
    :stop-label="words.stop"
    @update:model-value="onUpdateQuestion"
    @submit="onSubmit"
    @stop="onStop"
    @open="onOpenTurn"
    @follow="onFollowLink"
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
