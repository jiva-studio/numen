<script setup lang="ts">
/**
 * Declarative settings fields and secrets for a plugin: switches,
 * number inputs, sliders, selects, paths, secrets, and text fields. A setting or
 * secret whose condition fails is not shown, and keeps its value.
 */
import { NumberField, Select, Slider, Switch } from '@numen/ui'
import SettingRow from '../setting-row/SettingRow.vue'
import { PathField } from '../path-field'
import PluginSecretField from './PluginSecret.vue'
import SettingsViews from './SettingsViews.vue'
import type { InstallationSettings, PluginFieldsHost, SecretChoices } from '../../types'
import type { PluginInfo } from '@/entities/settings'
import { useEnumFields, usePluginFields, usePluginPaths } from '../../model/usePluginFields'
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  settings: InstallationSettings
  host: PluginFieldsHost
  choices: SecretChoices
  plugin: PluginInfo
}>()

const emit = defineEmits<{
  (e: 'open-secrets'): void
}>()

/* --------------------------------- State ---------------------------------- */
const fieldDeps = {
  getSettings: () => props.settings,
  getHost: () => props.host,
  getPlugin: () => props.plugin,
}
const fields = usePluginFields(fieldDeps)
const { getBooleanSetting, getNumberSetting, getStringSetting, setSetting: onSettingChange } = fields
const { getDraft, setDraft, writeDraft } = fields
const {
  getEnumOptions,
  getEnumChoices,
  getEnumValue,
  getShownSettings,
  getTopSecrets,
  getSecretsAfter,
} = useEnumFields(fieldDeps)
const {
  problems: pathProblems,
  hasManyPaths,
  choosePath: onPathChoose,
  clearPath: onPathClear,
} = usePluginPaths(fieldDeps, fields)
</script>

<template>
  <SettingsViews :settings="props.settings" :host="props.host" :plugin="props.plugin" />

  <PluginSecretField
    :choices="props.choices"
    :plugin="plugin"
    :secrets="getTopSecrets()"
    @open-secrets="emit('open-secrets')"
  />

  <template v-for="setting in getShownSettings()" :key="setting.key">
    <SettingRow
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
        v-else-if="setting.type === 'file'"
        :path="getStringSetting(setting.key, setting.default)"
        :problem="pathProblems[setting.key] ?? ''"
        :labelled-by="labelledBy"
        :is-multi="hasManyPaths(setting)"
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

    <PluginSecretField
      :choices="props.choices"
      :plugin="plugin"
      :secrets="getSecretsAfter(setting.key)"
      @open-secrets="emit('open-secrets')"
    />
  </template>
</template>

<style scoped>
.settings__number {
  inline-size: var(--settings-value);
  text-align: end;
}

.settings__slider-container {
  --gap: 0.75rem;
  display: flex;
  align-items: center;
  gap: var(--gap);
  inline-size: var(--settings-choice);
}

.settings__slider-container .settings__slider {
  flex: 1 1 auto;
  inline-size: auto;
}

.settings__slider-value {
  --min-width: 2.5rem;
  font-size: var(--numen-text-2);
  color: var(--numen-ink);
  font-variant-numeric: tabular-nums;
  min-inline-size: var(--min-width);
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
  --pad: 0.25rem 0.625rem;
  box-sizing: border-box;
  inline-size: var(--settings-choice);
  block-size: var(--numen-action-size);
  padding: var(--pad);
  background-color: var(--numen-field-bg);
  border: var(--numen-stroke) solid var(--numen-field-border);
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
