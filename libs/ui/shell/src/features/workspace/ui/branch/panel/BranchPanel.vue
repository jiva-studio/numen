<script setup lang="ts">
/**
 * One child of a branch in the panel it is given, with the handle that divides
 * it from the child before it.
 *
 * A child that is a pane is rendered as a pane. A child that is itself a branch
 * is rendered from what the parent branch passes down, which is the branch
 * itself: a panel depends only on a node having children.
 */
import { computed, inject, useSlots, type Ref } from 'vue'
import { SplitterPanel } from 'reka-ui'
import { BranchHandle } from '../handle'
import { WorkspacePane } from '../../pane'
import { WORKSPACE_CONTEXT, type WorkspaceContext } from '../../../model/context'
import type { TabId } from '../../../lib/node'
import type { BranchPanelProps } from './BranchPanel.types'

/* --------------------------------- Props ---------------------------------- */
defineProps<BranchPanelProps>()

/* --------------------------------- Events --------------------------------- */
defineEmits<{
  /** The handle before this panel taken up, and put down again. */
  (event: 'hold', now: boolean): void
}>()

defineSlots<{
  tab(props: { id: TabId }): unknown
  icon(props: { id: TabId }): unknown
  mark(props: { id: TabId; mark: string }): unknown
  silence(): unknown
}>()

/* ---------------------------------- State --------------------------------- */
/** Context provided once at the top and shared by every branch and pane of one workspace. */
const workspace = inject(WORKSPACE_CONTEXT) as Ref<WorkspaceContext>

type BranchSlotName = 'tab' | 'icon' | 'mark' | 'silence'

const slots = useSlots()

/** Every slot handed down, under the name it arrived under. */
const passed = computed(() => Object.keys(slots) as BranchSlotName[])

/* --------------------------------- Helpers -------------------------------- */
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
      <template v-for="name in passed" #[name]="slotProps">
        <slot :name="name" v-bind="getSlotProps(slotProps)" />
      </template>
    </component>

    <WorkspacePane
      v-else
      :pane="node"
      :is-focused="node.id === workspace.focus"
      @choose="workspace.choose"
      @close="workspace.close"
      @lift="workspace.lift"
      @menu="workspace.menu"
      @show="workspace.show"
      @claim="workspace.claim(node.id)"
    >
      <template v-for="name in passed" #[name]="slotProps">
        <slot :name="name" v-bind="getSlotProps(slotProps)" />
      </template>
    </WorkspacePane>
  </SplitterPanel>
</template>
