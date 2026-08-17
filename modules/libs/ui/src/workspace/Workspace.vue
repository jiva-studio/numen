<script setup lang="ts">
/**
 * Tabs, and the ways of dividing the screen between them.
 *
 * The workspace it is given is the whole of what it draws, and a gesture gives
 * back another one. What a tab holds is the caller's: this takes titles and
 * hands identities back.
 */
import { computed, onBeforeUnmount, shallowRef, useTemplateRef } from 'vue'
import WorkspaceBranch from './render/WorkspaceBranch.vue'
import WorkspaceGroup from './render/WorkspaceGroup.vue'
import {
  activateTab,
  closeTab,
  dropOnEdge,
  dropTab,
  focusGroup,
  moveTabWithin,
  resizeBranch,
  type Naming,
} from './edit'
import { overlayFor, sideAt, slotAt } from './drop'
import type { NodeId, Rect, Side, TabId, TabLabel, Workspace } from './model'

const props = withDefaults(
  defineProps<{
    tabs: readonly TabLabel[]
    /** Where identities for what a gesture makes come from. */
    naming?: Naming | undefined
    /** How close to the outer edge divides the whole workspace. */
    edge?: number
    /** How far the pointer travels before a press becomes a drag. */
    threshold?: number
    /** The least room a group is worth drawing in. */
    minimum?: number
  }>(),
  { edge: 22, threshold: 4, minimum: 220 },
)

defineSlots<{
  tab(props: { id: TabId }): unknown
  silence(): unknown
}>()

const workspace = defineModel<Workspace>({ required: true })

const emit = defineEmits<{
  (event: 'close', tab: TabId): void
  (event: 'activate', tab: TabId): void
}>()

const titles = computed(() =>
  Object.fromEntries(props.tabs.map((tab) => [tab.id, tab.title])),
)

let made = 0
const naming = computed<Naming>(() => props.naming ?? { id: () => `node-${++made}` })

const frame = useTemplateRef<HTMLElement>('frame')

/** A tab under the pointer, once the pointer has gone far enough to mean it. */
interface Dragging {
  readonly tab: TabId
  readonly from: NodeId
  readonly startX: number
  readonly startY: number
  readonly x: number
  readonly y: number
  readonly moved: boolean
}

/** Where the tab would go if it were let go now. */
type Landing =
  | { readonly kind: 'edge'; readonly side: Side; readonly box: Rect }
  | { readonly kind: 'group'; readonly group: NodeId; readonly side: Side; readonly box: Rect }
  | { readonly kind: 'strip'; readonly group: NodeId; readonly slot: number; readonly box: Rect }

const dragging = shallowRef<Dragging | null>(null)
const landing = shallowRef<Landing | null>(null)

const overlay = computed(() => (dragging.value?.moved ? (landing.value?.box ?? null) : null))

const carried = computed(() =>
  dragging.value?.moved ? (titles.value[dragging.value.tab] ?? dragging.value.tab) : null,
)

function choose(tab: TabId): void {
  if (dragging.value?.moved) return
  workspace.value = activateTab(workspace.value, tab)
  emit('activate', tab)
}

function close(tab: TabId): void {
  workspace.value = closeTab(workspace.value, tab)
  emit('close', tab)
}

function claim(group: NodeId): void {
  workspace.value = focusGroup(workspace.value, group)
}

function resize(branch: NodeId, sizes: readonly number[]): void {
  workspace.value = resizeBranch(workspace.value, branch, sizes)
}

function lift(tab: TabId, at: PointerEvent): void {
  if (at.button !== 0) return

  const holder = document.elementFromPoint(at.clientX, at.clientY)?.closest('[data-workspace-group]')
  const from = holder?.getAttribute('data-workspace-group') ?? workspace.value.focus

  dragging.value = {
    tab,
    from,
    startX: at.clientX,
    startY: at.clientY,
    x: at.clientX,
    y: at.clientY,
    moved: false,
  }
  window.addEventListener('pointermove', drag)
  window.addEventListener('pointerup', drop)
  window.addEventListener('pointercancel', drop)
}

function drag(at: PointerEvent): void {
  const held = dragging.value
  if (!held) return

  const moved =
    held.moved ||
    Math.abs(at.clientX - held.startX) > props.threshold ||
    Math.abs(at.clientY - held.startY) > props.threshold

  dragging.value = { ...held, x: at.clientX, y: at.clientY, moved }
  landing.value = moved ? landingAt(at.clientX, at.clientY) : null
}

function drop(): void {
  const held = dragging.value
  const at = landing.value

  window.removeEventListener('pointermove', drag)
  window.removeEventListener('pointerup', drop)
  window.removeEventListener('pointercancel', drop)

  if (held?.moved && at) land(held, at)
  landing.value = null
  // Held one frame longer: the click that follows the release reads it and
  // stands down.
  requestAnimationFrame(() => {
    dragging.value = null
  })
}

function land(held: Dragging, at: Landing): void {
  const ids = naming.value

  if (at.kind === 'edge') {
    workspace.value = dropOnEdge(workspace.value, held.tab, at.side, ids)
    return
  }

  if (at.kind === 'group') {
    workspace.value = dropTab(workspace.value, { tab: held.tab, onto: at.group, side: at.side }, ids)
    return
  }

  const joined =
    at.group === held.from
      ? workspace.value
      : dropTab(workspace.value, { tab: held.tab, onto: at.group, side: 'center' }, ids)
  workspace.value = moveTabWithin(joined, held.tab, at.slot)
}

/**
 * What the pointer is over, asked of the document.
 *
 * A strip is read first, so that a tab can be put in order among its
 * neighbours; then the outer edge, which divides the whole workspace; then the
 * group, which divides itself.
 */
function landingAt(x: number, y: number): Landing | null {
  const held = frame.value
  if (!held) return null

  const outer = held.getBoundingClientRect()
  if (x < outer.left || x > outer.right || y < outer.top || y > outer.bottom) return null

  const local = (box: Rect): Rect => ({ ...box, x: box.x - outer.left, y: box.y - outer.top })
  const under = document.elementFromPoint(x, y)

  const strip = under?.closest('[data-workspace-strip]')
  const group = under?.closest('[data-workspace-group]')
  const id = group?.getAttribute('data-workspace-group')

  if (strip && id) {
    const tabs = [...strip.querySelectorAll('[data-workspace-tab]')].map((tab) => boxOf(tab))
    const slot = slotAt(x, tabs)
    return { kind: 'strip', group: id, slot, box: local(caretAt(slot, tabs, strip)) }
  }

  const side = edgeOf(x, y, outer, props.edge)
  if (side) return { kind: 'edge', side, box: local(overlayFor(side, boxOf(held))) }

  if (!group || !id) return null

  const box = boxOf(group)
  const asked = sideAt({ x, y }, box)
  return { kind: 'group', group: id, side: asked, box: local(overlayFor(asked, box)) }
}

/** The gap a tab would take, along the strip. Its width is drawn in CSS. */
function caretAt(slot: number, tabs: readonly Rect[], strip: Element): Rect {
  const beside = tabs[slot] ?? tabs[tabs.length - 1]
  if (!beside) return boxOf(strip)

  const at = slot < tabs.length ? beside.x : beside.x + beside.width
  return { x: at, y: beside.y, width: 0, height: beside.height }
}

const boxOf = (element: Element): Rect => {
  const box = element.getBoundingClientRect()
  return { x: box.left, y: box.top, width: box.width, height: box.height }
}

/** The outer edge a point is within reach of. */
function edgeOf(x: number, y: number, box: DOMRect, reach: number): Side | null {
  const near: readonly (readonly [Side, number])[] = [
    ['left', x - box.left],
    ['right', box.right - x],
    ['top', y - box.top],
    ['bottom', box.bottom - y],
  ]

  let side: Side | null = null
  let nearest = reach
  for (const [each, distance] of near) {
    if (distance < nearest) {
      side = each
      nearest = distance
    }
  }
  return side
}

onBeforeUnmount(() => {
  window.removeEventListener('pointermove', drag)
  window.removeEventListener('pointerup', drop)
  window.removeEventListener('pointercancel', drop)
})
</script>

<template>
  <div ref="frame" class="workspace numen relative min-h-0 min-w-0 bg-surface text-ink">
    <WorkspaceBranch
      v-if="workspace.root.kind === 'branch'"
      :node="workspace.root"
      :axis="workspace.axis"
      :depth="0"
      :titles="titles"
      :focus="workspace.focus"
      :minimum="minimum"
      @choose="choose"
      @close="close"
      @lift="lift"
      @claim="claim"
      @resize="resize"
    >
      <template #tab="bound"><slot name="tab" v-bind="bound" /></template>
      <template #silence><slot name="silence" /></template>
    </WorkspaceBranch>

    <WorkspaceGroup
      v-else
      :group="workspace.root"
      :titles="titles"
      :focused="workspace.root.id === workspace.focus"
      @choose="choose"
      @close="close"
      @lift="lift"
      @claim="claim(workspace.root.id)"
    >
      <template #tab="bound"><slot name="tab" v-bind="bound" /></template>
      <template #silence><slot name="silence" /></template>
    </WorkspaceGroup>

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

    <p
      v-if="carried"
      class="workspace__carried font-sans text-small"
      :style="{ left: `${(dragging?.x ?? 0) + 12}px`, top: `${(dragging?.y ?? 0) + 12}px` }"
    >
      {{ carried }}
    </p>
  </div>
</template>

<style scoped>
.workspace {
  /* The line standing in the gap a dragged tab would take. */
  --caret: 2px;

  block-size: 100%;
  overflow: hidden;
}

/* Where the tab would go, shown over everything and catching nothing. The
   wash is the same colour as the outline, laid on thinly. */
.workspace__overlay {
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
  opacity: 0.16;
}

.workspace__overlay[data-caret] {
  inline-size: var(--caret);
  margin-inline-start: calc(var(--caret) / -2);
  border: none;
  background: var(--numen-ring);
}

/* What is being carried, said beside the pointer. */
.workspace__carried {
  position: fixed;
  z-index: 3;
  margin: 0;
  padding: 0.15rem 0.5rem;
  pointer-events: none;
  border-radius: var(--numen-radius);
  background: var(--numen-focus-bg);
  color: var(--numen-focus-fg);
}
</style>
