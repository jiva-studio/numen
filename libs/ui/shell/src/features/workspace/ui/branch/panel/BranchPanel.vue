<script setup lang="ts">
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
  (event: 'hold', now: boolean): void
}>()

defineSlots<{
  tab(props: { id: TabId }): unknown
  icon(props: { id: TabId }): unknown
  mark(props: { id: TabId; mark: string }): unknown
  silence(): unknown
}>()

/* ---------------------------------- State --------------------------------- */
const workspace = inject(WORKSPACE_CONTEXT) as Ref<WorkspaceContext>

type BranchSlotName = 'tab' | 'icon' | 'mark' | 'silence'

const slots = useSlots()

const passed = computed(() => Object.keys(slots) as BranchSlotName[])

/* --------------------------------- Helpers -------------------------------- */
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
      @press="workspace.startTabPress"
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
