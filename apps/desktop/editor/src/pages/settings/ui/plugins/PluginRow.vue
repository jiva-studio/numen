<script setup lang="ts">
/**
 * One installed plugin row in the plugins registry: its name, version,
 * single-line description, failure/disabled badge, and navigation chevron.
 */
import { computed, ref } from 'vue'
import { Menu, type MenuItem } from '@numen/ui'
import type { PluginInfo } from '@/entities/settings'
import { getPluginState, type PluginState } from '../../lib/failures'
import { WORDS as words } from '../../words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  plugin: PluginInfo
  isEnabled: boolean
  categoryPrefix?: string
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (e: 'select-plugin'): void
  (e: 'toggle-enabled', val: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
const menuPosition = ref<{ x: number; y: number } | null>(null)
const versionSuffix = computed(() => (props.plugin.version ? ` v${props.plugin.version}` : ''))
const categoryPrefixText = computed(() =>
  props.categoryPrefix ? `${props.categoryPrefix} / ` : '',
)
const rowTitle = computed(
  () => `${categoryPrefixText.value}${props.plugin.name}${versionSuffix.value}`,
)
const pluginState = computed<PluginState>(() => getPluginState(props.plugin, props.isEnabled))

const menuItems = computed<readonly MenuItem[]>(() => [
  {
    id: 'toggle',
    text: props.isEnabled ? words.pluginDisable : words.pluginEnable,
  },
  {
    id: 'configure',
    text: words.settings,
  },
])

/* -------------------------------- Handlers -------------------------------- */
function onClick(): void {
  emit('select-plugin')
}

function onKeydown(e: KeyboardEvent): void {
  if (e.key === 'Enter' || e.key === ' ') {
    e.preventDefault()
    emit('select-plugin')
  }
}

function onContextMenu(e: MouseEvent): void {
  menuPosition.value = { x: e.clientX, y: e.clientY }
}

function onDismissMenu(): void {
  menuPosition.value = null
}

function onChooseMenuItem(id: string): void {
  menuPosition.value = null
  if (id === 'toggle') {
    emit('toggle-enabled', !props.isEnabled)
  } else if (id === 'configure') {
    emit('select-plugin')
  }
}
</script>

<template>
  <div
    class="plugin-row"
    :class="{ 'plugin-row--disabled': !props.isEnabled }"
    role="button"
    tabindex="0"
    :aria-label="rowTitle"
    @click="onClick"
    @keydown="onKeydown"
    @contextmenu.prevent="onContextMenu"
  >
    <div class="plugin-row__main">
      <div class="plugin-row__header">
        <span v-if="props.categoryPrefix" class="plugin-row__category">
          {{ props.categoryPrefix }} /
        </span>
        <span class="plugin-row__name">{{ props.plugin.name }}</span>
        <span v-if="props.plugin.version" class="plugin-version-badge">
          v{{ props.plugin.version }}
        </span>
      </div>
      <p v-if="props.plugin.description" class="plugin-row__description">
        {{ props.plugin.description }}
      </p>
    </div>

    <div class="plugin-row__trail">
      <span
        v-if="pluginState === 'stopped'"
        class="plugin-row__badge"
        :class="`plugin-row__badge--${pluginState}`"
      >
        {{ words.pluginState[pluginState] }}
      </span>
      <span class="plugin-row__chevron" aria-hidden="true">&rsaquo;</span>
    </div>

    <Menu
      v-if="menuPosition"
      :items="menuItems"
      :at="menuPosition"
      is-open
      name="Plugin Actions"
      @choose="onChooseMenuItem"
      @dismiss="onDismissMenu"
    />
  </div>
</template>

<style scoped>
.plugin-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--numen-panel-gap, 0.75rem);
  min-block-size: 3.5rem;
  padding: 0.625rem 0.75rem;
  border-block-end: var(--numen-stroke, 1px) solid var(--numen-rule);
  cursor: pointer;
  user-select: none;
  transition:
    background-color 0.15s ease,
    opacity 0.15s ease;
  border-radius: var(--numen-radius-small, 4px);
}

.plugin-row:last-child {
  border-block-end: none;
}

.plugin-row:hover {
  background-color: var(--numen-surface-hover, rgba(125, 125, 125, 0.08));
}

.plugin-row:focus-visible {
  outline: 2px solid var(--numen-accent, #0066cc);
  outline-offset: -2px;
}

.plugin-row--disabled {
  opacity: 0.5;
}

.plugin-row--disabled .plugin-row__name,
.plugin-row--disabled .plugin-row__category,
.plugin-row--disabled .plugin-row__description,
.plugin-row--disabled .plugin-row__chevron {
  color: var(--numen-hushed);
}

.plugin-row__main {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  flex: 1 1 auto;
  min-inline-size: 0;
}

.plugin-row__header {
  display: flex;
  align-items: baseline;
  gap: 0.25rem;
}

.plugin-row__category {
  color: var(--numen-hushed);
  font-weight: 400;
  font-size: var(--numen-text-2, 0.95rem);
}

.plugin-row__name {
  color: var(--numen-ink);
  font-weight: 500;
  font-size: var(--numen-text-2, 0.95rem);
}

.plugin-version-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.125rem 0.375rem;
  border-radius: var(--numen-radius-pill, 9999px);
  background: var(--numen-surface, rgba(125, 125, 125, 0.12));
  border: 1px solid var(--numen-rule, rgba(125, 125, 125, 0.2));
  color: var(--numen-hushed);
  font-size: var(--numen-text-1, 0.75rem);
  font-weight: 500;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  vertical-align: middle;
  margin-inline-start: 0.25rem;
}

.plugin-row__description {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-1);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-inline-size: 100%;
}

.plugin-row__trail {
  display: flex;
  align-items: center;
  gap: var(--numen-panel-gap, 0.75rem);
  flex-shrink: 0;
}

.plugin-row__badge {
  font-size: var(--numen-text-1, 0.75rem);
  padding: 0.125rem 0.375rem;
  border-radius: var(--numen-radius-small, 3px);
}

.plugin-row__badge--stopped {
  color: var(--numen-danger, #d14);
  background-color: var(--numen-danger-soft, rgba(221, 17, 68, 0.1));
}

.plugin-row__chevron {
  font-size: 1.25rem;
  line-height: 1;
  color: var(--numen-hushed);
}
</style>
