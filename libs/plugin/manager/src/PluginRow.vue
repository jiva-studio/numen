<script setup lang="ts">
/**
 * One plugin row in the plugins registry: the category icon, the name, a single-line description,
 * a state label when the plugin is not installed or has an error or a warning, and the navigation
 * chevron.
 */
import { computed, ref } from 'vue'
import { ChevronRight } from '@lucide/vue'
import { Menu, type MenuItem } from '@numen/ui'
import { getCategoryIcon } from './categoryIcons'
import { getPluginCategory } from './filter'
import type { PluginHealth } from './pluginInfo'
import { isInstalled, type ManagedPlugin } from './plugins'
import { getPluginState, type PluginState } from './state'
import { PLUGIN_WORDS as words } from './words'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  plugin: ManagedPlugin
  isEnabled: boolean
  health?: readonly PluginHealth[]
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (e: 'select-plugin'): void
  (e: 'toggle-enabled', val: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
const menuPosition = ref<{ x: number; y: number } | null>(null)
const icon = computed(() => getCategoryIcon(getPluginCategory(props.plugin)))
const pluginState = computed<PluginState>(() =>
  props.plugin.info ? getPluginState(props.isEnabled, props.health) : 'available',
)

const LABELED_STATES: readonly PluginState[] = ['available', 'error', 'warning']
const hasStateLabel = computed(() => LABELED_STATES.includes(pluginState.value))

const menuItems = computed<readonly MenuItem[]>(() => [
  ...(isInstalled(props.plugin)
    ? [{ id: 'toggle', text: props.isEnabled ? words.pluginDisable : words.pluginEnable }]
    : []),
  { id: 'configure', text: words.settings },
])

/* -------------------------------- Handlers -------------------------------- */
function onClick(): void {
  emit('select-plugin')
}

function onKeydown(e: KeyboardEvent): void {
  if (e.target !== e.currentTarget) return
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
    :class="{ 'plugin-row--disabled': pluginState === 'disabled' }"
    role="button"
    tabindex="0"
    :aria-label="props.plugin.name"
    @click="onClick"
    @keydown="onKeydown"
    @contextmenu.prevent="onContextMenu"
  >
    <span class="plugin-row__glyph" aria-hidden="true">
      <component :is="icon" :size="20" :stroke-width="1.75" />
    </span>
    <div class="plugin-row__main">
      <span class="plugin-row__name">{{ props.plugin.name }}</span>
      <p v-if="props.plugin.description" class="plugin-row__description">
        {{ props.plugin.description }}
      </p>
    </div>

    <div class="plugin-row__trail">
      <span
        v-if="hasStateLabel"
        class="plugin-row__state"
        :class="`plugin-row__state--${pluginState}`"
      >
        {{ words.pluginState[pluginState] }}
      </span>
      <ChevronRight class="plugin-row__chevron" :size="16" aria-hidden="true" />
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
  gap: 0.75rem;
  box-sizing: border-box;
  min-block-size: var(--settings-row-min, 3.25rem);
  padding: var(--settings-row-pad, 0.625rem) var(--settings-card-pad, 1rem);
  border-block-start: var(--numen-stroke) solid var(--numen-rule);
  cursor: pointer;
  user-select: none;
  transition: background-color var(--numen-motion-hover) ease;
}

.plugin-row:first-child {
  border-block-start: none;
}

.plugin-row:hover {
  background-color: color-mix(in oklab, var(--numen-raised), var(--numen-ink) 5%);
}

.plugin-row:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
  outline-offset: calc(-1 * var(--numen-stroke));
}

.plugin-row__glyph {
  display: grid;
  place-items: center;
  flex: none;
  color: var(--numen-hushed);
}

.plugin-row--disabled .plugin-row__glyph,
.plugin-row--disabled .plugin-row__name {
  opacity: 0.55;
}

.plugin-row__main {
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  min-inline-size: 0;
}

.plugin-row__name {
  color: var(--numen-ink);
  font-weight: 500;
  font-size: var(--numen-text-2);
}

.plugin-row__description {
  margin: 0;
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.plugin-row__trail {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-shrink: 0;
}

.plugin-row__state {
  color: var(--numen-hushed);
  font-size: var(--numen-text-2);
}

.plugin-row__state--error {
  color: var(--numen-alarm);
}

.plugin-row__state--warning {
  color: var(--numen-ink);
}

.plugin-row__chevron {
  color: var(--numen-hushed);
}
</style>
