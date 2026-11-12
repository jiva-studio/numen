<script setup lang="ts">
/**
 * The sections of a plugin page as a row of tabs over a hairline; the active tab is underlined in
 * the accent. The arrow keys move between the tabs and activate them, Home and End go to the first
 * and the last. Each tab controls the panel of the ids `getPanelElementId` gives for `idBase`.
 */
import { useTemplateRef } from 'vue'
import {
  getPanelElementId,
  getTabElementId,
  getTabIndexForKey,
  type PluginTab,
  type PluginTabId,
} from './tabs'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  tabs: readonly PluginTab[]
  label: string
  idBase: string
}>()

const activeTabId = defineModel<PluginTabId>({ required: true })

/* --------------------------------- State ---------------------------------- */
const buttons = useTemplateRef<HTMLButtonElement[]>('buttons')

/* -------------------------------- Handlers -------------------------------- */
function onKeydown(e: KeyboardEvent, index: number): void {
  const next = getTabIndexForKey(e.key, index, props.tabs.length)
  const tab = next === undefined ? undefined : props.tabs[next]
  if (next === undefined || !tab) return
  e.preventDefault()
  activeTabId.value = tab.id
  buttons.value?.[next]?.focus()
}

function onSelect(id: PluginTabId): void {
  activeTabId.value = id
}
</script>

<template>
  <div class="plugin-tabs" role="tablist" :aria-label="props.label">
    <button
      v-for="(tab, index) in props.tabs"
      :id="getTabElementId(props.idBase, tab.id)"
      :key="tab.id"
      ref="buttons"
      type="button"
      role="tab"
      class="plugin-tabs__tab"
      :aria-selected="activeTabId === tab.id"
      :aria-controls="getPanelElementId(props.idBase, tab.id)"
      :tabindex="activeTabId === tab.id ? 0 : -1"
      @click="onSelect(tab.id)"
      @keydown="onKeydown($event, index)"
    >
      {{ tab.text }}
    </button>
  </div>
</template>

<style scoped>
.plugin-tabs {
  display: flex;
  gap: 1.5rem;
  padding-inline: var(--settings-card-pad);
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

.plugin-tabs__tab {
  margin-block-end: calc(-1 * var(--numen-stroke));
  padding: 0.5rem 0;
  border: none;
  border-block-end: 0.125rem solid transparent;
  background: none;
  color: var(--numen-hushed);
  font-family: inherit;
  font-size: var(--numen-text-2);
  font-weight: 500;
  cursor: pointer;
  transition: color var(--numen-motion-hover) ease;
}

.plugin-tabs__tab:hover,
.plugin-tabs__tab[aria-selected='true'] {
  color: var(--numen-ink);
}

.plugin-tabs__tab[aria-selected='true'] {
  border-block-end-color: var(--numen-accent);
}

.plugin-tabs__tab:focus-visible {
  outline: var(--numen-stroke) solid var(--numen-accent);
  outline-offset: -0.125rem;
}
</style>
