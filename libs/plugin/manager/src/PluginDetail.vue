<script setup lang="ts">
/**
 * The page of one plugin: a header with its icon, name, version, state and enabled switch (or the
 * install button for a plugin that is not installed), the failure of a suspended plugin or the bar
 * of its worst health item, then tabs: the README (the tab the page opens on), the settings the
 * application draws in the `settings` and `sections` slots (a quiet line when there are none), what
 * the plugin adds to the window, and its data.
 */
import { computed, ref, watch } from 'vue'
import { ChevronLeft, ChevronRight } from '@lucide/vue'
import { Button, Prose, Switch } from '@numen/ui'
import { getCategoryIcon } from './categoryIcons'
import { formatCategoryLabel, getPluginCategory } from './filter'
import PluginContributions from './PluginContributions.vue'
import PluginData from './PluginData.vue'
import PluginHealthBar from './PluginHealthBar.vue'
import PluginSettingsCard from './PluginSettingsCard.vue'
import PluginStopped from './PluginStopped.vue'
import PluginTabs from './PluginTabs.vue'
import type { PluginDetailProps } from './PluginDetail.types'
import { isInstalled } from './plugins'
import { getPluginState, getStartItem, getWorstItem } from './state'
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
  settings(): unknown
  /** Sections of the settings tab under the card of the settings, drawn by the application. */
  sections(): unknown
}>()

/* --------------------------------- State ---------------------------------- */
const startItem = computed(() => getStartItem(props.health ?? []))
const worstItem = computed(() => getWorstItem(props.health ?? []))
const icon = computed(() => getCategoryIcon(getPluginCategory(props.plugin)))
const categoryLabel = computed(() => formatCategoryLabel(getPluginCategory(props.plugin)))
const hasConnectionSettings = computed(
  () => props.plugin.info?.settings.some((one) => one.type === 'connections') ?? false,
)
const contributions = computed(() => props.plugin.info?.contributions ?? [])
const readmeText = computed(() => props.readme.replace(/<!--[\s\S]*?-->/g, '').trim())
const state = computed(() =>
  props.plugin.info ? getPluginState(props.isEnabled, props.health) : 'available',
)
const stateText = computed(() =>
  state.value === 'enabled' ? words.pluginRunning : words.pluginState[state.value],
)

const tabs = computed(() => [
  { id: 'about', text: words.pluginTabs.about },
  { id: 'settings', text: words.pluginTabs.settings },
  { id: 'contributions', text: words.pluginTabs.contributions },
  ...(props.dataFileCount > 0 || props.problem ? [{ id: 'data', text: words.pluginTabs.data }] : []),
])
const chosenTab = ref(props.initialTab ?? 'about')
watch(
  () => props.initialTab,
  (tab) => {
    if (tab) chosenTab.value = tab
  },
)

watch(tabs, (now) => {
  if (!now.some((one) => one.id === chosenTab.value)) chosenTab.value = now[0]?.id ?? 'about'
})

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
    <div class="plugin-detail__bar">
      <button
        type="button"
        class="plugin-detail__back"
        :aria-label="words.backToPlugins"
        @click="onBack"
      >
        <ChevronLeft :size="16" aria-hidden="true" />
      </button>
      <nav class="plugin-detail__crumbs">
        <button type="button" class="plugin-detail__crumb" @click="onBack">
          {{ words.settings }}
        </button>
        <ChevronRight :size="12" aria-hidden="true" />
        <span>{{ categoryLabel }}</span>
      </nav>
    </div>

    <div class="plugin-detail__header">
      <component :is="icon" class="plugin-detail__glyph" :size="32" :stroke-width="1.5" aria-hidden="true" />
      <div class="plugin-detail__heading">
        <div class="plugin-detail__name-line">
          <h2 class="plugin-detail__title">{{ props.plugin.name }}</h2>
          <code class="plugin-detail__id">{{ props.plugin.id }}</code>
        </div>
        <div class="plugin-detail__meta">
          <span v-if="props.plugin.version">v{{ props.plugin.version }}</span>
          <span class="plugin-detail__state">
            <i class="plugin-detail__dot" :class="`plugin-detail__dot--${state}`" />
            {{ stateText }}
          </span>
        </div>
        <p v-if="props.plugin.description" class="plugin-detail__description">
          {{ props.plugin.description }}
        </p>
      </div>
      <div class="plugin-detail__master-switch">
        <Switch
          v-if="isInstalled(props.plugin)"
          :model-value="props.isEnabled"
          :is-busy="props.isPending ?? false"
          :aria-label="`${words.pluginEnabled} ${props.plugin.name}`"
          @update:model-value="onToggleEnabled"
        />
        <Button v-else-if="props.canInstall" size="small" @click="onInstall">
          {{ words.pluginInstall }}
        </Button>
      </div>
    </div>

    <PluginStopped v-if="startItem" :item="startItem" @retry="onRetry" />
    <PluginHealthBar v-else-if="worstItem" :item="worstItem" @run-action="onRunAction" />

    <PluginTabs v-model="chosenTab" :tabs="tabs" :label="words.pluginSections" />

    <div v-if="chosenTab === 'settings'" class="plugin-detail__section">
      <PluginSettingsCard v-if="props.hasSettings" class="plugin-detail__settings">
        <slot name="settings" />
      </PluginSettingsCard>
      <slot name="sections" />
      <p v-if="!props.hasSettings && !hasConnectionSettings" class="plugin-detail__nothing">
        {{ words.noSettings }}
      </p>
    </div>

    <div v-else-if="chosenTab === 'contributions'" class="plugin-detail__section">
      <PluginContributions v-if="contributions.length > 0" :contributions="contributions" />
      <p v-else class="plugin-detail__nothing">{{ words.noContributions }}</p>
    </div>

    <div v-else-if="chosenTab === 'data'" class="plugin-detail__section">
      <PluginData
        :plugin-id="props.plugin.id"
        :plugin-name="props.plugin.name"
        :file-count="props.dataFileCount"
        :problem="props.problem"
        @delete-data="onDeleteData"
      />
    </div>

    <div v-else class="plugin-detail__section">
      <div v-if="readmeText" class="plugin-detail__card plugin-detail__readme">
        <Prose :text="readmeText" />
      </div>
      <p v-else class="plugin-detail__nothing">{{ props.plugin.description }}</p>
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

.plugin-detail__bar {
  position: sticky;
  inset-block-start: calc(-1 * var(--numen-gutter));
  z-index: 5;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-block: calc(-1 * var(--numen-gutter)) 0;
  padding-block: var(--numen-gutter) 0.5rem;
  background-color: var(--numen-surface);
}

.plugin-detail__back {
  display: grid;
  place-items: center;
  inline-size: 1.75rem;
  block-size: 1.75rem;
  border: none;
  border-radius: var(--numen-radius);
  background: none;
  color: var(--numen-hushed);
  cursor: pointer;
}

.plugin-detail__back:hover,
.plugin-detail__crumb:hover {
  background-color: var(--numen-bubble-bg);
  color: var(--numen-ink);
}

.plugin-detail__crumbs {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  color: var(--numen-hushed);
}

.plugin-detail__crumb {
  padding: 0.125rem 0.25rem;
  border: none;
  border-radius: var(--numen-radius-tight);
  background: none;
  color: inherit;
  font-family: inherit;
  font-size: var(--numen-text-2);
  cursor: pointer;
}

.plugin-detail__header {
  display: grid;
  grid-template-columns: 32px minmax(0, 1fr) auto;
  column-gap: 1rem;
  align-items: start;
}

.plugin-detail__glyph {
  color: var(--numen-hushed);
}

.plugin-detail__heading {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  min-inline-size: 0;
}

.plugin-detail__name-line {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 0.25rem 0.75rem;
}

.plugin-detail__title {
  margin: 0;
  color: var(--numen-ink);
  font-size: var(--numen-text-3);
  font-weight: 600;
  line-height: 1.3;
}

.plugin-detail__id {
  color: var(--numen-hushed);
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-1);
}

.plugin-detail__master-switch {
  display: flex;
  align-items: center;
  block-size: calc(var(--numen-text-3) * 1.3);
}

.plugin-detail__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0 0.75rem;
  margin-block-start: 0.125rem;
  line-height: 1;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  font-variant-numeric: tabular-nums;
}

.plugin-detail__state {
  display: inline;
  line-height: 1;
}

.plugin-detail__dot {
  display: inline-block;
  vertical-align: middle;
  position: relative;
  inset-block-start: -0.09em;
  inline-size: 6px;
  block-size: 6px;
  margin-inline-end: 6px;
  border-radius: 50%;
  background-color: var(--numen-hushed);
}

.plugin-detail__dot--stopped {
  background-color: var(--numen-alarm);
}

.plugin-detail__description {
  margin: 0;
  color: var(--numen-hushed);
  max-inline-size: 62ch;
}

.plugin-detail__nothing {
  margin: 0;
  padding-inline: var(--settings-card-pad, 1rem);
  color: var(--numen-hushed);
}

.plugin-detail__section {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.plugin-detail__card {
  padding-block: calc(var(--settings-card-pad, 1rem) - var(--settings-row-pad, 0.75rem));
  padding-inline: var(--settings-card-pad, 1rem);
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-raised);
}

.plugin-detail__readme {
  padding-block: var(--settings-card-pad, 1rem);
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

.plugin-detail__group {
  margin: 1rem 0 0 0.125rem;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  font-weight: 600;
}

.plugin-detail__value {
  font-variant-numeric: tabular-nums;
}

.plugin-detail__code {
  font-family: var(--numen-font-mono);
  font-size: var(--numen-text-2);
}
</style>
