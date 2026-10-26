<script setup lang="ts">
/**
 * The view of the `conversation` capability, drawn into a panel of the plugin that mounts it.
 */
import { useTemplateRef } from 'vue'
import type { ConversationContext } from '@numen/plugin-sdk'
import { Agent } from '../agent'
import { WORDS as words } from '../words'
import { useConversationView } from './useConversationView'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ context: ConversationContext }>()

/* --------------------------------- State ---------------------------------- */
const view = useConversationView(props.context)
const agent = useTemplateRef<InstanceType<typeof Agent>>('agent')

// The panel it is mounted in may still be moving, so taking the keyboard scrolls nothing.
defineExpose({ focus: () => agent.value?.focus({ preventScroll: true }) })
</script>

<template>
  <Agent
    ref="agent"
    class="conversation"
    :model-value="view.question.value"
    :turns="view.turns.value"
    :is-working="view.isWorking.value"
    :placeholder="view.placeholder"
    :send-label="words.send"
    :stop-label="words.stop"
    @update:model-value="view.setQuestion"
    @submit="view.send"
    @stop="view.stop"
  >
    <template #silence>{{ view.getSilence() }}</template>
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
