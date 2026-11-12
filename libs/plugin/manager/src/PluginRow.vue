<script setup lang="ts">
/**
 * One plugin of the plugin list: a navigation row with the category icon, the name and the
 * description, a state label when the plugin is not installed or has an error or a warning, and a
 * context menu to enable, disable or open the plugin.
 */
import { computed, ref } from 'vue'
import { Menu, type MenuItem } from '@numen/ui'
import { getPluginCategory } from './filter'
import NavigationRow from './NavigationRow.vue'
import type { PluginHealth } from './pluginInfo'
import { isInstalled, type ManagedPlugin } from './plugins'
import { getPluginState, hasStateLabel } from './state'
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
  (e: 'toggle-enabled', isEnabled: boolean): void
}>()

/* --------------------------------- State ---------------------------------- */
const menuPosition = ref<{ x: number; y: number } | null>(null)
const pluginState = computed(() =>
  props.plugin.info ? getPluginState(props.isEnabled, props.health) : 'available',
)
const menuItems = computed<readonly MenuItem[]>(() => [
  ...(isInstalled(props.plugin)
    ? [{ id: 'toggle', text: props.isEnabled ? words.pluginDisable : words.pluginEnable }]
    : []),
  { id: 'configure', text: words.settings },
])

/* -------------------------------- Handlers -------------------------------- */
function onSelect(): void {
  emit('select-plugin')
}

function onContextMenu(e: MouseEvent): void {
  menuPosition.value = { x: e.clientX, y: e.clientY }
}

function onDismissMenu(): void {
  menuPosition.value = null
}

function onChooseMenuItem(id: string): void {
  menuPosition.value = null
  if (id === 'toggle') emit('toggle-enabled', !props.isEnabled)
  if (id === 'configure') emit('select-plugin')
}
</script>

<template>
  <NavigationRow
    :name="props.plugin.name"
    :detail="props.plugin.description"
    :category="getPluginCategory(props.plugin)"
    :label="words.pluginRow(props.plugin.name, pluginState)"
    :is-dimmed="pluginState === 'disabled'"
    @select="onSelect"
    @contextmenu.prevent="onContextMenu"
  >
    <template v-if="hasStateLabel(pluginState)" #trail>
      <span class="plugin-row__state" :class="`plugin-row__state--${pluginState}`">
        {{ words.pluginState[pluginState] }}
      </span>
    </template>
  </NavigationRow>
  <Menu
    v-if="menuPosition"
    :items="menuItems"
    :at="menuPosition"
    is-open
    :name="words.pluginActions"
    @choose="onChooseMenuItem"
    @dismiss="onDismissMenu"
  />
</template>

<style scoped>
.plugin-row__state {
  color: var(--numen-hushed);
}

.plugin-row__state--error {
  color: var(--numen-alarm);
}

.plugin-row__state--warning {
  color: var(--numen-ink);
}
</style>
