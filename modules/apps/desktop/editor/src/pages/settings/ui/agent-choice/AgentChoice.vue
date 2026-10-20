<script setup lang="ts">
/**
 * The agent a plugin's questions to a model go to, and the model it answers
 * with, each on a row of its own. Choosing another agent leaves the model to it.
 */
import { Select } from '@numen/ui'
import type { SelectChoice } from '@numen/ui'
import SettingRow from '../setting-row/SettingRow.vue'
import type { CompletionChoice } from '@/entities/settings'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  /** What the two rows are about, which their names are addressed by. */
  at: string
  agentName: string
  agentDetail: string
  modelName: string
  modelDetail: string
  agents: readonly SelectChoice[]
  /** The models of the agent in force. None hides the row. */
  models: readonly SelectChoice[] | null
  choice: CompletionChoice
}>()

const emit = defineEmits<{
  choose: [choice: CompletionChoice]
}>()

/* --------------------------------- State ---------------------------------- */
/** From this many models on, the list carries a line to search it by. */
const SEARCH_FROM = 8

/* -------------------------------- Handlers -------------------------------- */
function onAgentChange(agent: string) {
  if (agent === props.choice.agent) return
  emit('choose', { agent, model: '' })
}

function onModelChange(model: string) {
  emit('choose', { agent: props.choice.agent, model })
}
</script>

<template>
  <SettingRow v-slot="{ labelledBy }" :at="`${at}-agent`" :name="agentName" :detail="agentDetail">
    <Select
      :model-value="choice.agent"
      :choices="agents"
      :name="agentName"
      :aria-labelledby="labelledBy"
      class="settings__choice"
      @update:model-value="onAgentChange"
    />
  </SettingRow>

  <SettingRow
    v-if="models"
    v-slot="{ labelledBy }"
    :at="`${at}-model`"
    :name="modelName"
    :detail="modelDetail"
  >
    <Select
      :model-value="choice.model"
      :choices="models"
      :name="modelName"
      :searchable="models.length >= SEARCH_FROM"
      search-name="Search models"
      :aria-labelledby="labelledBy"
      class="settings__choice"
      @update:model-value="onModelChange"
    />
  </SettingRow>
</template>

<style scoped>
.settings__choice {
  inline-size: var(--settings-choice, 18rem);
}
</style>
