<script setup lang="ts">
/**
 * A branch, drawn as a row or a column of panels with handles between them.
 *
 * Which one it is comes from the depth. The panels are told the shares the
 * model holds, and a handle that moves reports shares back; the model is the
 * only place they are kept.
 */
import { computed, inject, onBeforeUnmount, ref, useTemplateRef, watch, type Ref } from 'vue'
import { SplitterGroup, SplitterPanel } from 'reka-ui'
import { BranchHandle } from './handle'
import { WorkspacePane } from '../pane'
import { WORKSPACE_CONTEXT, type WorkspaceContext } from '../../model/context'
import { orientationAt, type Branch, type Orientation, type TabId } from '../../lib/node'
import { atLeast, fit } from '../../lib/shares'

defineOptions({ name: 'WorkspaceBranch' })

const props = defineProps<{
  node: Branch
  axis: Orientation
  depth: number
}>()

/** What every branch and pane of one workspace is told once, at the top. */
const workspace = inject(WORKSPACE_CONTEXT) as Ref<WorkspaceContext>

const slots = defineSlots<{
  tab(props: { id: TabId }): unknown
  icon(props: { id: TabId }): unknown
  mark(props: { id: TabId; mark: string }): unknown
  silence(): unknown
}>()

/** Every slot handed down, under the name it arrived under. */
const passed = computed(() => Object.keys(slots) as (keyof typeof slots)[])

/** What a slot was given, handed on as it came. */
const getSlotProps = (bound: unknown) => (bound ?? {}) as { id: TabId; mark: string }

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
  () => atLeast(workspace.value.minimum, length.value, props.node.children.length) * 100,
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
function onLayout(reported: number[]): void {
  const shares = reported.map((size) => size / 100)
  const held = sizes.value
  const same =
    shares.length === held.length &&
    shares.every((share, index) => Math.abs(share - (held[index] ?? 0)) < 1e-6)

  if (same) return
  if (holding) reached = shares
  else workspace.value.resize(props.node.id, shares)
}

/**
 * What a held handle is marked by, on the branch so that it reaches the panels
 * the drag divides.
 */
const RESIZING = 'data-resizing'

/** The branch's own element, which carries that mark. */
const getFrameElement = (): HTMLElement | undefined => frame.value?.$el as HTMLElement | undefined

/** The handle put down, wherever the pointer had reached by then. */
const putDown = () => {
  if (!holding) return
  holding = false
  getFrameElement()?.removeAttribute(RESIZING)

  const settled = reached
  reached = null
  if (settled) workspace.value.resize(props.node.id, settled)
}

/** A handle taken up, and put down where it stopped. */
function setHolding(now: boolean): void {
  if (!now) {
    putDown()
    return
  }
  holding = true
  getFrameElement()?.setAttribute(RESIZING, '')
}
</script>

<template>
  <SplitterGroup
    ref="frame"
    :id="node.id"
    :key="shape"
    class="branch min-h-0 min-w-0"
    :direction="direction"
    @layout="onLayout"
  >
    <template v-for="(child, index) in node.children" :key="child.id">
      <BranchHandle v-if="index > 0" :direction="direction" @hold="setHolding" />

      <SplitterPanel
        :id="child.id"
        :order="index"
        :default-size="(sizes[index] ?? 0) * 100"
        :min-size="floor"
        class="min-h-0 min-w-0"
      >
        <WorkspaceBranch
          v-if="child.kind === 'branch'"
          :node="child"
          :axis="axis"
          :depth="depth + 1"
        >
          <template v-for="name in passed" #[name]="bound">
            <slot :name="name" v-bind="getSlotProps(bound)" />
          </template>
        </WorkspaceBranch>

        <WorkspacePane
          v-else
          :pane="child"
          :focused="child.id === workspace.focus"
          @choose="workspace.choose"
          @close="workspace.close"
          @lift="workspace.lift"
          @show="workspace.show"
          @claim="workspace.claim(child.id)"
        >
          <template v-for="name in passed" #[name]="bound">
            <slot :name="name" v-bind="getSlotProps(bound)" />
          </template>
        </WorkspacePane>
      </SplitterPanel>
    </template>
  </SplitterGroup>
</template>

<style scoped>
.branch {
  block-size: 100%;
}

/* While a handle is held, the drag moves the handle and the text under the
   pointer is left alone. Anything that makes itself selectable reads the mark
   and says so itself. */
.branch[data-resizing] {
  user-select: none;
  -webkit-user-select: none;
}
</style>
