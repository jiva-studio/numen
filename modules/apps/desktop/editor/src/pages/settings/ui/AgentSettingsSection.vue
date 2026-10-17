<script setup lang="ts">
/**
 * Settings for the AI agent and indexing model.
 */
import { computed } from 'vue'
import { NumberField, Select, Switch } from '@numen/ui'
import type { SelectChoice } from '@numen/ui'
import SettingRow from './setting-row/SettingRow.vue'
import type { SettingsTabState } from '../types'
import paths from '../paths.json'
import { createModelChoices } from '../lib/models'
import { createSettingFields } from '@/entities/settings'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: SettingsTabState }>()

/* --------------------------------- State ---------------------------------- */
const fields = createSettingFields(() => props.state.installation)

const WHOLE = Number.MAX_SAFE_INTEGER
const STEPS = { least: 1, most: WHOLE }

/** How many steps a program takes where the file says nothing. */
const DEFAULT_STEPS = 100

/** The program that answers: the one the file names, else the one an installation nobody configured runs on. */
const activeProgram = computed(() => {
  const named = props.state.installation.getSetting(paths.agent)
  if (typeof named === 'string') return named
  return fields.getModels(paths.agent).find((one) => one.isDefault)?.name ?? ''
})
const hasAgent = computed(() => activeProgram.value !== '')

const agentModelPath = computed(() => ['agent', activeProgram.value, 'model'] as const)
const agentStepsPath = computed(() => ['agent', activeProgram.value, 'max_steps'] as const)
const agentHooksPath = computed(() => ['agent', activeProgram.value, 'reads_hooks_and_skills'] as const)

const getChoices = (at: readonly string[]): readonly SelectChoice[] =>
  createModelChoices(fields, at, words)

/* -------------------------------- Handlers -------------------------------- */
function onIndexingModelChange(name: string) {
  fields.chooseModel(paths.indexingModel, name)
}

function onAgentUseChange(name: string) {
  fields.chooseModel(paths.agent, name)
}

function onAgentModelChange(name: string) {
  fields.chooseModel(agentModelPath.value, name)
}

function onAgentStepsChange(count: number | null) {
  if (count !== null) fields.setValue(agentStepsPath.value, count)
}

function onAgentToolsChange(value: boolean) {
  fields.setValue(paths.agentTools, value)
}

function onAgentHooksChange(value: boolean) {
  fields.setValue(agentHooksPath.value, value)
}
</script>

<template>
  <section class="settings__group" :aria-label="words.indexing">
    <h2 class="settings__heading">{{ words.indexing }}</h2>

    <SettingRow
      v-slot="{ labelledBy }"
      at="indexing"
      :name="words.indexingModel"
      :detail="words.indexingModelDetail"
    >
      <Select
        :model-value="fields.getString(paths.indexingModel)"
        :choices="getChoices(paths.indexingModel)"
        :name="words.indexingModel"
        :aria-labelledby="labelledBy"
        class="settings__choice"
        @update:model-value="onIndexingModelChange"
      />
    </SettingRow>
  </section>

  <section class="settings__group" :aria-label="words.agent">
    <h2 class="settings__heading">{{ words.agent }}</h2>

    <SettingRow
      v-slot="{ labelledBy }"
      at="agent"
      :name="words.agentUse"
      :detail="words.agentUseDetail"
    >
      <Select
        :model-value="activeProgram"
        :choices="getChoices(paths.agent)"
        :name="words.agentUse"
        :aria-labelledby="labelledBy"
        class="settings__choice"
        @update:model-value="onAgentUseChange"
      />
    </SettingRow>

    <SettingRow
      v-if="hasAgent"
      v-slot="{ labelledBy }"
      at="agent-model"
      :name="words.agentModel"
      :detail="words.agentModelDetail"
    >
      <Select
        :model-value="fields.getString(agentModelPath)"
        :choices="getChoices(agentModelPath)"
        :name="words.agentModel"
        :aria-labelledby="labelledBy"
        class="settings__choice"
        @update:model-value="onAgentModelChange"
      />
    </SettingRow>

    <SettingRow
      v-if="hasAgent"
      v-slot="{ labelledBy }"
      at="agent-steps"
      :name="words.agentSteps"
      :detail="words.agentStepsDetail"
    >
      <NumberField
        :model-value="fields.getNumber(agentStepsPath, DEFAULT_STEPS)"
        :min="STEPS.least"
        :max="STEPS.most"
        :step="1"
        :aria-labelledby="labelledBy"
        class="settings__number"
        @commit="onAgentStepsChange"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="agent-tools"
      :name="words.agentTools"
      :detail="words.agentToolsDetail"
    >
      <Switch
        :model-value="fields.isEnabled(paths.agentTools)"
        :aria-labelledby="labelledBy"
        @update:model-value="onAgentToolsChange"
      />
    </SettingRow>

    <SettingRow
      v-if="hasAgent"
      v-slot="{ labelledBy }"
      at="agent-hooks"
      :name="words.agentHooks"
      :detail="words.agentHooksDetail"
    >
      <Switch
        :model-value="fields.isEnabled(agentHooksPath)"
        :aria-labelledby="labelledBy"
        @update:model-value="onAgentHooksChange"
      />
    </SettingRow>
  </section>
</template>

<style scoped>
.settings__group {
  max-inline-size: var(--settings-measure, 54rem);
  margin-block-end: var(--settings-apart, 1.75rem);
  margin-inline: auto;
}

.settings__heading {
  margin: 0 0 var(--settings-near, 0.375rem);
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  font-weight: 600;
  letter-spacing: var(--numen-caps-tracking);
  text-transform: uppercase;
}

.settings__number {
  inline-size: var(--settings-value, 6rem);
}

.settings__choice {
  inline-size: var(--settings-choice, 18rem);
}
</style>
