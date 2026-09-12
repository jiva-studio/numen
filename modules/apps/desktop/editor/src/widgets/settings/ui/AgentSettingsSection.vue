<script setup lang="ts">
/**
 * Settings for the AI agent and indexing model.
 */
import { computed } from 'vue'
import { NumberField, Select, Switch } from '@numen/ui'
import type { SelectChoice } from '@numen/ui'
import SettingRow from './setting-row/SettingRow.vue'
import type { SettingsTabState } from '../model/useSettingsTab'
import AT from '../paths.json'
import { choicesFor } from '../models'
import { write } from '@/entities/settings/write'
import { WORDS as words } from '../words'

// --- Props & Emits ---
const props = defineProps<{ state: SettingsTabState }>()

// --- State ---
const installation = computed(() => props.state.installation)

const WHOLE = Number.MAX_SAFE_INTEGER
const STEPS = { least: 1, most: WHOLE }

// --- Handlers ---
function onModelChange(at: readonly string[], name: string) {
  const model = installation.value.models(at).find((one) => one.name === name)
  if (model) installation.value.writes(model.writes)
  else setSetting(at, name)
}

function onSettingChange(at: readonly string[], value: unknown) {
  setSetting(at, value)
}

// --- Helpers ---
function getSettingString(at: readonly string[]): string {
  const value = installation.value.setting(at)
  return typeof value === 'string' ? value : ''
}
const said = getSettingString

function isSettingEnabled(at: readonly string[]): boolean {
  return installation.value.setting(at) === true
}
const on = isSettingEnabled

function getSettingNumber(at: readonly string[]): number | null {
  const value = installation.value.setting(at)
  return typeof value === 'number' ? value : null
}
const counted = getSettingNumber

function getModels(at: readonly string[]): readonly SelectChoice[] {
  return choicesFor(installation.value.models(at), getSettingString(at), words)
}
const models = getModels

function setSetting(at: readonly string[], value: unknown): void {
  installation.value.writes([{ at, value: write(value) }])
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
        :model-value="said(AT.indexingModel)"
        :choices="models(AT.indexingModel)"
        :name="words.indexingModel"
        :aria-labelledby="labelledBy"
        class="settings__choice"
        @update:model-value="(name: string) => onModelChange(AT.indexingModel, name)"
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
        :model-value="said(AT.agent)"
        :choices="models(AT.agent)"
        :name="words.agentUse"
        :aria-labelledby="labelledBy"
        class="settings__choice"
        @update:model-value="(name: string) => onModelChange(AT.agent, name)"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="agent-model"
      :name="words.agentModel"
      :detail="words.agentModelDetail"
    >
      <Select
        :model-value="said(AT.agentModel)"
        :choices="models(AT.agentModel)"
        :name="words.agentModel"
        :aria-labelledby="labelledBy"
        class="settings__choice"
        @update:model-value="(name: string) => onModelChange(AT.agentModel, name)"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="agent-steps"
      :name="words.agentSteps"
      :detail="words.agentStepsDetail"
    >
      <NumberField
        :model-value="counted(AT.agentSteps)"
        :min="STEPS.least"
        :max="STEPS.most"
        :step="1"
        :aria-labelledby="labelledBy"
        class="settings__number"
        @settles="(count: number | null) => count !== null && onSettingChange(AT.agentSteps, count)"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="agent-tools"
      :name="words.agentTools"
      :detail="words.agentToolsDetail"
    >
      <Switch
        :model-value="on(AT.agentTools)"
        :aria-labelledby="labelledBy"
        @update:model-value="(kept: boolean) => onSettingChange(AT.agentTools, kept)"
      />
    </SettingRow>

    <SettingRow
      v-slot="{ labelledBy }"
      at="agent-hooks"
      :name="words.agentHooks"
      :detail="words.agentHooksDetail"
    >
      <Switch
        :model-value="on(AT.agentHooks)"
        :aria-labelledby="labelledBy"
        @update:model-value="(kept: boolean) => onSettingChange(AT.agentHooks, kept)"
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
