<script setup lang="ts">
/**
 * Declarative settings fields for a plugin: agent choices, switches,
 * number inputs, sliders, selects, paths, and text fields. A setting whose
 * condition fails is not shown, and keeps its value.
 */
import { computed, ref } from 'vue'
import { NumberField, Select, Slider, Switch } from '@numen/ui'
import SettingRow from '../setting-row/SettingRow.vue'
import { AgentChoice } from '../agent-choice'
import { PathField } from '../path-field'
import type { SettingsTabState } from '../../types'
import {
  createSettingFields,
  isShown,
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
import { createTextDrafts } from '../../lib/drafts'
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
const agentSettings = computed(() => props.plugin.settings.filter((one) => one.type === AGENT_TYPE))
const otherSettings = computed(() =>
  props.plugin.settings.filter(
    (one) => one.type !== AGENT_TYPE && isShown(props.plugin, one.when, readSetting),
  ),
)
const { getEnumOptions, getEnumValue, getDetail, getOrigin } = createEnumFields(
  () => props.plugin,
  readSetting,
  words,
)
/** Why a path setting did not take the path last chosen, by key. */
const pathProblems = ref<Record<string, string>>({})
const { getDraft, setDraft, writeDraft } = createTextDrafts(onSettingChange)

/* -------------------------------- Handlers -------------------------------- */
function onSettingChange(key: string, value: unknown): void {
  fields.setValue(['plugins', props.plugin.id, key], value)
}

async function onPathChoose(key: string): Promise<void> {
  const problem = await installation.value.choosePath(props.plugin.id, key)
  pathProblems.value = { ...pathProblems.value, [key]: problem }
}

function onPathClear(key: string): void {
  pathProblems.value = { ...pathProblems.value, [key]: '' }
  onSettingChange(key, '')
}

function onAgentChoose(key: string, choice: CompletionChoice): void {
  fields.setValue(['plugins', props.plugin.id, key], choice)
}

/* -------------------------------- Helpers --------------------------------- */
function readSetting(key: string): unknown {
  return installation.value.getSetting(['plugins', props.plugin.id, key])
}

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
      :detail="getDetail(setting)"
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
          :choices="
            getEnumOptions(setting).map((opt) => ({ id: opt.value, text: opt.title || opt.value }))
          "
          :name="setting.title"
          :aria-labelledby="labelledBy"
          class="settings__choice"
          @update:model-value="(val) => onSettingChange(setting.key, val)"
        />
        <span class="settings__origin">{{ getOrigin(setting) }}</span>
      </span>

      <PathField
        v-else-if="setting.type === 'file' || setting.type === 'folder'"
        :path="getStringSetting(setting.key, '')"
        :problem="pathProblems[setting.key] ?? ''"
        :labelled-by="labelledBy"
        :words="words"
        @choose="onPathChoose(setting.key)"
        @clear="onPathClear(setting.key)"
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
        :value="getDraft(setting.key) ?? getStringSetting(setting.key, setting.default)"
        :aria-labelledby="labelledBy"
        class="settings__text-input"
        @input="(e) => setDraft(setting.key, (e.target as HTMLInputElement).value)"
        @change="(e) => writeDraft(setting.key, (e.target as HTMLInputElement).value)"
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

.settings__enum {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.125rem;
}

.settings__fixed {
  color: var(--numen-ink);
  font-size: var(--numen-text-2);
}

.settings__origin {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
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
