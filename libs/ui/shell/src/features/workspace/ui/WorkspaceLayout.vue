<script setup lang="ts">
/**
 * Tabs, and the ways of dividing the screen between them.
 *
 * The workspace it is given is the whole of what it draws, and a gesture gives
 * back another one. What a tab holds is the caller's: this takes titles and
 * hands identities back.
 */
import { computed, provide, useTemplateRef } from 'vue'
import { browserClock, type Clock } from '@numen/ui'
import { DragPreview } from '@numen/ui'
import { WorkspaceBranch } from './branch'
import { WorkspacePane } from './pane'
import WorkspaceOverlay from './WorkspaceOverlay.vue'
import { WORKSPACE_CONTEXT, type WorkspaceContext } from '../model/context'
import { useTabDrag } from '../model/drag'
import { TAB_HOSTS, createTabHosts } from '../model/hosts'
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
    /** Distance the pointer moves before a press becomes a drag. */
    threshold?: number
    /** The minimum size at which a pane is rendered. */
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
   * A close request, emitted before anything is applied. A caller that calls
   * `hold` while handling the event takes over the close, and the tab stays
   * until the caller closes it.
   */
  (event: 'close', tab: TabId, hold: () => void): void
  (event: 'activate', tab: TabId): void
  /** A context menu requested on a tab, with its screen position. */
  (event: 'tab-context-menu', tab: TabId, pos: { x: number; y: number }): void
  /**
   * The tab a pane now shows, once it is visible. Every tab of a pane is
   * rendered and the inactive ones are hidden, so this event signals the tab's
   * content that it can measure itself.
   */
  (event: 'show', tab: TabId): void
  /** A tab dragged out of the workspace into empty space or a new window. */
  (event: 'tear-off', tab: TabId, pos: { x: number; y: number }): void
}>()

const slots = defineSlots<{
  tab(props: { id: TabId }): unknown
  /** Rendered before a tab's name; identifies the type of tab. */
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

/** Where each tab is drawn: the host element of the pane that holds it. */
const hosts = createTabHosts()
provide(TAB_HOSTS, hosts)

/** The ids this layout creates when the caller supplies no factory. */
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
  onTearOff: (tab, pos) => emit('tear-off', tab, pos),
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

function onMenuTab(tab: TabId, pos: { x: number; y: number }): void {
  emit('tab-context-menu', tab, pos)
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

    <!-- Tab content is rendered here once per tab and placed into the pane that
         holds it; a tab with no pane stays here, hidden. -->
    <div class="workspace__stage" hidden>
      <Teleport
        v-for="tab in tabs"
        :key="tab.id"
        :to="hosts.getElement(tab.id)"
        :disabled="!hosts.getElement(tab.id)"
      >
        <slot name="tab" :id="tab.id" />
      </Teleport>
    </div>

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

.workspace__dragged {
  z-index: var(--numen-lift-tooltip);
}
</style>
