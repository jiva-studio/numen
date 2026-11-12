<script setup lang="ts">
import { computed, inject, type ComponentPublicInstance, type VNode } from 'vue'
import { WorkspaceTab } from '../../tab'
import { isFocusable, stepTo } from '../keys'
import { getPanelName, getTabName } from '../names'
import { NO_STRIP_EDGE, WORKSPACE_CONTEXT } from '../../../model/context'
import type { Pane, TabId } from '../../../lib/node'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    pane: Pane
    isFocused?: boolean
  }>(),
  { isFocused: false },
)

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'choose', tab: TabId): void
  (event: 'close', tab: TabId): void
  (event: 'press', tab: TabId, pointer: PointerEvent): void
  (event: 'menu', tab: TabId, pos: { x: number; y: number }): void
}>()

defineSlots<{
  icon(props: { id: TabId }): VNode[]
  mark(props: { id: TabId; mark: string }): VNode[]
}>()

defineExpose({ reach })

/* ---------------------------------- State --------------------------------- */
const workspace = inject(WORKSPACE_CONTEXT)

const edge = computed(() => workspace?.value.getStripEdge(props.pane.id) ?? NO_STRIP_EDGE)

const edgeStyle = computed(() =>
  edge.value.isDraggable
    ? {
        '--wails-draggable': 'drag',
        paddingInlineStart: `${edge.value.leftInset}px`,
        paddingInlineEnd: `${edge.value.rightInset}px`,
      }
    : {},
)

const drawnTabs = new Map<TabId, { focus: () => void }>()

/* -------------------------------- Handlers -------------------------------- */
const onHoldTab = (tab: TabId, element: Element | ComponentPublicInstance | null): void => {
  if (isFocusable(element)) drawnTabs.set(tab, element)
  else drawnTabs.delete(tab)
}

function onStripKey(pos: number, event: KeyboardEvent): void {
  const next = stepTo(event.key, pos, props.pane.tabs.length)
  const tab = next === null ? undefined : props.pane.tabs[next]
  if (next === null || tab === undefined) return

  event.preventDefault()
  emit('choose', tab)
  reach(tab)
}

/* --------------------------------- Helpers -------------------------------- */
const getTitle = (tab: TabId): string => workspace?.value.findTab(tab)?.title ?? tab

const getMarkOf = (tab: TabId): string | undefined => workspace?.value.findTab(tab)?.mark

const tabName = (pos: number): string => getTabName(props.pane.id, pos)
const panelName = (pos: number): string => getPanelName(props.pane.id, pos)

function reach(tab: TabId): void {
  drawnTabs.get(tab)?.focus()
}
</script>

<template>
  <div
    class="pane__strip flex min-w-0 shrink-0 items-stretch overflow-hidden"
    role="tablist"
    :data-workspace-strip="pane.id"
    :data-window-drag="edge.isDraggable || undefined"
    :style="edgeStyle"
  >
    <WorkspaceTab
      v-for="(tab, index) in pane.tabs"
      :id="tabName(index)"
      :ref="(element) => onHoldTab(tab, element)"
      :key="tab"
      :aria-controls="panelName(index)"
      :tab="tab"
      :title="getTitle(tab)"
      :mark="getMarkOf(tab)"
      :is-showing="tab === pane.active"
      :is-focused="tab === pane.active && isFocused"
      @press="emit('press', tab, $event)"
      @close="emit('close', tab)"
      @menu="emit('menu', tab, $event)"
      @click="emit('choose', tab)"
      @keydown="(event: KeyboardEvent) => onStripKey(index, event)"
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
.pane__strip {
  border-block-end: var(--numen-stroke) solid var(--numen-rule);
}

.pane__strip[data-window-drag] {
  box-sizing: border-box;
  min-block-size: var(--numen-title-bar-height);
  align-items: center;
}
</style>
