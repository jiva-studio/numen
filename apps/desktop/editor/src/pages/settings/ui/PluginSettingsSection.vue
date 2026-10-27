<script setup lang="ts">
/**
 * Dynamic settings section rendered for active plugins with declarative schemas.
 */
import { computed } from 'vue'
import { AGENT_SETTING_TYPE } from '@numen/plugin-sdk'
import { NumberField, Select, Slider, Switch } from '@numen/ui'
import SettingRow from './setting-row/SettingRow.vue'
import { AgentChoice } from './agent-choice'
import type { SettingsTabState } from '../types'
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
} from '../lib/agents'
import { WORDS as words } from '../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: SettingsTabState }>()

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)
const fields = createSettingFields(() => props.state.installation)

const plugins = computed<readonly PluginInfo[]>(() => {
  const list = installation.value.plugins?.value ?? []
  return list.filter((one) => one.settings.length > 0)
})

const agents = computed(() => getAgents(installation.value))
const defaultAgent = computed(() => fields.getString(DEFAULT_AGENT_AT))

/* -------------------------------- Handlers -------------------------------- */
function onSettingChange(pluginId: string, key: string, value: unknown): void {
  fields.setValue(['plugins', pluginId, key], value)
}

function onAgentChoose(pluginId: string, key: string, choice: CompletionChoice): void {
  fields.setValue(['plugins', pluginId, key], choice)
}

/* -------------------------------- Helpers --------------------------------- */
/** What the agent setting of a plugin holds. */
function getChoice(pluginId: string, key: string): CompletionChoice {
  return readChoice(installation.value.getSetting(['plugins', pluginId, key]))
}

/** The agents the setting can name, the default first. */
const getAgentChoices = (choice: CompletionChoice) =>
  createPluginAgentChoices(
    agents.value,
    { defaultAgent: defaultAgent.value, chosen: choice.agent },
    words,
  )

/** The models of the agent the setting names, and none while it names the default. */
const getModelChoices = (choice: CompletionChoice) => {
  if (choice.agent === '') return null
  return createAgentModelChoices(findAgent(agents.value, choice.agent), choice.model, words)
}

/** The setting of a plugin that chooses its agent, drawn first, and the rest. */
function getAgentSettings(plugin: PluginInfo): readonly PluginSettingInfo[] {
  return plugin.settings.filter((one) => one.type === AGENT_SETTING_TYPE)
}

function getOtherSettings(plugin: PluginInfo): readonly PluginSettingInfo[] {
  return plugin.settings.filter((one) => one.type !== AGENT_SETTING_TYPE)
}

/** The models a string setting is chosen from, and none where it is typed or the program lists none. */
function getProgramChoices(plugin: PluginInfo, setting: PluginSettingInfo) {
  if (!setting.modelsOf) return []
  const models = installation.value.getModels(getProgramModelsAt(setting.modelsOf))
  return createProgramModelChoices(
    models,
    getStringSetting(plugin.id, setting.key, setting.default),
  )
}

function getSetting(pluginId: string, key: string, fallback: unknown): unknown {
  const val = installation.value.getSetting(['plugins', pluginId, key])
  return val !== undefined ? val : fallback
}

function getBooleanSetting(pluginId: string, key: string, fallback: unknown): boolean {
  const val = getSetting(pluginId, key, fallback)
  return typeof val === 'boolean' ? val : Boolean(fallback)
}

function getNumberSetting(pluginId: string, key: string, fallback: unknown): number {
  const val = getSetting(pluginId, key, fallback)
  return typeof val === 'number' ? val : Number(fallback ?? 0)
}

function getStringSetting(pluginId: string, key: string, fallback: unknown): string {
  const val = getSetting(pluginId, key, fallback)
  return typeof val === 'string' ? val : String(fallback ?? '')
}
</script>

<template>
  <div v-if="plugins.length > 0" class="settings__plugins">
    <section
      v-for="plugin in plugins"
      :key="plugin.id"
      class="settings__group"
      :aria-label="plugin.name"
    >
      <h2 class="settings__heading">{{ plugin.name }}</h2>

      <AgentChoice
        v-for="setting in getAgentSettings(plugin)"
        :key="setting.key"
        :at="`${plugin.id}-${setting.key}`"
        :agent-name="setting.title"
        :agent-detail="setting.description"
        :model-name="words.pluginModel"
        :model-detail="words.defaultModelDetail"
        :agents="getAgentChoices(getChoice(plugin.id, setting.key))"
        :models="getModelChoices(getChoice(plugin.id, setting.key))"
        :choice="getChoice(plugin.id, setting.key)"
        @choose="(choice) => onAgentChoose(plugin.id, setting.key, choice)"
      />
      <SettingRow
        v-for="setting in getOtherSettings(plugin)"
        :key="setting.key"
        v-slot="{ labelledBy }"
        :at="`${plugin.id}-${setting.key}`"
        :name="setting.title"
        :detail="setting.description"
      >
        <!-- boolean: Switch -->
        <Switch
          v-if="setting.type === 'boolean'"
          :model-value="getBooleanSetting(plugin.id, setting.key, setting.default)"
          :aria-labelledby="labelledBy"
          @update:model-value="(val) => onSettingChange(plugin.id, setting.key, val)"
        />

        <!-- number: NumberField -->
        <NumberField
          v-else-if="setting.type === 'number'"
          :model-value="getNumberSetting(plugin.id, setting.key, setting.default)"
          :min="setting.min ?? -999999"
          :max="setting.max ?? 999999"
          :step="setting.step ?? 1"
          :aria-labelledby="labelledBy"
          class="settings__number"
          @commit="(val) => val !== null && onSettingChange(plugin.id, setting.key, val)"
        />

        <!-- range: Slider -->
        <Slider
          v-else-if="setting.type === 'range'"
          :model-value="getNumberSetting(plugin.id, setting.key, setting.default)"
          :min="setting.min ?? 0"
          :max="setting.max ?? 100"
          :step="setting.step ?? 1"
          :aria-labelledby="labelledBy"
          class="settings__slider"
          @commit="(val) => onSettingChange(plugin.id, setting.key, val)"
        />

        <!-- enum: Select -->
        <Select
          v-else-if="setting.type === 'enum' && setting.options"
          :model-value="getStringSetting(plugin.id, setting.key, setting.default)"
          :choices="setting.options.map((opt) => ({ id: opt.value, text: opt.title || opt.value }))"
          :name="setting.title"
          :aria-labelledby="labelledBy"
          class="settings__choice"
          @update:model-value="(val) => onSettingChange(plugin.id, setting.key, val)"
        />

        <!-- string whose values are the models of a program: searchable Select -->
        <Select
          v-else-if="getProgramChoices(plugin, setting).length > 0"
          :model-value="getStringSetting(plugin.id, setting.key, setting.default)"
          :choices="getProgramChoices(plugin, setting)"
          :name="setting.title"
          :aria-labelledby="labelledBy"
          is-searchable
          :search-name="words.searchModels"
          class="settings__choice"
          @update:model-value="(val) => onSettingChange(plugin.id, setting.key, val)"
        />

        <!-- string: text input -->
        <input
          v-else
          type="text"
          :value="getStringSetting(plugin.id, setting.key, setting.default)"
          :aria-labelledby="labelledBy"
          class="settings__text-input"
          @change="
            (e) => onSettingChange(plugin.id, setting.key, (e.target as HTMLInputElement).value)
          "
        />
      </SettingRow>
    </section>
  </div>
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

.settings__slider {
  inline-size: var(--settings-choice, 16rem);
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
