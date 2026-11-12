<script setup lang="ts">
/**
 * The page of one plugin: the breadcrumb bar, the header, the failure of a suspended plugin or the
 * bar of its worst health item, then the tabs Settings (the tab the page opens on), About,
 * Contributes and Data. The application draws the settings in the `settings` and `sections` slots;
 * Data is there only while the plugin keeps files in the vault or their deletion failed.
 */
import { computed, ref, watch, type VNode } from 'vue'
import { Prose } from '@numen/ui'
import BreadcrumbBar from './BreadcrumbBar.vue'
import { formatCategoryLabel, getPluginCategory } from './filter'
import PluginContributions from './PluginContributions.vue'
import PluginData from './PluginData.vue'
import type { PluginDetailProps } from './PluginDetail.types'
import PluginHeader from './PluginHeader.vue'
import PluginSettingsCard from './PluginSettingsCard.vue'
import PluginNotice from './PluginNotice.vue'
import PluginTabs from './PluginTabs.vue'
import { getReadmeText } from './readme'
import { getPluginState, getStartItem, getWorstItem, hasConnectionSettings } from './state'
import { FIRST_PLUGIN_TAB, getPanelElementId, getPluginTabs, getTabElementId } from './tabs'
import { PLUGIN_WORDS as words } from './words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<PluginDetailProps>()

const emit = defineEmits<{
  (e: 'back'): void
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
const categoryLabel = computed(() => formatCategoryLabel(getPluginCategory(props.plugin)))
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
    <BreadcrumbBar :current="categoryLabel" @back="onBack" />
    <PluginHeader
      :plugin="props.plugin"
      :state="state"
      :is-enabled="props.isEnabled"
      :is-pending="props.isPending ?? false"
      :can-install="props.canInstall"
      @toggle-enabled="onToggleEnabled"
      @install="onInstall"
    />

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

      <div
        v-if="activeTab === 'about' && readmeText"
        class="settings-card plugin-detail__readme"
      >
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
  max-inline-size: 66ch;
}

.plugin-detail__readme :deep(.prose > :first-child) {
  margin-block-start: 0;
}

.plugin-detail__readme :deep(.prose > :last-child) {
  margin-block-end: 0;
}
</style>
