<script setup lang="ts">
/**
 * The window: one vault, and tabs to divide the screen between.
 *
 * The window's structure is defined in `window.ts`. This file holds what the user
 * sees of it, and the binding between the two.
 */
import { onMounted, onUnmounted } from 'vue'
import { WorkspaceLayout } from '@numen/shell'
import { LoadingIndicator } from '@numen/ui'
import '@numen/ui/styles.css'
import '@numen/shell/styles.css'
import { UnsavedChangesPrompt } from '@/features/file-conflict'
import CoreFailureNotice from '@/shared/notices/CoreFailureNotice.vue'
import { CommandPalette } from '@/features/command-palette'
import { TabLinkIndicator, TabLinkMenu, useTabLinkMenu } from '@/features/tab-linking'
import { StatusBarHost } from '@/features/status-bar-view'
import { TitleBarHost } from '@/features/title-bar-view'
import { WelcomeScreen } from '@/pages/welcome'
import BootstrapFailure from './BootstrapFailure.vue'
import { useWindow } from '../window/useWindow'
import { useWindowFrame } from '../window/useWindowFrame'
import { useWindowBar } from '../window/useWindowBar'
import { COMMAND } from '@/shared/commandIds'
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
  menus,
  runChoice,
  openCurrentVault,
  isLayoutReady,
  bootstrapFailure,
  isStarted,
  titleBarView,
  retryBootstrap,
  openSettings,
  commands,
  pluginWays,
  commandDeps,
  failure,
  fileFlush,
  windowTabs,
  layout,
  vaultList,
  statusBarView,
  palette,
  destinations,
  closeTab,
  getTabIcon,
  getKindIcon,
  getTitle,
  getTarget,
  tabLinks,
  windowId,
  isHighlighted,
  highlightWindow,
  isTabAsking,
  detachTab,
  detachTabAt,
  moveTab,
} = useWindow()

const frame = useWindowFrame()

const bar = useWindowBar({ menus, vaultList, runCommand, getTarget, commandDeps })

const linkMenu = useTabLinkMenu({
  tabLinks,
  getTabs: () => windowTabs.tabs.value,
  getKind: (id) => windowTabs.getTab(id)?.kind.kind,
  isLinkable: (id) => windowTabs.getTab(id)?.kind.navigate !== undefined,
  onDetach: detachTabAt,
  windowId,
  isAsking: isTabAsking,
  onMove: moveTab,
  onHighlight: highlightWindow,
})

/* --------------------------------- Hooks ---------------------------------- */
function onOpenSettings(event: Event): void {
  const detail = (event as CustomEvent<{ filter?: string; pluginId?: string }>).detail
  if ((detail?.filter || detail?.pluginId) && openSettings) {
    openSettings(detail.pluginId, detail.filter)
    return
  }
  runCommand(COMMAND.openSettings, getTarget())
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

function onShowTab(id: string) {
  windowTabs.onTabShown(id)
}

function onTabContextMenu(tabId: string, position: { x: number; y: number }) {
  linkMenu.open(tabId, position)
}
</script>

<template>
  <!-- The window is drawn once the plugins of the interface have started or the wait for them has
       passed, so the layout does not move when the bar is drawn. -->
  <LoadingIndicator v-if="!isStarted" class="starting" :label="words.appName" />

  <main v-else :class="{ 'is-highlighted': isHighlighted }">
    <TitleBarHost
      :view="titleBarView"
      :app-name="words.appName"
      :vault-name="bar.vaultName.value"
      :platform="frame.platform.value"
      :is-maximized="frame.isMaximized.value"
      :is-fullscreen="frame.isFullscreen.value"
      :commands="bar.commands.value"
      :vaults="bar.vaults.value"
      @run-command="bar.runCommand"
      @switch-vault="bar.switchVault"
      @minimize-window="frame.minimizeWindow"
      @toggle-maximize-window="frame.toggleMaximizeWindow"
      @close-window="frame.closeWindow"
    />

    <CoreFailureNotice :failure="failure" />

    <WorkspaceLayout
      v-model="layout"
      class="below"
      :tabs="windowTabs.tabs.value"
      :window-edge="titleBarView ? undefined : frame.windowEdge.value"
      @close="onCloseTab"
      @show="onShowTab"
      @tab-context-menu="onTabContextMenu"
      @tear-off="detachTab"
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
        <BootstrapFailure
          v-if="bootstrapFailure"
          :message="bootstrapFailure"
          @retry="retryBootstrap"
        />
        <WelcomeScreen
          v-else-if="isLayoutReady || !vaultList.showing"
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
      :can-detach="linkMenu.canDetach.value"
      :windows="linkMenu.windows.value"
      :is-asking="linkMenu.isAsking.value"
      :linked-target-title="linkMenu.linkedTargetTitle.value"
      :get-tab-icon="getTabIcon"
      :get-kind-icon="getKindIcon"
      @link="linkMenu.link"
      @unlink="linkMenu.unlink"
      @detach="linkMenu.detach"
      @move="linkMenu.move"
      @highlight="linkMenu.highlight"
      @close="linkMenu.close"
    />

    <StatusBarHost :view="statusBarView" />

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
      :run-choice="runChoice"
    />
  </main>
</template>

<style scoped>
main {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

/* Another window offers to move a tab into this one. */
main.is-highlighted::after {
  content: '';
  position: fixed;
  inset: 0;
  z-index: var(--numen-lift-menu);
  border: 3px solid var(--numen-accent);
  pointer-events: none;
}

.starting {
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
