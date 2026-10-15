<script setup lang="ts">
/**
 * Dynamic settings section rendered for active plugins with declarative schemas.
 */
import { computed } from 'vue'
import { NumberField, Select, Slider, Switch } from '@numen/ui'
import SettingRow from './setting-row/SettingRow.vue'
import type { SettingsTabState } from '../types'
import { createSettingFields, type PluginInfo } from '@/entities/settings'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{ state: SettingsTabState }>()

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)
const fields = createSettingFields(() => props.state.installation)

const plugins = computed<readonly PluginInfo[]>(() => {
  const list = installation.value.plugins?.value ?? []
  return list.filter((one) => one.settings.length > 0)
})

/* -------------------------------- Handlers -------------------------------- */
function onSettingChange(pluginId: string, key: string, value: unknown): void {
  fields.setValue(['plugins', pluginId, key], value)
}

/* -------------------------------- Helpers --------------------------------- */
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

      <SettingRow
        v-for="setting in plugin.settings"
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
          :choices="setting.options.map((opt) => ({ id: opt, text: opt }))"
          :name="setting.title"
          :aria-labelledby="labelledBy"
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
  inline-size: var(--settings-choice, 18rem);
}

.settings__choice {
  inline-size: var(--settings-choice, 18rem);
}

.settings__text-input {
  box-sizing: border-box;
  inline-size: var(--settings-choice, 18rem);
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
