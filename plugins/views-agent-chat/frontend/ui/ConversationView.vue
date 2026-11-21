<script setup lang="ts">
/**
 * The view of the `conversation` capability, drawn into a panel of the plugin that mounts it.
 */
import { useTemplateRef } from 'vue'
import type { ConversationContext } from '@numen/plugin-sdk'
import Agent from './Agent.vue'
import { WORDS as words } from '../words'
import { useConversationView } from '../model/useConversationView'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ context: ConversationContext }>()

// The panel it is mounted in may still be moving, so taking the keyboard scrolls nothing.
defineExpose({ focus: () => agent.value?.focus({ preventScroll: true }) })

/* --------------------------------- State ---------------------------------- */
const view = useConversationView(props.context)
const agent = useTemplateRef<InstanceType<typeof Agent>>('agent')

const onOpenChoices = () => {
  view.agents.listChoices().catch(() => undefined)
}
</script>

<template>
  <Agent
    ref="agent"
    class="conversation"
    :model-value="view.question.value"
    :turns="view.turns.value"
    :is-answer-pending="view.isAnswerPending.value"
    :placeholder="view.placeholder"
    :can-submit="view.agents.selected.canSubmit.value"
    :options="view.agents.offered.options.value"
    :selected-agent-id="view.agents.selected.agentId.value"
    :selected-model-id="view.agents.selected.modelId.value"
    :reasoning-levels="view.agents.offered.reasoningLevels.value"
    :selected-reasoning-id="view.agents.selected.reasoningId.value"
    :reasoning-name="words.reasoning"
    :pending-agent-ids="view.agents.offered.pendingAgentIds.value"
    :loading-label="words.loading"
    :failed-agent-ids="view.agents.offered.failedAgentIds.value"
    :failed-label="words.modelsFailed"
    :send-label="words.send"
    :stop-label="words.stop"
    @update:model-value="view.setQuestion"
    @submit="view.send"
    @stop="view.stop"
    @select-model="view.agents.selectModel"
    @select-reasoning="view.agents.selectReasoningLevel"
    @open-choices="onOpenChoices"
  >
    <template #silence>{{ words.noTranscript }}</template>
    <template #failure="{ turn }">
      {{ turn.voice === 'prompt' ? words.unsent : words.stopped }}
    </template>
  </Agent>
</template>

<style scoped>
.conversation {
  flex: 1;
  min-block-size: 0;
}
</style>
