<script setup lang="ts">
/**
 * The list of tabs a pane is chosen from. It carries the pane's identity on the
 * element, so that a drag can find out what the pointer is over by asking the
 * document.
 *
 * The strip is walked with the arrows, and what is reached is shown.
 */
import { inject } from 'vue'
import { WorkspaceTab } from '../../tab'
import { stepTo } from '../keys'
import { getPanelName, getTabName } from '../names'
import { WORKSPACE_CONTEXT } from '../../../model/context'
import type { Pane, TabId } from '../../../lib/node'

const props = withDefaults(
  defineProps<{
    pane: Pane
    /** The pane a tab would open into. */
    focused?: boolean
  }>(),
  { focused: false },
)

const emit = defineEmits<{
  (event: 'choose', tab: TabId): void
  (event: 'close', tab: TabId): void
  (event: 'lift', tab: TabId, at: PointerEvent): void
}>()

defineSlots<{
  icon(props: { id: TabId }): unknown
  mark(props: { id: TabId; mark: string }): unknown
}>()

const workspace = inject(WORKSPACE_CONTEXT)

/** What a tab is called. A tab with no title is shown by its identity. */
const titleOf = (tab: TabId): string => workspace?.value.tabOf(tab)?.title ?? tab

/** What a tab is carrying, and nothing for a tab carrying nothing. */
const getMarkOf = (tab: TabId): string | undefined => workspace?.value.tabOf(tab)?.mark

const tabName = (at: number): string => getTabName(props.pane.id, at)
const panelName = (at: number): string => getPanelName(props.pane.id, at)

/** The tabs as they are drawn, each under the tab it stands for. */
const drawnTabs = new Map<TabId, { focus: () => void }>()

const holdTab = (tab: TabId, element: unknown): void => {
  if (element) drawnTabs.set(tab, element as { focus: () => void })
  else drawnTabs.delete(tab)
}

/** The keyboard put on one tab of the strip. */
function reach(tab: TabId): void {
  drawnTabs.get(tab)?.focus()
}

/**
 * The tabs are what the arrows walk; the way to a new tab is a stop of its
 * own.
 */
function onStripKey(at: number, event: KeyboardEvent): void {
  const next = stepTo(event.key, at, props.pane.tabs.length)
  const tab = next === null ? undefined : props.pane.tabs[next]
  if (next === null || tab === undefined) return

  event.preventDefault()
  emit('choose', tab)
  reach(tab)
}

defineExpose({ reach })
</script>

<template>
  <div
    class="pane__strip flex min-w-0 shrink-0 items-stretch overflow-hidden"
    role="tablist"
    :data-workspace-strip="pane.id"
  >
    <WorkspaceTab
      v-for="(tab, at) in pane.tabs"
      :id="tabName(at)"
      :ref="(element) => holdTab(tab, element)"
      :key="tab"
      :aria-controls="panelName(at)"
      :tab="tab"
      :title="titleOf(tab)"
      :mark="getMarkOf(tab)"
      :showing="tab === pane.active"
      :focused="tab === pane.active && focused"
      @lift="emit('lift', tab, $event)"
      @close="emit('close', tab)"
      @click="emit('choose', tab)"
      @keydown="(event: KeyboardEvent) => onStripKey(at, event)"
    >
      <template v-if="$slots.icon" #icon>
        <slot name="icon" :id="tab" />
      </template>

      <template v-if="$slots.mark" #mark="bound">
        <slot name="mark" :id="tab" v-bind="bound" />
      </template>
    </WorkspaceTab>
  </div>
</template>

<style scoped>
/* Sits on the same surface as what it stands over, told apart by one line. */
.pane__strip {
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}
</style>
