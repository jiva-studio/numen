<script setup lang="ts">
/**
 * The page of one plugin in the settings tab: the plugin manager's page, with its health items and
 * the declarative settings drawn in its slot, the settings the fixes of the items name marked.
 */
import { computed } from 'vue'
import { PluginDetail as PluginDetailView } from '@numen/plugin-manager'
import type { PluginInfo } from '@/entities/settings'
import type { JsonValue } from '@/shared/api/json'
import { usePluginDetail, usePluginSwitch } from '../../model/usePluginDetail'
import { usePluginFields } from '../../model/usePluginFields'
import type {
  InstallationConnections,
  InstallationPlugin,
  InstallationSettings,
  MarkedSetting,
} from '../../types'
import PluginConnectionSections from './PluginConnectionSections.vue'
import PluginSettings from './PluginSettings.vue'
import PluginSlots from './PluginSlots.vue'
import SettingsViews from './SettingsViews.vue'
import { getManagedHealth, getMarkedSettings } from '../../lib/pluginHealth'
import { keepEqual } from '../../lib/keepEqual'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<{
  settings: InstallationSettings
  manager: InstallationPlugin
  connections: InstallationConnections
  plugin: PluginInfo
  markedSetting?: MarkedSetting | null
}>()

const emit = defineEmits<{
  (e: 'back'): void
  (e: 'open-connections'): void
}>()

/* --------------------------------- State ---------------------------------- */
const detailDeps = {
  getSettings: () => props.settings,
  getManager: () => props.manager,
  getPlugin: () => props.plugin,
}
const {
  managed,
  hasSettings,
  slotGroups,
  isApple,
  fileCount,
  deleteProblem,
  readmeText,
  deleteData,
} = usePluginDetail(detailDeps)
const { isEnabled, isPending, setEnabled, retry } = usePluginSwitch(detailDeps)
const slotsKey = computed(() => props.plugin.settings.find((one) => one.type === 'slots')?.key)
const slotsLayout = computed<JsonValue | undefined>((held) =>
  keepEqual(held, slotsKey.value === undefined ? undefined : fields.getValue(slotsKey.value)),
)
const hasSlots = computed(() => slotsKey.value !== undefined && slotGroups.value.length > 0)
const contributedTabs = computed(() =>
  (props.plugin.settingsTabs ?? []).map((one) => ({ id: one.id, title: one.title })),
)
const fields = usePluginFields({
  getSettings: () => props.settings,
  getHost: () => props.manager,
  getPlugin: () => props.plugin,
})
const items = computed(() =>
  props.manager.health.items.value.filter((item) => item.pluginId === props.plugin.id),
)
const health = computed(() =>
  getManagedHealth(items.value, props.plugin.id, props.manager.health.canRunCommand),
)
const marked = computed(() => [
  ...getMarkedSettings(items.value, props.plugin.id),
  ...(props.markedSetting ? [props.markedSetting.setting] : []),
])

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}

function onRunAction(key: string): void {
  const action = items.value.find((item) => item.key === key)?.action
  if (action) props.manager.health.runCommand(action.command, action.values)
}
</script>

<template>
  <PluginDetailView
    v-if="managed"
    :plugin="managed"
    :is-enabled="isEnabled"
    :is-pending="isPending"
    :readme="readmeText"
    :data-file-count="fileCount"
    :problem="deleteProblem"
    :can-install="false"
    :has-settings="hasSettings"
    :has-slots="hasSlots"
    :contributed-tabs="contributedTabs"
    :health="health"
    :is-apple="isApple"
    @back="onBack"
    @toggle-enabled="setEnabled"
    @retry="retry"
    @delete-data="deleteData"
    @run-action="onRunAction"
  >
    <template #settings>
      <PluginSettings
        v-if="hasSettings"
        :settings="props.settings"
        :host="props.manager"
        :plugin="props.plugin"
        :marked="marked"
        :focused="props.markedSetting ?? null"
      />
    </template>
    <template #tab="{ id }">
      <SettingsViews
        :settings="props.settings"
        :host="props.manager"
        :plugin="props.plugin"
        :tab="id"
      />
    </template>
    <template #slots>
      <PluginSlots
        v-if="slotsKey !== undefined"
        :groups="slotGroups"
        :layout="slotsLayout"
        @change="(layout) => fields.setSetting(slotsKey ?? '', layout)"
      />
    </template>
    <template #sections>
      <PluginConnectionSections
        :settings="props.settings"
        :host="props.manager"
        :connections="props.connections"
        :plugin="props.plugin"
        @open-connections="emit('open-connections')"
      />
    </template>
  </PluginDetailView>
</template>
