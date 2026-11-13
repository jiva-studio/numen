<script setup lang="ts">
/**
 * The page of one plugin: the toolbar with the path Settings, the category and the plugin and with
 * the state and the enabled switch or the install button, the header, the failure of a suspended
 * plugin or the bar of its worst health item, then the tabs Settings (the tab the page opens on), About,
 * Contributes and Data. The application draws the settings in the `settings` and `sections` slots;
 * Data is there only while the plugin keeps files in the vault or their deletion failed.
 */
import { computed, ref, watch, type VNode } from 'vue'
import { Button, Prose, Switch } from '@numen/ui'
import { formatCategoryLabel, getPluginCategory } from './filter'
import PluginContributions from './PluginContributions.vue'
import PluginData from './PluginData.vue'
import type { PluginDetailProps } from './PluginDetail.types'
import PluginHeader from './PluginHeader.vue'
import PluginSettingsCard from './PluginSettingsCard.vue'
import PluginNotice from './PluginNotice.vue'
import PluginTabs from './PluginTabs.vue'
import { isInstalled } from './plugins'
import SettingsToolbar from './SettingsToolbar.vue'
import { getReadmeText } from './readme'
import { getPluginState, getStartItem, getWorstItem, hasConnectionSettings } from './state'
import { FIRST_PLUGIN_TAB, getPanelElementId, getPluginTabs, getTabElementId } from './tabs'
import { PLUGIN_WORDS as words } from './words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<PluginDetailProps>()

const emit = defineEmits<{
  (e: 'back'): void
  /** The category of the plugin was chosen in the path, by its key. */
  (e: 'open-category', category: string): void
  (e: 'toggle-enabled', isEnabled: boolean): void
  (e: 'retry'): void
  (e: 'install'): void
  (e: 'delete-data'): void
  (e: 'run-action', key: string): void
}>()

defineSlots<{
  /** The settings of the plugin, drawn by the application. */
  settings?(): VNode[]
  /** Sections of the settings tab under the card of the settings, drawn by the application. */
  sections?(): VNode[]
}>()

/* --------------------------------- State ---------------------------------- */
const startItem = computed(() => getStartItem(props.health ?? []))
const worstItem = computed(() => getWorstItem(props.health ?? []))
const category = computed(() => getPluginCategory(props.plugin))
const path = computed(() => [
  words.settings,
  formatCategoryLabel(category.value),
  props.plugin.name,
])
const hasConnections = computed(() => hasConnectionSettings(props.plugin.info))
const contributions = computed(() => props.plugin.info?.contributions ?? [])
const readmeText = computed(() => getReadmeText(props.readme))
const state = computed(() =>
  props.plugin.info ? getPluginState(props.isEnabled, props.health) : 'available',
)
const tabs = computed(() => getPluginTabs(props.dataFileCount > 0 || props.problem !== ''))
const activeTab = ref(FIRST_PLUGIN_TAB)
const idBase = computed(() => `plugin-${props.plugin.id.replaceAll('.', '-')}`)

/* --------------------------------- Hooks ---------------------------------- */
watch(tabs, (now) => {
  if (!now.some((one) => one.id === activeTab.value)) activeTab.value = FIRST_PLUGIN_TAB
})

watch(
  () => props.plugin.id,
  () => {
    activeTab.value = FIRST_PLUGIN_TAB
  },
)

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}

function onOpen(level: number): void {
  if (level === 0) emit('back')
  else emit('open-category', category.value)
}

function onToggleEnabled(isEnabled: boolean): void {
  emit('toggle-enabled', isEnabled)
}

function onRetry(): void {
  emit('retry')
}

function onInstall(): void {
  emit('install')
}

function onDeleteData(): void {
  emit('delete-data')
}

function onRunAction(key: string): void {
  emit('run-action', key)
}
</script>

<template>
  <div class="plugin-detail">
    <SettingsToolbar :path="path" @back="onBack" @open="onOpen">
      <span class="plugin-detail__state">
        <span
          class="plugin-detail__dot"
          :class="`plugin-detail__dot--${state}`"
          aria-hidden="true"
        />
        {{ words.pluginState[state] }}
      </span>
      <Switch
        v-if="isInstalled(props.plugin)"
        :model-value="props.isEnabled"
        :is-busy="props.isPending ?? false"
        :aria-label="words.pluginSwitch(props.plugin.name)"
        @update:model-value="onToggleEnabled"
      />
      <Button v-else-if="props.canInstall" size="small" @click="onInstall">
        {{ words.pluginInstall }}
      </Button>
    </SettingsToolbar>
    <PluginHeader :plugin="props.plugin" />

    <PluginNotice
      v-if="startItem"
      :item="startItem"
      :action-label="words.pluginRetry"
      @run-action="onRetry"
    />
    <PluginNotice
      v-else-if="worstItem"
      :item="worstItem"
      :action-label="worstItem.actionLabel"
      @run-action="onRunAction(worstItem.key)"
    />

    <PluginTabs v-model="activeTab" :tabs="tabs" :label="words.pluginSections" :id-base="idBase" />

    <div
      :id="getPanelElementId(idBase, activeTab)"
      class="plugin-detail__panel"
      role="tabpanel"
      :aria-labelledby="getTabElementId(idBase, activeTab)"
    >
      <PluginSettingsCard v-if="activeTab === 'settings' && props.hasSettings">
        <slot name="settings" />
      </PluginSettingsCard>
      <slot v-if="activeTab === 'settings'" name="sections" />
      <p
        v-if="activeTab === 'settings' && !props.hasSettings && !hasConnections"
        class="plugin-detail__nothing"
      >
        {{ words.noSettings }}
      </p>

      <div v-if="activeTab === 'about' && readmeText" class="settings-card plugin-detail__readme">
        <Prose :text="readmeText" />
      </div>
      <p v-else-if="activeTab === 'about'" class="plugin-detail__nothing">
        {{ props.plugin.description }}
      </p>

      <PluginContributions
        v-if="activeTab === 'contributions' && contributions.length > 0"
        :contributions="contributions"
        :is-apple="props.isApple ?? false"
      />
      <p v-else-if="activeTab === 'contributions'" class="plugin-detail__nothing">
        {{ words.noContributions }}
      </p>

      <PluginData
        v-if="activeTab === 'data'"
        :plugin-id="props.plugin.id"
        :plugin-name="props.plugin.name"
        :file-count="props.dataFileCount"
        :problem="props.problem"
        @delete-data="onDeleteData"
      />
    </div>
  </div>
</template>

<style scoped>
.plugin-detail {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 1rem;
}

.plugin-detail__state {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
}

.plugin-detail__dot {
  inline-size: 0.375rem;
  block-size: 0.375rem;
  border-radius: 50%;
  background-color: var(--numen-hushed);
}

.plugin-detail__dot--stopped,
.plugin-detail__dot--error {
  background-color: var(--numen-alarm);
}

.plugin-detail__panel {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.plugin-detail__nothing {
  margin: 0;
  padding-inline: var(--settings-card-pad);
  color: var(--numen-hushed);
}

.plugin-detail__readme {
  padding: var(--settings-card-pad);
}

.plugin-detail__readme :deep(.prose) {
  max-inline-size: 36rem;
}

.plugin-detail__readme :deep(.prose > :first-child) {
  margin-block-start: 0;
}

.plugin-detail__readme :deep(.prose > :last-child) {
  margin-block-end: 0;
}
</style>
