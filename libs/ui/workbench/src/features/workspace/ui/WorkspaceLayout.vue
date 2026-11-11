<script setup lang="ts">
import { computed, provide, useTemplateRef, type VNode } from 'vue'
import { browserClock } from '@numen/ui'
import { DragPreview, Menu } from '@numen/ui'
import { WorkspaceBranch } from './branch'
import { WorkspacePane } from './pane'
import WorkspaceOverlay from './WorkspaceOverlay.vue'
import type { WorkspaceLayoutProps } from './WorkspaceLayout.types'
import { WORKSPACE_CONTEXT, type WorkspaceContext } from '../model/context'
import { findTopPanes, getStripEdge } from '../lib/edges'
import { useTabDrag } from '../model/drag'
import { TAB_HOSTS, createTabHosts } from '../model/hosts'
import { createBrowserNodeIdFactory } from '../api/randomness'
import { getSlotProps } from '../lib/slotBinding'
import { activateTab, closeTab, focusPane, resizeBranch } from '../lib/edit'
import { type NodeId, type Tab, type TabId, type Workspace } from '../lib/node'

/* --------------------------------- Props ---------------------------------- */
const props = withDefaults(defineProps<WorkspaceLayoutProps>(), {
  edge: 22,
  threshold: 4,
  minimum: 220,
  clock: () => browserClock,
})

const workspace = defineModel<Workspace>({ required: true })

/* --------------------------------- Events --------------------------------- */
const emit = defineEmits<{
  (event: 'close', tab: TabId, hold: () => void): void
  (event: 'activate', tab: TabId): void
  (event: 'tab-context-menu', tab: TabId, pos: { x: number; y: number }): void
  (event: 'show', tab: TabId): void
  (event: 'tear-off', tab: TabId, pos: { x: number; y: number }): void
  (event: 'tab-menu-choose', tab: TabId, id: string): void
  (event: 'tab-menu-dismiss'): void
  (event: 'tab-menu-highlight', id: string | null): void
}>()

const slots = defineSlots<{
  tab(props: { id: TabId }): VNode[]
  icon(props: { id: TabId }): VNode[]
  mark(props: { id: TabId; mark: string }): VNode[]
  silence(): VNode[]
  'tab-menu-icon'(props: { id: string }): VNode[]
}>()

/* --------------------------------- State ---------------------------------- */
const passed = computed(
  () =>
    Object.keys(slots).filter((name) => name !== 'tab-menu-icon') as Exclude<
      keyof typeof slots,
      'tab-menu-icon'
    >[],
)

const findTab = (id: TabId): Tab | undefined => props.tabs.find((tab) => tab.id === id)

const topPanes = computed(() => findTopPanes(workspace.value))

const hosts = createTabHosts()
provide(TAB_HOSTS, hosts)

const createNodeId = createBrowserNodeIdFactory()

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

provide(
  WORKSPACE_CONTEXT,
  computed<WorkspaceContext>(() => ({
    findTab,
    getStripEdge: (pane: NodeId) => getStripEdge(topPanes.value, props.windowEdge, pane),
    focus: workspace.value.focus,
    minimum: props.minimum,
    choose: onChooseTab,
    close: onCloseTab,
    startTabPress: press,
    menu: onMenuTab,
    claim: onClaimPane,
    resize: onResizeBranch,
    show: (tab: TabId) => emit('show', tab),
  })),
)

/* -------------------------------- Handlers -------------------------------- */
function onChooseTab(tab: TabId): void {
  if (hasMoved.value) return
  workspace.value = activateTab(workspace.value, tab)
  emit('activate', tab)
}

function onCloseTab(tab: TabId): void {
  const veto = { isPrevented: false }
  emit('close', tab, () => {
    veto.isPrevented = true
  })
  if (veto.isPrevented) return

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
      @press="press"
      @menu="onMenuTab"
      @show="emit('show', $event)"
      @claim="onClaimPane(workspace.root.id)"
    >
      <template v-for="name in passed" #[name]="slotProps">
        <slot :name="name" v-bind="getSlotProps(slotProps)" />
      </template>
    </WorkspacePane>

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

    <Menu
      v-if="tabMenu"
      :items="tabMenu.items"
      :at="tabMenu.at"
      is-open
      @choose="emit('tab-menu-choose', tabMenu.tab, $event)"
      @dismiss="emit('tab-menu-dismiss')"
      @highlight="emit('tab-menu-highlight', $event)"
    >
      <template v-if="slots['tab-menu-icon']" #icon="{ id }">
        <slot name="tab-menu-icon" :id="id" />
      </template>
    </Menu>

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
