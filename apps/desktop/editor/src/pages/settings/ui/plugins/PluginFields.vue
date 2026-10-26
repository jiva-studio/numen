<script setup lang="ts">
/**
 * Declarative settings fields for a plugin: agent choices, switches,
 * number inputs, sliders, selects, and text fields.
 */
import { computed } from 'vue'
import { NumberField, Select, Slider, Switch } from '@numen/ui'
import SettingRow from '../setting-row/SettingRow.vue'
import { AgentChoice } from '../agent-choice'
import type { SettingsTabState } from '../../types'
import {
  createSettingFields,
  type CompletionChoice,
  type PluginInfo,
  type PluginSettingInfo,
} from '@/entities/settings'
import {
  DEFAULT_AGENT_AT,
  createAgentModelChoices,
  createPluginAgentChoices,
  createProgramModelChoices,
  getProgramModelsAt,
  findAgent,
  getAgents,
  readChoice,
} from '../../lib/agents'
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: SettingsTabState
  plugin: PluginInfo
}>()

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)
const fields = createSettingFields(() => props.state.installation)

const agents = computed(() => getAgents(installation.value))
const defaultAgent = computed(() => fields.getString(DEFAULT_AGENT_AT))
const agentSettings = computed(() => props.plugin.settings.filter((one) => one.type === 'agent'))
const otherSettings = computed(() => props.plugin.settings.filter((one) => one.type !== 'agent'))

/* -------------------------------- Handlers -------------------------------- */
function onSettingChange(key: string, value: unknown): void {
  fields.setValue(['plugins', props.plugin.id, key], value)
}

function onAgentChoose(key: string, choice: CompletionChoice): void {
  fields.setValue(['plugins', props.plugin.id, key], choice)
}

/* -------------------------------- Helpers --------------------------------- */
function getChoice(key: string): CompletionChoice {
  return readChoice(installation.value.getSetting(['plugins', props.plugin.id, key]))
}

function getAgentChoices(choice: CompletionChoice) {
  return createPluginAgentChoices(
    agents.value,
    { defaultAgent: defaultAgent.value, chosen: choice.agent },
    words,
  )
}

function getModelChoices(choice: CompletionChoice) {
  if (choice.agent === '') return null
  return createAgentModelChoices(findAgent(agents.value, choice.agent), choice.model, words)
}

function getProgramChoices(setting: PluginSettingInfo) {
  if (!setting.modelsOf) return []
  const models = installation.value.getModels(getProgramModelsAt(setting.modelsOf))
  return createProgramModelChoices(models, getStringSetting(setting.key, setting.default))
}

function getSetting(key: string, fallback: unknown): unknown {
  const val = installation.value.getSetting(['plugins', props.plugin.id, key])
  return val !== undefined ? val : fallback
}

function getBooleanSetting(key: string, fallback: unknown): boolean {
  const val = getSetting(key, fallback)
  return typeof val === 'boolean' ? val : Boolean(fallback)
}

function getNumberSetting(key: string, fallback: unknown): number {
  const val = getSetting(key, fallback)
  return typeof val === 'number' ? val : Number(fallback ?? 0)
}

function getStringSetting(key: string, fallback: unknown): string {
  const val = getSetting(key, fallback)
  return typeof val === 'string' ? val : String(fallback ?? '')
}
</script>

<template>
  <div class="plugin-fields">
    <AgentChoice
      v-for="setting in agentSettings"
      :key="setting.key"
      :at="`${props.plugin.id}-${setting.key}`"
      :agent-name="setting.title"
      :agent-detail="setting.description"
      :model-name="words.pluginModel"
      :model-detail="words.defaultModelDetail"
      :agents="getAgentChoices(getChoice(setting.key))"
      :models="getModelChoices(getChoice(setting.key))"
      :choice="getChoice(setting.key)"
      @choose="(choice) => onAgentChoose(setting.key, choice)"
    />

    <SettingRow
      v-for="setting in otherSettings"
      :key="setting.key"
      v-slot="{ labelledBy }"
      :at="`${props.plugin.id}-${setting.key}`"
      :name="setting.title"
      :detail="setting.description"
    >
      <Switch
        v-if="setting.type === 'boolean'"
        :model-value="getBooleanSetting(setting.key, setting.default)"
        :aria-labelledby="labelledBy"
        @update:model-value="(val) => onSettingChange(setting.key, val)"
      />

      <NumberField
        v-else-if="setting.type === 'number'"
        :model-value="getNumberSetting(setting.key, setting.default)"
        :min="setting.min ?? -999999"
        :max="setting.max ?? 999999"
        :step="setting.step ?? 1"
        :aria-labelledby="labelledBy"
        class="settings__number"
        @commit="(val) => val !== null && onSettingChange(setting.key, val)"
      />

      <div v-else-if="setting.type === 'range'" class="settings__slider-container">
        <Slider
          :model-value="getNumberSetting(setting.key, setting.default)"
          :min="setting.min ?? 0"
          :max="setting.max ?? 100"
          :step="setting.step ?? 1"
          :aria-labelledby="labelledBy"
          class="settings__slider"
          @update:model-value="(val) => onSettingChange(setting.key, val)"
          @commit="(val) => onSettingChange(setting.key, val)"
        />
        <span class="settings__slider-value">
          {{ getNumberSetting(setting.key, setting.default) }}
        </span>
      </div>

      <Select
        v-else-if="setting.type === 'enum' && setting.options"
        :model-value="getStringSetting(setting.key, setting.default)"
        :choices="setting.options.map((opt) => ({ id: opt, text: opt }))"
        :name="setting.title"
        :aria-labelledby="labelledBy"
        class="settings__choice"
        @update:model-value="(val) => onSettingChange(setting.key, val)"
      />

      <Select
        v-else-if="getProgramChoices(setting).length > 0"
        :model-value="getStringSetting(setting.key, setting.default)"
        :choices="getProgramChoices(setting)"
        :name="setting.title"
        :aria-labelledby="labelledBy"
        is-searchable
        :search-name="words.searchModels"
        class="settings__choice"
        @update:model-value="(val) => onSettingChange(setting.key, val)"
      />

      <input
        v-else
        type="text"
        :value="getStringSetting(setting.key, setting.default)"
        :aria-labelledby="labelledBy"
        class="settings__text-input"
        @change="(e) => onSettingChange(setting.key, (e.target as HTMLInputElement).value)"
      />
    </SettingRow>
  </div>
</template>

<style scoped>
.plugin-fields {
  display: flex;
  flex-direction: column;
}

.settings__number {
  inline-size: var(--settings-value, 6rem);
}

.settings__slider-container {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  inline-size: var(--settings-choice, 16rem);
}

.settings__slider-container .settings__slider {
  flex: 1 1 auto;
  inline-size: auto;
}

.settings__slider-value {
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
  font-variant-numeric: tabular-nums;
  min-inline-size: 2.5rem;
  text-align: end;
}

.settings__choice {
  inline-size: var(--settings-choice, 16rem);
}

.settings__text-input {
  box-sizing: border-box;
  inline-size: var(--settings-choice, 16rem);
  padding: 0.25rem 0.5rem;
  background-color: var(--numen-surface, transparent);
  border: var(--numen-stroke, 1px) solid var(--numen-rule);
  border-radius: var(--numen-radius-small, 4px);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
}

.settings__text-input:focus {
  outline: var(--numen-stroke, 1px) solid var(--numen-accent, currentColor);
}
</style>
