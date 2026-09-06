<script setup lang="ts">
/**
 * Tabs, and the ways of dividing the screen between them.
 *
 * The workspace it is given is the whole of what it draws, and a gesture gives
 * back another one. What a tab holds is the caller's: this takes titles and
 * hands identities back.
 */
import { computed, provide, useTemplateRef } from 'vue'
import { browserClock, type Clock } from '../lib/clock'
import DragPreview from '../press/DragPreview.vue'
import { usePressDrag } from '../press/press'
import WorkspaceBranch from './render/WorkspaceBranch.vue'
import WorkspacePane from './render/WorkspacePane.vue'
import { WORKSPACE_CONTEXT, type WorkspaceContext } from './render/context'
import {
  activateTab,
  closeTab,
  dropOnEdge,
  dropTab,
  focusPane,
  moveTabWithin,
  resizeBranch,
  type NodeIdFactory,
} from './edit'
import { boxOf, caretAt, edgeOf, overlayFor, sideAt, slotAt, type TabLanding } from './drop'
import { type NodeId, type Tab, type TabId, type Workspace } from './node'
import type { Rect } from './rect'

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
const handedOn = (bound: unknown) => (bound ?? {}) as { id: TabId; mark: string }

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

const tabOf = (id: TabId): Tab | undefined => props.tabs.find((tab) => tab.id === id)

/**
 * A node's identity reaches the DOM as the id of a splitter panel, so it is
 * unique to the document and not only to this tree.
 */
let made = 0
const mint = (): NodeId =>
  typeof crypto?.randomUUID === 'function' ? crypto.randomUUID() : `node-${++made}-${Date.now()}`

const naming = computed<NodeIdFactory>(() => props.naming ?? mint)

const frame = useTemplateRef<HTMLElement>('frame')

/** A tab under the pointer, and the pane its strip belongs to. */
interface Drag {
  readonly tab: TabId
  readonly from: NodeId
}

const {
  dragging,
  at: landing,
  position,
  lift,
} = usePressDrag<Drag, TabLanding>({
  threshold: () => props.threshold,
  clock: () => props.clock,
  landingAt: (_held, at) => landingAt(at.x, at.y),
  settle: (held, at) => {
    if (at) land(held, at)
  },
})

const overlay = computed(() => (dragging.value?.moved ? (landing.value?.box ?? null) : null))

const label = computed(() => {
  const held = dragging.value
  if (!held?.moved) return null
  return tabOf(held.held.tab)?.title ?? held.held.tab
})

function choose(tab: TabId): void {
  if (dragging.value?.moved) return
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

function press(tab: TabId, at: PointerEvent): void {
  if (at.button !== 0) return

  const holder = document.elementFromPoint(at.clientX, at.clientY)?.closest('[data-workspace-pane]')
  const from = holder?.getAttribute('data-workspace-pane') ?? workspace.value.focus

  lift({ tab, from }, at)
}

function land(held: Drag, at: TabLanding): void {
  const ids = naming.value

  if (at.kind === 'edge') {
    workspace.value = dropOnEdge(workspace.value, held.tab, at.side, ids)
    return
  }

  if (at.kind === 'pane') {
    workspace.value = dropTab(workspace.value, { tab: held.tab, onto: at.pane, side: at.side }, ids)
    return
  }

  const joined =
    at.pane === held.from
      ? workspace.value
      : dropTab(workspace.value, { tab: held.tab, onto: at.pane, side: 'center' }, ids)
  workspace.value = moveTabWithin(joined, held.tab, at.slot)
}

/**
 * What the pointer is over, asked of the document.
 *
 * A strip is read first, so that a tab can be put in order among its
 * neighbours; then the outer edge, which divides the whole workspace; then the
 * pane, which divides itself.
 */
function landingAt(x: number, y: number): TabLanding | null {
  const held = frame.value
  if (!held) return null

  const outer = held.getBoundingClientRect()
  if (x < outer.left || x > outer.right || y < outer.top || y > outer.bottom) return null

  const local = (box: Rect): Rect => ({ ...box, x: box.x - outer.left, y: box.y - outer.top })
  const under = document.elementFromPoint(x, y)

  const strip = under?.closest('[data-workspace-strip]')
  const pane = under?.closest('[data-workspace-pane]')
  const id = pane?.getAttribute('data-workspace-pane')

  if (strip && id) {
    // A strip holds tabs and nothing else, in the order they are drawn.
    const tabs = [...strip.children].map((tab) => boxOf(tab))
    const slot = slotAt(x, tabs)
    return { kind: 'strip', pane: id, slot, box: local(caretAt(slot, tabs, boxOf(strip))) }
  }

  const side = edgeOf({ x, y }, boxOf(held), props.edge)
  if (side) return { kind: 'edge', side, box: local(overlayFor(side, boxOf(held))) }

  if (!pane || !id) return null

  const box = boxOf(pane)
  const asked = sideAt({ x, y }, box)
  return { kind: 'pane', pane: id, side: asked, box: local(overlayFor(asked, box)) }
}

</script>

<template>
  <div ref="frame" class="workspace numen relative min-h-0 min-w-0 bg-surface text-ink">
    <WorkspaceBranch
      v-if="workspace.root.kind === 'branch'"
      :node="workspace.root"
      :axis="workspace.axis"
      :depth="0"
    >
      <template v-for="name in passed" #[name]="bound">
        <slot :name="name" v-bind="handedOn(bound)" />
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
        <slot :name="name" v-bind="handedOn(bound)" />
      </template>
    </WorkspacePane>

    <div
      v-if="overlay"
      class="workspace__overlay"
      :data-caret="landing?.kind === 'strip' || undefined"
      :style="{
        left: `${overlay.x}px`,
        top: `${overlay.y}px`,
        width: `${overlay.width}px`,
        height: `${overlay.height}px`,
      }"
    />

    <DragPreview
      v-if="label && position"
      class="workspace__dragged"
      :at="position"
      :says="label"
    />
  </div>
</template>

<style scoped>
.workspace {
  block-size: 100%;
  overflow: hidden;
}

/* Where the tab would go, shown over everything and catching nothing. The
   wash is the same colour as the outline, laid on thinly. */
.workspace__overlay {
  /* How heavily the wash inside the outline is laid on. */
  --wash: 0.16;

  position: absolute;
  z-index: 2;
  pointer-events: none;
  border: var(--numen-ring-width) solid var(--numen-ring);
  border-radius: var(--numen-radius);
}

.workspace__overlay::before {
  content: '';
  position: absolute;
  inset: 0;
  background: var(--numen-ring);
  opacity: var(--wash);
}

.workspace__overlay[data-caret] {
  inline-size: var(--numen-caret);
  margin-inline-start: calc(var(--numen-caret) / -2);
  border: none;
  background: var(--numen-ring);
}

/* The tab being dragged stands over the panes and the overlay both. */
.workspace__dragged {
  z-index: var(--numen-lift-tooltip);
}
</style>
