<script setup lang="ts">
/**
 * What the tabs of a pane hold, and what is shown there while it holds none.
 *
 * Every tab is rendered; the ones not shown are hidden, so what a tab
 * holds stays alive for as long as the tab does.
 */
import { getPanelName, getTabName } from '../names'
import type { Pane, TabId } from '../../../lib/node'

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  pane: Pane
}>()

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /** The panel release, back to the tab it is held under. */
  (event: 'leave', tab: TabId): void
}>()

defineSlots<{
  tab(props: { id: TabId }): unknown
  silence(): unknown
}>()

/* -------------------------------- Handlers -------------------------------- */
/**
 * The way out of what a tab holds, back to the tab itself. A panel that acts
 * on Escape itself keeps it.
 */
function onPanelEscape(event: KeyboardEvent): void {
  if (event.defaultPrevented) return

  const showing = props.pane.active
  if (showing === null) return

  event.preventDefault()
  emit('leave', showing)
}

/* -------------------------------- Helpers --------------------------------- */
const tabName = (pos: number): string => getTabName(props.pane.id, pos)
const panelName = (pos: number): string => getPanelName(props.pane.id, pos)
</script>

<template>
  <div class="pane__body min-h-0 min-w-0 flex-1">
    <div
      v-for="(tab, at) in pane.tabs"
      :id="panelName(at)"
      :key="tab"
      class="pane__pending"
      role="tabpanel"
      tabindex="0"
      :aria-labelledby="tabName(at)"
      :data-showing="tab === pane.active || undefined"
      @keydown.escape="onPanelEscape"
    >
      <slot name="tab" :id="tab" />
    </div>
    <div v-if="pane.tabs.length === 0" class="pane__silence">
      <slot name="silence">
        <p class="pane__nothing text-small text-hushed font-sans">Nothing open</p>
      </slot>
    </div>
  </div>
</template>

<style scoped>
.pane__body {
  position: relative;
  overflow: hidden;
}

/* Every tab of the pane is drawn; the ones not shown are held out of sight. What
   a tab holds is alive for as long as the tab is: a caret, a scroll offset, an
   undo history. */
.pane__pending {
  display: none;
  block-size: 100%;
  min-block-size: 0;
  min-inline-size: 0;
  contain: strict;
}

.pane__pending[data-showing] {
  display: block;
}

/* Drawn inside, because what a pane holds fills it to its edges. */
.pane__pending:focus-visible {
  outline: none;
}

/* The whole of a pane holding no tabs. Its size is all it hands down. */
.pane__silence {
  block-size: 100%;
}

.pane__nothing {
  /* How prominently the text shown in place of a tab is rendered. */
  --fade: 0.6;

  display: grid;
  place-items: center;
  block-size: 100%;
  margin: 0;
  opacity: var(--fade);
}
</style>
