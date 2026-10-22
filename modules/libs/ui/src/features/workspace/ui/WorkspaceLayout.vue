<script setup lang="ts">
/**
 * Tabs, and the ways of dividing the screen between them.
 *
 * The workspace it is given is the whole of what it draws, and a gesture gives
 * back another one. What a tab holds is the caller's: this takes titles and
 * hands identities back.
 */
import { computed, provide, useTemplateRef } from 'vue'
import { browserClock, type Clock } from '@/shared/lib/clock'
import { DragPreview } from '@/shared/ui/drag-preview'
import { WorkspaceBranch } from './branch'
import { WorkspacePane } from './pane'
import WorkspaceOverlay from './WorkspaceOverlay.vue'
import { WORKSPACE_CONTEXT, type WorkspaceContext } from '../model/context'
import { useTabDrag } from '../model/drag'
import { createNodeIdFactory } from '../lib/identity'
import { activateTab, closeTab, focusPane, resizeBranch, type NodeIdFactory } from '../lib/edit'
import { type NodeId, type Tab, type TabId, type Workspace } from '../lib/node'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(
  defineProps<{
    tabs: readonly Tab[]
    /** Where identities for what a gesture makes come from. */
    createId?: NodeIdFactory | undefined
    /** How close to the outer edge divides the whole workspace. */
    edge?: number
    /** How far the pointer travels before a press becomes a drag. */
    threshold?: number
    /** The least room a pane is worth drawing in. */
    minimum?: number
    /** The clock. Browser by default; a test hands in its own. */
    clock?: Clock
  }>(),
  { edge: 22, threshold: 4, minimum: 220, clock: () => browserClock },
)

const workspace = defineModel<Workspace>({ required: true })

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  /**
   * A close asked for, before anything is applied. A caller that calls `hold`
   * while it is being told takes the close, and the tab stays until the
   * caller closes it itself.
   */
  (event: 'close', tab: TabId, hold: () => void): void
  (event: 'activate', tab: TabId): void
  /** A menu asked for on a tab, and where on the screen. */
  (event: 'tab-context-menu', tab: TabId, at: { x: number; y: number }): void
  /**
   * The tab a pane is now showing, once it is on screen. Every tab of a pane is
   * drawn and the ones not shown are held out of sight, so what a tab holds is
   * told here that it can measure itself.
   */
  (event: 'show', tab: TabId): void
  /** A tab dragged out of the workspace into empty space or a new window. */
  (event: 'tear-off', tab: TabId, at: { x: number; y: number }): void
}>()

const slots = defineSlots<{
  tab(props: { id: TabId }): unknown
  /** What is drawn before a tab's name, which says what kind of tab it is. */
  icon(props: { id: TabId }): unknown
  /** What a mark is drawn as. Given none, a tab carrying one draws a dot. */
  mark(props: { id: TabId; mark: string }): unknown
  silence(): unknown
}>()

/* --------------------------------- State ---------------------------------- */
/** Every slot the caller gave, handed down under the name it was given. */
const passed = computed(() => Object.keys(slots) as (keyof typeof slots)[])

/** What a slot was given, handed on as it came. */
const getSlotProps = (bound: unknown) => (bound ?? {}) as { id: TabId; mark: string }

const findTab = (id: TabId): Tab | undefined => props.tabs.find((tab) => tab.id === id)

/** The identities this layout makes where a caller hands over no factory. */
const createNodeId = createNodeIdFactory()

const frame = useTemplateRef<HTMLElement>('frame')

const { hasMoved, overlay, label, position, dropTarget, press } = useTabDrag({
  workspace,
  frame,
  getTab: findTab,
  getIdFactory: () => props.createId ?? createNodeId,
  getEdge: () => props.edge,
  getThreshold: () => props.threshold,
  getClock: () => props.clock,
  onTearOff: (tab, at) => emit('tear-off', tab, at),
})

/* -------------------------------- Handlers -------------------------------- */
function onChooseTab(tab: TabId): void {
  if (hasMoved.value) return
  workspace.value = activateTab(workspace.value, tab)
  emit('activate', tab)
}

function onCloseTab(tab: TabId): void {
  let isPrevented = false
  emit('close', tab, () => {
    isPrevented = true
  })
  if (isPrevented) return

  workspace.value = closeTab(workspace.value, tab)
}

function onMenuTab(tab: TabId, at: { x: number; y: number }): void {
  emit('tab-context-menu', tab, at)
}

function onClaimPane(pane: NodeId): void {
  workspace.value = focusPane(workspace.value, pane)
}

function onResizeBranch(branch: NodeId, sizes: readonly number[]): void {
  workspace.value = resizeBranch(workspace.value, branch, sizes)
}

provide(
  WORKSPACE_CONTEXT,
  computed<WorkspaceContext>(() => ({
    findTab,
    focus: workspace.value.focus,
    minimum: props.minimum,
    choose: onChooseTab,
    close: onCloseTab,
    lift: press,
    menu: onMenuTab,
    claim: onClaimPane,
    resize: onResizeBranch,
    show: (tab: TabId) => emit('show', tab),
  })),
)
</script>

<template>
  <div ref="frame" class="workspace numen bg-surface text-ink relative min-h-0 min-w-0">
    <WorkspaceBranch
      v-if="workspace.root.kind === 'branch'"
      :node="workspace.root"
      :axis="workspace.axis"
      :depth="0"
    >
      <template v-for="name in passed" #[name]="slotProps">
        <slot :name="name" v-bind="getSlotProps(slotProps)" />
      </template>
    </WorkspaceBranch>

    <WorkspacePane
      v-else
      :pane="workspace.root"
      :is-focused="workspace.root.id === workspace.focus"
      @choose="onChooseTab"
      @close="onCloseTab"
      @lift="press"
      @menu="onMenuTab"
      @show="emit('show', $event)"
      @claim="onClaimPane(workspace.root.id)"
    >
      <template v-for="name in passed" #[name]="slotProps">
        <slot :name="name" v-bind="getSlotProps(slotProps)" />
      </template>
    </WorkspacePane>

    <WorkspaceOverlay v-if="overlay" :box="overlay" :has-caret="dropTarget?.kind === 'strip'" />

    <DragPreview
      v-if="label && position"
      class="workspace__dragged"
      :at="position"
      :label="label"
    />
  </div>
</template>

<style scoped>
.workspace {
  block-size: 100%;
  overflow: hidden;
}

/* The tab being dragged stands over the panes and the overlay both. */
.workspace__dragged {
  z-index: var(--numen-lift-tooltip);
}
</style>
