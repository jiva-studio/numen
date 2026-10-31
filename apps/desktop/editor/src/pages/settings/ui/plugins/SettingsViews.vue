<script setup lang="ts">
/**
 * The views a plugin's frontend registered for its settings form, drawn at the top of the form
 * in the order the manifest declares them. A declared view without a registration draws nothing.
 */
import { computed } from 'vue'
import type { SettingsTabState } from '../../types'
import { createSettingFields, type PluginInfo } from '@/entities/settings'
import { getRegisteredViews, getSettingValues } from '../../lib/settingsViews'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: SettingsTabState
  plugin: PluginInfo
}>()

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)
const fields = createSettingFields(() => props.state.installation)
const views = computed(() =>
  getRegisteredViews(
    props.plugin.settingsViews ?? [],
    installation.value.getSettingsViews?.(props.plugin.id) ?? new Map(),
  ),
)
const values = computed(() =>
  getSettingValues(props.plugin, (key) =>
    installation.value.getSetting(['plugins', props.plugin.id, key]),
  ),
)

/* -------------------------------- Handlers -------------------------------- */
function onSettingUpdate(key: string, value: unknown): void {
  fields.setValue(['plugins', props.plugin.id, key], value)
}
</script>

<template>
  <component
    :is="view.component"
    v-for="view in views"
    :key="`${props.plugin.id}-view-${view.id}`"
    :plugin-id="props.plugin.id"
    :values="values"
    @update:setting="onSettingUpdate"
  />
</template>
