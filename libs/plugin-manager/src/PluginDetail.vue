<script setup lang="ts">
/**
 * The page of one plugin: a header card with the README and the enabled switch (or the install
 * button for a plugin that is not installed), the failure card of a suspended plugin, the AI
 * consent, the settings the application draws in the `settings` slot, and the plugin's data.
 */
import { computed, ref } from 'vue'
import { Button, Prose, Switch } from '@numen/ui'
import ChevronIcon from './ChevronIcon.vue'
import DetailRow from './DetailRow.vue'
import PluginData from './PluginData.vue'
import PluginStopped from './PluginStopped.vue'
import type { PluginDetailProps } from './PluginDetail.types'
import { isInstalled } from './plugins'
import { parsePluginReadme } from './readme'
import { PLUGIN_WORDS as words } from './words'

/* ----------------------------- Props & Emits ------------------------------ */
const props = defineProps<PluginDetailProps>()

const emit = defineEmits<{
  (e: 'back'): void
  (e: 'toggle-enabled', isEnabled: boolean): void
  (e: 'retry'): void
  (e: 'install'): void
  (e: 'consent-change', isAllowed: boolean): void
  (e: 'delete-data'): void
}>()

defineSlots<{
  /** The settings of the plugin, drawn by the application. */
  settings(): unknown
}>()

/* --------------------------------- State ---------------------------------- */
const isExpanded = ref(false)
const consent = computed(() => props.plugin.info?.consent ?? 'none')
const hasConsent = computed(() => consent.value !== 'none')
const readmeParts = computed(() =>
  parsePluginReadme(props.readme, props.plugin.id, props.plugin.description),
)

/* -------------------------------- Handlers -------------------------------- */
function onBack(): void {
  emit('back')
}

function onToggleEnabled(isEnabled: boolean): void {
  emit('toggle-enabled', isEnabled)
}

function onToggleReadme(): void {
  isExpanded.value = !isExpanded.value
}

function onRetry(): void {
  emit('retry')
}

function onDisable(): void {
  emit('toggle-enabled', false)
}

function onInstall(): void {
  emit('install')
}

function onConsentChange(isAllowed: boolean): void {
  emit('consent-change', isAllowed)
}

function onDeleteData(): void {
  emit('delete-data')
}
</script>

<template>
  <div class="plugin-detail">
    <Button variant="ghost" size="small" class="plugin-detail__back" @click="onBack">
      &larr; {{ words.backToPlugins }}
    </Button>

    <div class="plugin-detail__header">
      <h2 class="plugin-detail__title">
        <span>{{ props.plugin.name }}</span>
        <span v-if="props.plugin.version" class="plugin-version-badge">
          v{{ props.plugin.version }}
        </span>
      </h2>
      <div class="plugin-detail__master-switch">
        <Switch
          v-if="isInstalled(props.plugin)"
          :model-value="props.isEnabled"
          :aria-label="`${words.pluginEnabled} ${props.plugin.name}`"
          @update:model-value="onToggleEnabled"
        />
        <Button v-else-if="props.canInstall" size="small" @click="onInstall">
          {{ words.pluginInstall }}
        </Button>
        <span v-else class="plugin-detail__absent">{{ words.pluginState.available }}</span>
      </div>

      <div v-if="readmeParts.summary" class="plugin-detail__description">
        <Prose :text="readmeParts.summary" />
      </div>

      <div v-if="isExpanded && readmeParts.more" class="plugin-detail__more">
        <Prose :text="readmeParts.more" />
      </div>

      <button
        v-if="readmeParts.more"
        type="button"
        class="plugin-detail__toggle"
        :aria-expanded="isExpanded"
        @click="onToggleReadme"
      >
        <span>{{ isExpanded ? words.showLess : words.readMore }}</span>
        <ChevronIcon
          class="plugin-detail__toggle-icon"
          :class="{ 'plugin-detail__toggle-icon--open': isExpanded }"
        />
      </button>
    </div>

    <PluginStopped
      v-if="props.plugin.info?.failure"
      :failure="props.plugin.info.failure"
      @retry="onRetry"
      @disable="onDisable"
    />

    <div v-if="hasConsent || props.hasSettings" class="plugin-detail__settings">
      <DetailRow
        v-if="hasConsent"
        v-slot="{ labelledBy }"
        :at="`ai-consent-${props.plugin.id}`"
        :name="words.consentAllowed"
        :detail="words.consentDetail[consent]"
      >
        <Switch
          :model-value="consent === 'allowed'"
          :aria-labelledby="labelledBy"
          @update:model-value="onConsentChange"
        />
      </DetailRow>
      <slot name="settings" />
    </div>

    <PluginData
      :plugin-id="props.plugin.id"
      :plugin-name="props.plugin.name"
      :file-count="props.dataFileCount"
      :problem="props.problem"
      @delete-data="onDeleteData"
    />
  </div>
</template>

<style scoped>
.plugin-detail {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  max-inline-size: var(--settings-measure, 54rem);
  margin-inline: auto;
}

.plugin-detail__back {
  align-self: flex-start;
  margin-block-end: 0.75rem;
  color: var(--numen-hushed);
}

.plugin-detail__settings {
  padding-inline: 1rem;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-raised);
}

.plugin-detail__settings + .plugin-detail__settings {
  margin-block-start: 1.25rem;
}

.plugin-detail__settings :deep(.settings__row:last-child) {
  border-block-end: none;
}

.plugin-detail__header {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  column-gap: var(--numen-panel-gap);
  row-gap: 0.625rem;
  align-items: center;
  padding: 0.875rem 1rem;
  margin-block-end: 1.25rem;
  border: var(--numen-stroke) solid var(--numen-rule);
  border-radius: var(--numen-radius);
  background-color: var(--numen-raised);
}

.plugin-detail__title {
  margin: 0;
  font-size: var(--numen-text-3);
  font-weight: 600;
  color: var(--numen-ink);
}

.plugin-version-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.125rem 0.4375rem;
  border-radius: var(--numen-radius-pill);
  background: color-mix(in srgb, var(--numen-accent) 16%, transparent);
  border: 1px solid color-mix(in srgb, var(--numen-accent) 40%, transparent);
  color: var(--numen-accent);
  font-size: var(--numen-text-1);
  font-weight: 500;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  vertical-align: middle;
  margin-inline-start: 0.375rem;
}

.plugin-detail__master-switch {
  justify-self: end;
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.plugin-detail__absent {
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
}

.plugin-detail__description {
  grid-column: 1 / -1;
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
}

.plugin-detail__description :deep(.prose) {
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
}

.plugin-detail__description :deep(p:first-child),
.plugin-detail__more :deep(:first-child) {
  margin-block-start: 0;
}

.plugin-detail__description :deep(p:last-child),
.plugin-detail__more :deep(:last-child) {
  margin-block-end: 0;
}

.plugin-detail__more {
  grid-column: 1 / -1;
  padding-block-start: 0.625rem;
  border-block-start: var(--numen-stroke) solid var(--numen-rule);
  font-size: var(--numen-text-2);
}

.plugin-detail__toggle {
  grid-column: 1 / -1;
  justify-self: start;
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.125rem 0;
  border: none;
  background: none;
  color: var(--numen-hushed);
  font-family: inherit;
  font-size: var(--numen-text-1);
  font-weight: 500;
  cursor: pointer;
}

.plugin-detail__toggle:hover {
  color: var(--numen-ink);
}

.plugin-detail__toggle-icon {
  transition: transform var(--numen-motion-hover) ease;
}

.plugin-detail__toggle-icon--open {
  transform: rotate(180deg);
}
</style>
