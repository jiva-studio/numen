<script setup lang="ts">
/**
 * The views a plugin's frontend registered for its settings form, placed after the setting
 * named by `after`, or at the top of the form for an empty `after`. A declared view without
 * a registration draws nothing.
 */
import { computed } from 'vue'
import type { SettingsTabState } from '../../types'
import { createSettingFields, type PluginInfo } from '@/entities/settings'
import { getRegisteredViewsAfter, getSettingValues } from '../../lib/settingsViews'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  state: SettingsTabState
  plugin: PluginInfo
  after: string
}>()

/* --------------------------------- State ---------------------------------- */
const installation = computed(() => props.state.installation)
const fields = createSettingFields(() => props.state.installation)
const views = computed(() =>
  getRegisteredViewsAfter(
    props.plugin.settingsViews ?? [],
    installation.value.getSettingsViews?.(props.plugin.id) ?? new Map(),
    props.after,
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
