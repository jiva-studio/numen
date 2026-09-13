<script setup lang="ts">
/**
 * What the tabs of a pane hold, and what stands there while it holds none.
 *
 * Every tab is drawn; the ones not shown are held out of sight, so what a tab
 * holds is alive for as long as the tab is.
 */
import { getPanelName, getTabName } from '../names'
import type { Pane, TabId } from '../../../lib/node'

const props = defineProps<{
  pane: Pane
}>()

const emit = defineEmits<{
  /** The panel let go of, back to the tab it is held under. */
  (event: 'leave', tab: TabId): void
}>()

defineSlots<{
  tab(props: { id: TabId }): unknown
  silence(): unknown
}>()

const tabName = (at: number): string => getTabName(props.pane.id, at)
const panelName = (at: number): string => getPanelName(props.pane.id, at)

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
</script>

<template>
  <div class="pane__body min-h-0 min-w-0 flex-1">
    <div
      v-for="(tab, at) in pane.tabs"
      :id="panelName(at)"
      :key="tab"
      class="pane__held"
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
.pane__held {
  display: none;
  block-size: 100%;
  min-block-size: 0;
  min-inline-size: 0;
}

.pane__held[data-showing] {
  display: block;
}

/* Drawn inside, because what a pane holds fills it to its edges. */
.pane__held:focus-visible {
  outline: none;
}

/* The whole of a pane holding no tabs. Its size is all it hands down. */
.pane__silence {
  block-size: 100%;
}

.pane__nothing {
  /* How plainly what is said in place of a tab is drawn. */
  --fade: 0.6;

  display: grid;
  place-items: center;
  block-size: 100%;
  margin: 0;
  opacity: var(--fade);
}
</style>
