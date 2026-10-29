<script setup lang="ts">
/**
 * A branch, drawn as a row or a column of panels with handles between them.
 *
 * The orientation comes from the depth. The panels receive the shares the model
 * holds, and a handle that moves reports shares back; the model is the only
 * place they are stored.
 */
import { computed, inject, resolveComponent, useTemplateRef, type Component, type Ref } from 'vue'
import { SplitterGroup } from 'reka-ui'
import { BranchPanel } from './panel'
import { WORKSPACE_CONTEXT, type WorkspaceContext } from '../../model/context'
import { getOrientation, type Branch, type Orientation, type TabId } from '../../lib/node'
import { atLeast, fit } from '../../lib/shares'
import { useViewport } from '@numen/ui'
import { useBranchResize } from '../../model/resize'

defineOptions({ name: 'WorkspaceBranch' })

/* --------------------------------- Props ---------------------------------- */
const props = defineProps<{
  node: Branch
  axis: Orientation
  depth: number
}>()

const slots = defineSlots<{
  tab(props: { id: TabId }): unknown
  icon(props: { id: TabId }): unknown
  mark(props: { id: TabId; mark: string }): unknown
  silence(): unknown
}>()

/* ---------------------------------- State --------------------------------- */
/** A branch inside a branch is drawn by this same component, under its own name. */
const branchView = resolveComponent('WorkspaceBranch') as Component

/** What every branch and pane of one workspace receives once, at the top. */
const workspace = inject(WORKSPACE_CONTEXT) as Ref<WorkspaceContext>

/** Every slot handed down, under the name it arrived under. */
const passed = computed(() => Object.keys(slots) as (keyof typeof slots)[])

const direction = computed(() => getOrientation(props.axis, props.depth))

const sizes = computed(() => fit(props.node.sizes, props.node.children.length))

/** What each child receives, in percent. */
const percents = computed(() => sizes.value.map((share) => share * 100))

/**
 * The smallest share a handle may leave a child, worked out from how long the
 * branch is on screen. A splitter counts in percent, and a pane is only worth
 * drawing above a certain number of pixels.
 */
const frame = useTemplateRef<InstanceType<typeof SplitterGroup>>('frame')
const frameElement = computed(() => (frame.value?.$el as HTMLElement | undefined) ?? null)
const { viewport } = useViewport(frameElement)

const length = computed(() =>
  direction.value === 'horizontal' ? viewport.value.width : viewport.value.height,
)

const floor = computed(
  () => atLeast(workspace.value.minimum, length.value, props.node.children.length) * 100,
)

/** Which children exist, so a change of shape resets the pane. */
const shape = computed(() => props.node.children.map((child) => child.id).join(' '))

/* -------------------------------- Handlers -------------------------------- */
const { onLayout, onHold: onSplitterHold } = useBranchResize({
  getBranchId: () => props.node.id,
  getShares: () => sizes.value,
  getFrame: () => frameElement.value ?? undefined,
  resize: (id, shares) => workspace.value.resize(id, shares),
})

/* --------------------------------- Helpers -------------------------------- */
/** What a slot was given, handed on as it came. */
const getSlotProps = (bound: unknown) => (bound ?? {}) as { id: TabId; mark: string }
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
    <BranchPanel
      v-for="(child, index) in node.children"
      :key="child.id"
      :node="child"
      :index="index"
      :share="percents[index] ?? 0"
      :floor="floor"
      :axis="axis"
      :depth="depth"
      :direction="direction"
      :branch-view="branchView"
      @hold="onSplitterHold"
    >
      <template v-for="name in passed" #[name]="slotProps">
        <slot :name="name" v-bind="getSlotProps(slotProps)" />
      </template>
    </BranchPanel>
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
