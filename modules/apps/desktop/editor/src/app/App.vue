<script setup lang="ts">
/**
 * The window: one vault, and tabs to divide the screen between.
 *
 * What the window is made of is `window.ts`. What is here is what a person
 * sees of it, and the binding between the two.
 */
import { Notices, StatusBar, WorkspaceLayout } from '@numen/ui'
import '@numen/ui/styles.css'
import './app.css'
import { computed, ref } from 'vue'
import { UnsavedChangesPrompt } from '@/features/file-conflict'
import CoreFailureNotice from '@/shared/notices/CoreFailureNotice.vue'
import { CommandPalette } from '@/features/command-palette'
import { TabLinkIndicator, TabLinkMenu } from '@/features/tab-linking'
import { WelcomeScreen } from '@/pages/welcome'
import { useWindow } from './useWindow'
import { WORDS as words } from '@/shared/words'
import { WORDS as note } from '@/entities/note'

/* --------------------------------- State ---------------------------------- */
/** What the notes still unwritten are put in: the window's words and a note's. */
const unsaved = {
  going: words.going,
  keep: note.keep,
  take: note.take,
  later: words.later,
}

const {
  runCommand,
  commands,
  commandDeps,
  failure,
  fileFlush,
  held,
  layout,
  listed,
  log,
  notices,
  statusBar,
  palette,
  destinations,
  closeTab,
  tabIcon,
  getTitle,
  getTarget,
  tabLinks,
} = useWindow()

const activeMenuTabId = ref<string | null>(null)
const menuPosition = ref<{ x: number; y: number }>({ x: 0, y: 0 })

const eligibleTabs = computed(() => {
  if (!activeMenuTabId.value) return []
  const group = tabLinks.getGroupOf(activeMenuTabId.value)
  return held.tabs.value
    .filter((t) => {
      if (t.id === activeMenuTabId.value) return false
      if (group && group.tabs.includes(t.id)) return false
      const kind = held.getTab(t.id)?.kind.kind
      return kind === 'note' || kind === 'files' || kind === 'plex'
    })
    .map((t) => ({
      id: t.id,
      title: t.title,
      kind: held.getTab(t.id)?.kind.kind ?? '',
      isLinked: tabLinks.getGroupOf(t.id) !== null,
    }))
})

const activeLinkedTargetTitle = computed(() => {
  if (!activeMenuTabId.value) return ''
  const targets = tabLinks.getLinkedTargets(activeMenuTabId.value)
  if (targets.length === 0) return ''
  return targets.map((id) => held.tabs.value.find((t) => t.id === id)?.title || id).join(', ')
})

/* -------------------------------- Handlers -------------------------------- */
function onCloseTab(id: string) {
  closeTab(id)
}

function onShowTab(id: string) {
  held.onTabShown(id)
}

function dismissNotice(id: string) {
  log.dismiss(id)
}

function onOpenLinkMenu(tabId: string, event: PointerEvent | MouseEvent) {
  activeMenuTabId.value = tabId
  menuPosition.value = { x: event.clientX, y: event.clientY }
}

function onTabContextMenu(event: MouseEvent) {
  const target = (event.target as HTMLElement).closest('[data-workspace-tab]') as HTMLElement | null
  const tabId = target?.getAttribute('data-workspace-tab')
  if (tabId) {
    const kind = held.getTab(tabId)?.kind.kind
    if (kind === 'note' || kind === 'files' || kind === 'plex') {
      event.preventDefault()
      onOpenLinkMenu(tabId, event)
    }
  }
}

function onLinkTab(targetId: string) {
  if (activeMenuTabId.value) {
    tabLinks.linkTabs(activeMenuTabId.value, targetId)
    activeMenuTabId.value = null
  }
}

function onUnlinkTab() {
  if (activeMenuTabId.value) {
    tabLinks.unlinkTab(activeMenuTabId.value)
    activeMenuTabId.value = null
  }
}

function onCloseLinkMenu() {
  activeMenuTabId.value = null
}
</script>

<template>
  <main>
    <CoreFailureNotice :failure="failure" />

    <WorkspaceLayout
      v-model="layout"
      class="below"
      :tabs="held.tabs.value"
      @close="onCloseTab"
      @show="onShowTab"
      @contextmenu="onTabContextMenu"
    >
      <template #icon="{ id }">
        <component :is="tabIcon(id)" v-if="tabIcon(id)" class="tab-icon" />
        <TabLinkIndicator
          :is-linked="tabLinks.getGroupOf(id) !== null"
          :color-index="tabLinks.getGroupOf(id)?.colorIndex"
        />
      </template>

      <template #tab="{ id }">
        <component
          :is="held.getTab(id)!.kind.pane"
          v-if="held.getTab(id)"
          :state="held.getTab(id)!.state"
        />

        <div v-else />
      </template>

      <template #silence>
        <WelcomeScreen
          :listed="listed"
          :tabs="held.tabs.value"
          :commands="commands"
          :search="palette"
          :doing="commandDeps"
          :get-target="getTarget"
          :run-command="runCommand"
        />
      </template>
    </WorkspaceLayout>

    <TabLinkMenu
      v-if="activeMenuTabId"
      :tab-id="activeMenuTabId"
      :eligible-tabs="eligibleTabs"
      :at="menuPosition"
      :is-linked="tabLinks.getGroupOf(activeMenuTabId) !== null"
      :linked-target-title="activeLinkedTargetTitle"
      @link="onLinkTab"
      @unlink="onUnlinkTab"
      @close="onCloseLinkMenu"
    />

    <StatusBar
      :tasks="statusBar.tasks.value"
      @cancel="statusBar.onCancelTask"
      @retry="statusBar.onRetryTask"
      @dismiss="statusBar.onDismissTask"
    />

    <Notices
      :notices="notices"
      :name="words.isWorking"
      :dismiss="words.dismiss"
      :more="words.more"
      @dismiss="dismissNotice"
    />

    <UnsavedChangesPrompt
      :conflicts="fileFlush.conflicts.value"
      :called="getTitle"
      :words="unsaved"
    />

    <CommandPalette
      :commands="commands"
      :search="palette"
      :doing="commandDeps"
      :get-target="getTarget"
      :places="destinations"
    />
  </main>
</template>

<style scoped>
main {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

.below {
  flex: 1;
  min-height: 0;
}

/* Lucide draws on a 24 grid, and the stroke is given in those units. */
.tab-icon {
  inline-size: 100%;
  block-size: 100%;
  stroke-width: 1.75;
  opacity: 0.75;
}
</style>
