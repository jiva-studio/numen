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
import { activateTab, closeTab, focusPane, resizeBranch, type NodeIdFactory } from '../lib/edit'
import { type NodeId, type Tab, type TabId, type Workspace } from '../lib/node'

const props = withDefaults(
  defineProps<{
    tabs: readonly Tab[]
    /** Where identities for what a gesture makes come from. */
    naming?: NodeIdFactory | undefined
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

const emit = defineEmits<{
  /**
   * A close asked for, before anything is applied. A caller that calls `hold`
   * while it is being told takes the close, and the tab stays until the
   * caller closes it itself.
   */
  (event: 'close', tab: TabId, hold: () => void): void
  (event: 'activate', tab: TabId): void
  /**
   * The tab a pane is now showing, once it is on screen. Every tab of a pane is
   * drawn and the ones not shown are held out of sight, so what a tab holds is
   * told here that it can measure itself.
   */
  (event: 'show', tab: TabId): void
}>()

const slots = defineSlots<{
  tab(props: { id: TabId }): unknown
  /** What is drawn before a tab's name, which says what kind of tab it is. */
  icon(props: { id: TabId }): unknown
  /** What a mark is drawn as. Given none, a tab carrying one draws a dot. */
  mark(props: { id: TabId; mark: string }): unknown
  silence(): unknown
}>()

/** Every slot the caller gave, handed down under the name it was given. */
const passed = computed(() => Object.keys(slots) as (keyof typeof slots)[])

/** What a slot was given, handed on as it came. */
const getSlotProps = (bound: unknown) => (bound ?? {}) as { id: TabId; mark: string }

const tabOf = (id: TabId): Tab | undefined => props.tabs.find((tab) => tab.id === id)

/**
 * A node's identity reaches the DOM as the id of a splitter panel, so it is
 * unique to the document and not only to this tree.
 */
let made = 0
const mint = (): NodeId =>
  typeof crypto?.randomUUID === 'function' ? crypto.randomUUID() : `node-${++made}-${Date.now()}`

const frame = useTemplateRef<HTMLElement>('frame')

const { moved, overlay, label, position, landing, press } = useTabDrag({
  workspace,
  frame,
  getTab: tabOf,
  getIdFactory: () => props.naming ?? mint,
  getEdge: () => props.edge,
  getThreshold: () => props.threshold,
  getClock: () => props.clock,
})

function choose(tab: TabId): void {
  if (moved.value) return
  workspace.value = activateTab(workspace.value, tab)
  emit('activate', tab)
}

function close(tab: TabId): void {
  let held = false
  emit('close', tab, () => {
    held = true
  })
  if (held) return

  workspace.value = closeTab(workspace.value, tab)
}

function claim(pane: NodeId): void {
  workspace.value = focusPane(workspace.value, pane)
}

function resize(branch: NodeId, sizes: readonly number[]): void {
  workspace.value = resizeBranch(workspace.value, branch, sizes)
}

provide(
  WORKSPACE_CONTEXT,
  computed<WorkspaceContext>(() => ({
    tabOf,
    focus: workspace.value.focus,
    minimum: props.minimum,
    choose,
    close,
    lift: press,
    claim,
    resize,
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
      <template v-for="name in passed" #[name]="bound">
        <slot :name="name" v-bind="getSlotProps(bound)" />
      </template>
    </WorkspaceBranch>

    <WorkspacePane
      v-else
      :pane="workspace.root"
      :focused="workspace.root.id === workspace.focus"
      @choose="choose"
      @close="close"
      @lift="press"
      @show="emit('show', $event)"
      @claim="claim(workspace.root.id)"
    >
      <template v-for="name in passed" #[name]="bound">
        <slot :name="name" v-bind="getSlotProps(bound)" />
      </template>
    </WorkspacePane>

    <WorkspaceOverlay v-if="overlay" :box="overlay" :is-caret-line="landing?.kind === 'strip'" />

    <DragPreview v-if="label && position" class="workspace__dragged" :at="position" :says="label" />
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
