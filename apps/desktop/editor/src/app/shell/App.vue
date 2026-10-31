<script setup lang="ts">
/**
 * The window: one vault, and tabs to divide the screen between.
 *
 * The window's structure is defined in `window.ts`. This file holds what the user
 * sees of it, and the binding between the two.
 */
import { onMounted, onUnmounted } from 'vue'
import { Notices, StatusBar, WorkspaceLayout } from '@numen/shell'
import '@numen/ui/styles.css'
import '@numen/shell/styles.css'
import { UnsavedChangesPrompt } from '@/features/file-conflict'
import CoreFailureNotice from '@/shared/notices/CoreFailureNotice.vue'
import { CommandPalette } from '@/features/command-palette'
import { TabLinkIndicator, TabLinkMenu, useTabLinkMenu } from '@/features/tab-linking'
import { UpdateIndicator } from '@/features/app-update'
import { WelcomeScreen } from '@/pages/welcome'
import { useWindow } from '../window/useWindow'
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
  openCurrentVault,
  isLayoutReady,
  openSettings,
  commands,
  pluginWays,
  commandDeps,
  failure,
  fileFlush,
  windowTabs,
  layout,
  vaultList,
  log,
  notices,
  dismissNotice,
  statusBar,
  palette,
  destinations,
  closeTab,
  getTabIcon,
  getTitle,
  getTarget,
  tabLinks,
  detachTab,
} = useWindow()

const linkMenu = useTabLinkMenu({
  tabLinks,
  getTabs: () => windowTabs.tabs.value,
  getKind: (id) => windowTabs.getTab(id)?.kind.kind,
  isLinkable: (id) => windowTabs.getTab(id)?.kind.navigate !== undefined,
})
const writeMessage = log.getWriter('window')

/* --------------------------------- Hooks ---------------------------------- */
function onOpenSettings(event: Event): void {
  const detail = (event as CustomEvent<{ filter?: string; pluginId?: string }>).detail
  if ((detail?.filter || detail?.pluginId) && openSettings) {
    openSettings(detail.pluginId, detail.filter)
    return
  }
  runCommand('settings', getTarget())
}

onMounted(() => {
  globalThis.addEventListener('numen:open-settings', onOpenSettings)
})

onUnmounted(() => {
  globalThis.removeEventListener('numen:open-settings', onOpenSettings)
})

/* -------------------------------- Handlers -------------------------------- */
function onCloseTab(id: string, hold?: () => void) {
  closeTab(id, hold)
}

async function onTearOffTab(id: string, position: { x: number; y: number }) {
  try {
    await detachTab(id, position)
    onCloseTab(id)
  } catch (err) {
    writeMessage(err instanceof Error ? err.message : String(err), 'error')
  }
}

function onShowTab(id: string) {
  windowTabs.onTabShown(id)
}

function onDismissNotice(id: string) {
  dismissNotice(id)
}

function onTabContextMenu(tabId: string, position: { x: number; y: number }) {
  linkMenu.open(tabId, position)
}
</script>

<template>
  <main>
    <CoreFailureNotice :failure="failure" />

    <WorkspaceLayout
      v-model="layout"
      class="below"
      :tabs="windowTabs.tabs.value"
      @close="onCloseTab"
      @show="onShowTab"
      @tab-context-menu="onTabContextMenu"
      @tear-off="onTearOffTab"
    >
      <template #icon="{ id }">
        <component :is="getTabIcon(id)" v-if="getTabIcon(id)" class="tab-icon" />
        <TabLinkIndicator
          :is-linked="tabLinks.getGroupOf(id) !== null"
          :color-index="tabLinks.getGroupOf(id)?.colorIndex"
        />
      </template>

      <template #tab="{ id }">
        <component
          :is="windowTabs.getTab(id)!.kind.pane"
          v-if="windowTabs.getTab(id)"
          :state="windowTabs.getTab(id)!.state"
        />

        <div v-else />
      </template>

      <template #silence>
        <WelcomeScreen
          v-if="isLayoutReady || !vaultList.showing"
          :vault-list="vaultList"
          :tabs="windowTabs.tabs.value"
          :commands="commands"
          :search="palette"
          :doing="commandDeps"
          :get-target="getTarget"
          :plugin-ways="pluginWays"
          :run-command="runCommand"
          :open-current-vault="openCurrentVault"
        />
        <div v-else />
      </template>
    </WorkspaceLayout>

    <TabLinkMenu
      v-if="linkMenu.tabId.value"
      :tab-id="linkMenu.tabId.value"
      :eligible-tabs="linkMenu.eligibleTabs.value"
      :at="linkMenu.at.value"
      :is-linked="linkMenu.isLinked.value"
      :linked-target-title="linkMenu.linkedTargetTitle.value"
      :get-tab-icon="getTabIcon"
      @link="linkMenu.link"
      @unlink="linkMenu.unlink"
      @close="linkMenu.close"
    />

    <StatusBar
      :tasks="statusBar.tasks.value"
      @cancel="statusBar.onCancelTask"
      @retry="statusBar.onRetryTask"
      @dismiss="statusBar.onDismissTask"
      @action="statusBar.onActionTask"
    >
      <template #trailing>
        <UpdateIndicator />
      </template>
    </StatusBar>

    <Notices
      :notices="notices"
      :name="words.isWorking"
      :dismiss="words.dismiss"
      :more="words.more"
      @dismiss="onDismissNotice"
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
