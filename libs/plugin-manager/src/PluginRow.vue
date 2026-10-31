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
  gap: var(--numen-panel-gap);
  min-block-size: 3.5rem;
  padding: 0.625rem 0.75rem;
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
  cursor: pointer;
  user-select: none;
  transition:
    background-color 0.15s ease,
    opacity 0.15s ease;
  border-radius: var(--numen-radius-small);
}

.plugin-row:last-child {
  border-block-end: none;
}

.plugin-row:hover {
  background-color: var(--numen-surface-hover);
}

.plugin-row:focus-visible {
  outline: 2px solid var(--numen-accent);
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
  font-size: var(--numen-text-2);
}

.plugin-row__name {
  color: var(--numen-ink);
  font-weight: 500;
  font-size: var(--numen-text-2);
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
  gap: var(--numen-panel-gap);
  flex-shrink: 0;
}

.plugin-row__badge {
  font-size: var(--numen-text-1);
  padding: 0.125rem 0.375rem;
  border-radius: var(--numen-radius-small);
}

.plugin-row__badge--stopped {
  color: var(--numen-danger);
  background-color: var(--numen-danger-soft);
}

.plugin-row__chevron {
  font-size: 1.25rem;
  line-height: 1;
  color: var(--numen-hushed);
}
</style>
