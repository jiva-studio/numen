<script setup lang="ts">
/**
 * One child of a branch in the panel it is given, with the handle that divides
 * it from the child before it.
 *
 * A child that is a pane is drawn as a pane. A child that is itself a branch is
 * drawn with what the branch above handed down, which is the branch itself: a
 * panel knows a node has children and nothing else about a branch.
 */
import { inject, type Component, type Ref } from 'vue'
import { SplitterPanel } from 'reka-ui'
import { BranchHandle } from '../handle'
import { WorkspacePane } from '../../pane'
import { WORKSPACE_CONTEXT, type WorkspaceContext } from '../../../model/context'
import type { Orientation, TabId, WorkspaceNode } from '../../../lib/node'

defineProps<{
  node: WorkspaceNode
  /** Where the child stands among its siblings; the first is given no handle. */
  index: number
  /** The share of the branch the panel opens at, in percent. */
  share: number
  /** The smallest share a handle may leave it, in percent. */
  floor: number
  axis: Orientation
  depth: number
  /** Which way the branch divides its length. */
  direction: Orientation
  /** What a child with children of its own is drawn with. */
  branchView: Component
}>()

defineEmits<{
  /** The handle before this panel taken up, and put down again. */
  (event: 'hold', now: boolean): void
}>()

const slots = defineSlots<{
  tab(props: { id: TabId }): unknown
  icon(props: { id: TabId }): unknown
  mark(props: { id: TabId; mark: string }): unknown
  silence(): unknown
}>()

/** What every branch and pane of one workspace is told once, at the top. */
const workspace = inject(WORKSPACE_CONTEXT) as Ref<WorkspaceContext>

/** Every slot handed down, under the name it arrived under. */
const passed = Object.keys(slots) as (keyof typeof slots)[]

/** What a slot was given, handed on as it came. */
const getSlotProps = (bound: unknown) => (bound ?? {}) as { id: TabId; mark: string }
</script>

<template>
  <BranchHandle v-if="index > 0" :direction="direction" @hold="$emit('hold', $event)" />

  <SplitterPanel
    :id="node.id"
    :order="index"
    :default-size="share"
    :min-size="floor"
    class="min-h-0 min-w-0"
  >
    <component
      :is="branchView"
      v-if="node.kind === 'branch'"
      :node="node"
      :axis="axis"
      :depth="depth + 1"
    >
      <template v-for="name in passed" #[name]="bound">
        <slot :name="name" v-bind="getSlotProps(bound)" />
      </template>
    </component>

    <WorkspacePane
      v-else
      :pane="node"
      :focused="node.id === workspace.focus"
      @choose="workspace.choose"
      @close="workspace.close"
      @lift="workspace.lift"
      @show="workspace.show"
      @claim="workspace.claim(node.id)"
    >
      <template v-for="name in passed" #[name]="bound">
        <slot :name="name" v-bind="getSlotProps(bound)" />
      </template>
    </WorkspacePane>
  </SplitterPanel>
</template>
