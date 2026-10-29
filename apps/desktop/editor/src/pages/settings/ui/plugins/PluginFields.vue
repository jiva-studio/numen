<script setup lang="ts">
/**
 * Declarative settings fields and secrets for a plugin: agent choices, switches,
 * number inputs, sliders, selects, paths, secrets, and text fields. A setting or
 * secret whose condition fails is not shown, and keeps its value.
 */
import { computed, ref } from 'vue'
import { NumberField, Select, Slider, Switch } from '@numen/ui'
import SettingRow from '../setting-row/SettingRow.vue'
import { AgentChoice } from '../agent-choice'
import { PathField } from '../path-field'
import { SecretField } from '../secret-field'
import SettingsViews from './SettingsViews.vue'
import type { SettingsTabState } from '../../types'
import {
  createSettingFields,
  type CompletionChoice,
  type PluginInfo,
  type PluginSettingInfo,
} from '@/entities/settings'
import {
  AGENT_SETTING_TYPE as AGENT_TYPE,
  DEFAULT_AGENT_AT,
  createAgentModelChoices,
  createPluginAgentChoices,
  createProgramModelChoices,
  getProgramModelsAt,
  findAgent,
  getAgents,
  readChoice,
} from '../../lib/agents'
import { createEnumFields } from '../../lib/enums'
import { appendPathEntry, createTextDrafts, removePathEntry } from '../../model/drafts'
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
const {
  getEnumOptions,
  getEnumChoices,
  getEnumValue,
  getShownSettings,
  getTopSecrets,
  getSecretsAfter,
} = createEnumFields(() => props.plugin, readSetting)
/** Why a path setting did not take the path last chosen, by key. */
const pathProblems = ref<Record<string, string>>({})
const { getDraft, setDraft, writeDraft } = createTextDrafts(onSettingChange)

/* -------------------------------- Handlers -------------------------------- */
function onSettingChange(key: string, value: unknown): void {
  fields.setValue(['plugins', props.plugin.id, key], value)
}

async function onPathChoose(setting: PluginSettingInfo): Promise<void> {
  const previous = getStringSetting(setting.key, setting.default)
  const problem = await installation.value.choosePath(props.plugin.id, setting.key)
  pathProblems.value = { ...pathProblems.value, [setting.key]: problem }
  if (problem === '' && previous !== '' && setting.key === 'executable_path') {
    onSettingChange(setting.key, appendPathEntry(previous, getStringSetting(setting.key, '')))
  }
}

function onPathClear(setting: PluginSettingInfo, targetPath?: string): void {
  pathProblems.value = { ...pathProblems.value, [setting.key]: '' }
  const current = getStringSetting(setting.key, setting.default)
  onSettingChange(setting.key, removePathEntry(current, targetPath))
}

function onSecretSave(name: string, key: string): void {
  installation.value.writeSecret(props.plugin.id, name, key)
}

function onSecretDelete(name: string): void {
  installation.value.deleteSecret(props.plugin.id, name)
}

/* -------------------------------- Helpers --------------------------------- */
function readSetting(key: string): unknown {
  return installation.value.getSetting(['plugins', props.plugin.id, key])
}

function getChoice(key: string): CompletionChoice {
  return readChoice(readSetting(key))
}

function getAgentChoices(choice: CompletionChoice) {
  const selection = { defaultAgent: defaultAgent.value, chosen: choice.agent }
  return createPluginAgentChoices(agents.value, selection, words)
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

function getBooleanSetting(key: string, fallback: unknown): boolean {
  const val = readSetting(key) ?? fallback
  return typeof val === 'boolean' ? val : Boolean(fallback)
}

function getNumberSetting(key: string, fallback: unknown): number {
  const val = readSetting(key) ?? fallback
  return typeof val === 'number' ? val : Number(fallback ?? 0)
}

function getStringSetting(key: string, fallback: unknown): string {
  const val = readSetting(key) ?? fallback
  return typeof val === 'string' ? val : String(fallback ?? '')
}
</script>

<template>
  <SettingsViews :state="props.state" :plugin="props.plugin" after="" />

  <SecretField
    v-for="secret in getTopSecrets()"
    :key="`${props.plugin.id}-${secret.name}`"
    :at="`key-${props.plugin.id}-${secret.name}`"
    :name="secret.title"
    :is-stored="props.plugin.storedSecrets.includes(secret.name)"
    :words="words"
    @save="(key) => onSecretSave(secret.name, key)"
    @delete="onSecretDelete(secret.name)"
  />

  <template v-for="setting in getShownSettings()" :key="setting.key">
    <AgentChoice
      v-if="setting.type === AGENT_TYPE"
      :at="`${props.plugin.id}-${setting.key}`"
      :agent-name="setting.title"
      :agent-detail="setting.description"
      :model-name="words.pluginModel"
      :model-detail="words.defaultModelDetail"
      :agents="getAgentChoices(getChoice(setting.key))"
      :models="getModelChoices(getChoice(setting.key))"
      :choice="getChoice(setting.key)"
      @choose="(choice) => onSettingChange(setting.key, choice)"
    />

    <SettingRow
      v-else
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

      <span
        v-else-if="setting.type === 'enum'"
        class="settings__enum"
        :data-testid="`setting-${setting.key}`"
      >
        <span
          v-if="getEnumOptions(setting).length === 1"
          class="settings__fixed"
          :aria-labelledby="labelledBy"
        >
          {{ getEnumOptions(setting)[0]?.title || getEnumValue(setting) }}
        </span>
        <Select
          v-else
          :model-value="getEnumValue(setting)"
          :choices="getEnumChoices(setting)"
          :name="setting.title"
          :aria-labelledby="labelledBy"
          class="settings__choice"
          @update:model-value="(val) => onSettingChange(setting.key, val)"
        />
      </span>

      <PathField
        v-else-if="setting.type === 'file' || setting.type === 'folder'"
        :path="getStringSetting(setting.key, setting.default)"
        :problem="pathProblems[setting.key] ?? ''"
        :labelled-by="labelledBy"
        :is-multi="setting.title === 'Executable Paths'"
        :words="words"
        @choose="onPathChoose(setting)"
        @clear="(targetPath) => onPathClear(setting, targetPath)"
      />

      <Select
        v-else-if="setting.type === 'string' && getEnumOptions(setting).length > 0"
        :model-value="getEnumValue(setting) || getStringSetting(setting.key, setting.default)"
        :choices="getEnumChoices(setting)"
        :name="setting.title"
        :aria-labelledby="labelledBy"
        is-searchable
        can-enter-custom
        :search-name="words.searchModels"
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
        can-enter-custom
        :search-name="words.searchModels"
        class="settings__choice"
        @update:model-value="(val) => onSettingChange(setting.key, val)"
      />

      <input
        v-else
        type="text"
        :value="getDraft(setting.key) ?? getStringSetting(setting.key, setting.default)"
        :aria-labelledby="labelledBy"
        class="settings__text-input"
        @input="(e) => setDraft(setting.key, (e.target as HTMLInputElement).value)"
        @change="(e) => writeDraft(setting.key, (e.target as HTMLInputElement).value)"
      />
    </SettingRow>

    <SecretField
      v-for="secret in getSecretsAfter(setting.key)"
      :key="`${props.plugin.id}-${secret.name}`"
      :at="`key-${props.plugin.id}-${secret.name}`"
      :name="secret.title"
      :is-stored="props.plugin.storedSecrets.includes(secret.name)"
      :words="words"
      @save="(key) => onSecretSave(secret.name, key)"
      @delete="onSecretDelete(secret.name)"
    />

    <SettingsViews :state="props.state" :plugin="props.plugin" :after="setting.key" />
  </template>
</template>

<style scoped>
.settings__number {
  inline-size: var(--settings-value);
  text-align: end;
}

.settings__slider-container {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  inline-size: var(--settings-choice);
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
  inline-size: var(--settings-choice);
}

.settings__enum {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}

.settings__fixed {
  color: var(--numen-ink);
  font-size: var(--numen-text-2);
}

.settings__text-input {
  box-sizing: border-box;
  inline-size: var(--settings-choice);
  block-size: var(--numen-action-size);
  padding: 0.25rem 0.625rem;
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-rule);
  border-radius: var(--numen-radius);
  color: var(--numen-ink);
  font-family: inherit;
  font-size: var(--numen-text-2);
  text-align: start;
}

.settings__text-input:focus {
  outline: var(--numen-stroke) solid var(--numen-accent);
}
</style>
