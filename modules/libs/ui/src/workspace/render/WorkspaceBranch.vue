<script setup lang="ts">
/**
 * A branch, drawn as a row or a column of panels with handles between them.
 *
 * Which one it is comes from the depth. The panels are told the shares the
 * model holds, and a handle that moves reports shares back; the model is the
 * only place they are kept.
 */
import { computed, onBeforeUnmount, ref, useTemplateRef, watch } from 'vue'
import { SplitterGroup, SplitterPanel, SplitterResizeHandle } from 'reka-ui'
import WorkspacePane from './WorkspacePane.vue'
import { orientationAt, type Branch, type NodeId, type Orientation, type TabId } from '../model'
import { atLeast, fit } from '../model/shares'

defineOptions({ name: 'WorkspaceBranch' })

const props = defineProps<{
  node: Branch
  axis: Orientation
  depth: number
  titles: Readonly<Record<TabId, string>>
  marks: Readonly<Record<TabId, string>>
  focus: NodeId
  /** The least room a pane is worth drawing in. */
  minimum: number
  /** Whether the tabs below are offered a way to be closed. */
  closable: boolean
  /** What the way to a new tab is called, for every strip below. */
  newTab?: string | undefined
}>()

const emit = defineEmits<{
  (event: 'choose', tab: TabId): void
  (event: 'close', tab: TabId): void
  (event: 'lift', tab: TabId, at: PointerEvent): void
  (event: 'claim', pane: NodeId): void
  (event: 'open', pane: NodeId): void
  (event: 'resize', branch: NodeId, sizes: readonly number[]): void
  (event: 'show', tab: TabId): void
}>()

/** What a child says, said again unchanged. */
const passed = {
  onChoose: (tab: TabId) => emit('choose', tab),
  onClose: (tab: TabId) => emit('close', tab),
  onLift: (tab: TabId, at: PointerEvent) => emit('lift', tab, at),
  onShow: (tab: TabId) => emit('show', tab),
}

defineSlots<{
  tab(props: { id: TabId }): unknown
  mark(props: { id: TabId; mark: string }): unknown
  silence(): unknown
}>()

const direction = computed(() => orientationAt(props.axis, props.depth))

const sizes = computed(() => fit(props.node.sizes, props.node.children.length))

/**
 * The smallest share a handle may leave a child, worked out from how long the
 * branch is on screen. A splitter counts in percent, and a pane is only worth
 * drawing above a certain number of pixels.
 */
const frame = useTemplateRef<InstanceType<typeof SplitterGroup>>('frame')
const length = ref(0)

const floor = computed(
  () => atLeast(props.minimum, length.value, props.node.children.length) * 100,
)

let watching: ResizeObserver | undefined

// The element is followed: a change of shape gives the pane a new one.
watch(
  () => frame.value?.$el as HTMLElement | undefined,
  (element) => {
    watching?.disconnect()
    if (!element || typeof ResizeObserver === 'undefined') return

    watching = new ResizeObserver(([seen]) => {
      if (!seen) return
      const box = seen.contentRect
      length.value = direction.value === 'horizontal' ? box.width : box.height
    })
    watching.observe(element)
  },
  { immediate: true, flush: 'post' },
)

onBeforeUnmount(() => {
  watching?.disconnect()
  // The branch is going, and where its handle reached is not news about a node
  // that will not be there to hear it.
  reached = null
  putDown()
})

/** Which children there are, so that a change of shape starts the pane afresh. */
const shape = computed(() => props.node.children.map((child) => child.id).join(' '))

/**
 * Whether a handle is under the pointer, and where it has reached. The splitter
 * moves the panels itself while it is held, and the model is told where it came
 * to rest.
 */
let holding = false
let reached: readonly number[] | null = null

/**
 * A splitter reports the shares it settled on, including the ones it worked
 * out for itself when a panel arrived or left. Only a difference is passed on.
 */
function settled(reported: number[]): void {
  const shares = reported.map((size) => size / 100)
  const held = sizes.value
  const same =
    shares.length === held.length &&
    shares.every((share, index) => Math.abs(share - (held[index] ?? 0)) < 1e-6)

  if (same) return
  if (holding) reached = shares
  else emit('resize', props.node.id, shares)
}

/**
 * What a held handle is marked by, on the root so that it reaches everything
 * the drag passes over.
 */
const RESIZING = 'data-resizing'

/** The handle put down, wherever the pointer had reached by then. */
const putDown = () => {
  if (!holding) return
  holding = false
  document.documentElement.removeAttribute(RESIZING)

  const settled = reached
  reached = null
  if (settled) emit('resize', props.node.id, settled)
}

/** A handle taken up, and put down where it stopped. */
function handling(now: boolean): void {
  if (!now) {
    putDown()
    return
  }
  holding = true
  document.documentElement.setAttribute(RESIZING, '')
}
</script>

<template>
  <SplitterGroup
    ref="frame"
    :id="node.id"
    :key="shape"
    class="branch min-h-0 min-w-0"
    :direction="direction"
    @layout="settled"
  >
    <template v-for="(child, index) in node.children" :key="child.id">
      <SplitterResizeHandle
        v-if="index > 0"
        class="branch__handle"
        :data-direction="direction"
        @dragging="handling"
      />

      <SplitterPanel
        :id="child.id"
        :order="index"
        :default-size="(sizes[index] ?? 0) * 100"
        :min-size="floor"
        class="min-h-0 min-w-0"
      >
        <WorkspaceBranch
          v-if="child.kind === 'branch'"
          v-bind="passed"
          :node="child"
          :axis="axis"
          :depth="depth + 1"
          :titles="titles"
          :marks="marks"
          :focus="focus"
          :minimum="minimum"
          :closable="closable"
          :new-tab="newTab"
          @claim="emit('claim', $event)"
          @open="emit('open', $event)"
          @resize="(branch, next) => emit('resize', branch, next)"
        >
          <template #tab="bound"><slot name="tab" v-bind="bound" /></template>
          <template v-if="$slots.mark" #mark="bound"><slot name="mark" v-bind="bound" /></template>
          <template #silence><slot name="silence" /></template>
        </WorkspaceBranch>

        <WorkspacePane
          v-else
          v-bind="passed"
          :pane="child"
          :titles="titles"
          :marks="marks"
          :focused="child.id === focus"
          :closable="closable"
          :new-tab="newTab"
          @claim="emit('claim', child.id)"
          @open="emit('open', child.id)"
        >
          <template #tab="bound"><slot name="tab" v-bind="bound" /></template>
          <template v-if="$slots.mark" #mark="bound"><slot name="mark" v-bind="bound" /></template>
          <template #silence><slot name="silence" /></template>
        </WorkspacePane>
      </SplitterPanel>
    </template>
  </SplitterGroup>
</template>

<style scoped>
.branch {
  block-size: 100%;
}

/* The line between two panels, and the reach around it a pointer is caught by.
   The reach hangs over both panels and stands above them, so a press on either
   side of the line is a press on the handle. */
.branch__handle {
  --line: var(--numen-stroke);
  --reach: 7px;

  position: relative;
  z-index: 1;
  flex: none;
  background: var(--numen-node-border);
}

.branch__handle[data-direction='horizontal'] {
  inline-size: var(--line);
  cursor: col-resize;
}

.branch__handle[data-direction='vertical'] {
  block-size: var(--line);
  cursor: row-resize;
}

.branch__handle::after {
  content: '';
  position: absolute;
  inset-block: 0;
  inset-inline: calc(var(--reach) * -1);
}

.branch__handle[data-direction='vertical']::after {
  inset-inline: 0;
  inset-block: calc(var(--reach) * -1);
}

.branch__handle[data-state='drag'] {
  background: var(--numen-ring);
}

/* A finger is caught from further out than a pointer, at the reach the splitter
   answers a touch from. */
@media (pointer: coarse) {
  .branch__handle {
    --reach: 15px;
  }
}
</style>

<style>
/* While a handle is held, the drag moves the handle and the text under the
   pointer is left alone. */
[data-resizing] * {
  user-select: none !important;
  -webkit-user-select: none !important;
}
</style>
