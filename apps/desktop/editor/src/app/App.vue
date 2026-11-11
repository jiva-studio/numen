<script setup lang="ts">
import type { Caught } from '@/shared/api/errors'
/**
 * The window: one vault, and tabs to divide the screen between.
 *
 * The window's structure is defined in `window.ts`. This file holds what the user
 * sees of it, and the binding between the two.
 */
import { WorkspaceLayout } from '@numen/workbench'
import { SlotViewHost } from '@numen/plugin-loader'
import { LoadingIndicator } from '@numen/ui'
import { formatError } from '@/shared/api/errors'
import '@numen/ui/styles.css'
import '@numen/workbench/styles.css'
import { UnsavedChangesPrompt } from '@/features/file-conflict'
import CoreFailureNotice from './CoreFailureNotice.vue'
import { CommandPalette } from '@/features/command-palette'
import { TabLinkIndicator } from '@/features/tab-linking'
import { WelcomeScreen } from '@/pages/welcome'
import { BootstrapFailure } from '@/features/app-start'
import { getUserAgent } from './window/getUserAgent'
import { getWindowParts } from './getWindowParts'
import { getWindowPartsTabMenu } from '@/entities/tab'
import { COMMAND } from '@/entities/command'
import { WORDS as words } from '@/shared/ui/words'
import { WINDOW_WORDS } from './window/words'
import { WORDS as note } from '@/entities/note'

/* --------------------------------- State ---------------------------------- */
/** What the notes still unwritten are put in: the window's words and a note's. */
const unsaved = {
  going: words.going,
  keep: note.keep,
  take: note.take,
  later: words.later,
}

const parts = getWindowParts()
const { frame } = parts.base
const { windowTabs } = parts.base.tabs
const { tabLinks } = parts.base.tabs
const { closeTab } = windowTabs
const { layout } = windowTabs.panes
const { moves, openTabs, linkSession, openCurrentVault } = parts.tabs
const { highlightWindow, detachTab } = moves
const { getTabIcon } = openTabs
const { isHighlighted } = linkSession
const {
  commands,
  palette: search,
  commandDeps,
  pluginWays,
  runCommand,
  runChoice,
} = parts.commands.commands
const { getTarget } = parts.commands
const { findFileIcon } = parts.kinds
const { slots, windowCommands, readNoteKind } = parts.frame
const { titleBarView, statusBarView } = parts.frame.views
const { fileFlush, getTitle } = parts.services.editing
const { failure } = parts.services.window.health
const { vaultList } = parts.services.vaults
const { isStarted, failure: bootstrapFailure, retry: retryBootstrap } = parts.bootstrap
const { isReady: isLayoutReady } = parts.tabs.layout

const tabMenu = getWindowPartsTabMenu({
  moveCommand: COMMAND.moveTabToWindow,
  windowTabs,
  onHighlight: highlightWindow,
  slots,
  commands: windowCommands,
  readNoteKind,
  reportError: (error: Caught) => commandDeps.writeMessage(formatError(error), 'error'),
})

/* -------------------------------- Handlers -------------------------------- */
function onCloseTab(id: string, hold?: () => void) {
  closeTab(id, hold)
}

function onShowTab(id: string) {
  windowTabs.panes.onTabShown(id)
}
</script>

<template>
  <!-- The window is drawn once the plugins of the window slots have started or the wait for them has
       passed, so the layout does not move when the bar is drawn. -->
  <LoadingIndicator v-if="!isStarted" class="starting" :label="WINDOW_WORDS.appName" />

  <main v-else :class="{ 'is-highlighted': isHighlighted }">
    <SlotViewHost :view="titleBarView" />

    <CoreFailureNotice :failure="failure" />

    <WorkspaceLayout
      v-model="layout"
      class="below"
      :tabs="windowTabs.tabs.tabs.value"
      :window-edge="titleBarView ? undefined : frame.windowEdge.value"
      :tab-menu="tabMenu.menu.value"
      @close="onCloseTab"
      @show="onShowTab"
      @tab-context-menu="tabMenu.open"
      @tab-menu-choose="tabMenu.choose"
      @tab-menu-dismiss="tabMenu.dismiss"
      @tab-menu-highlight="tabMenu.highlight"
      @tear-off="detachTab"
    >
      <template #tab-menu-icon="{ id }">
        <component
          :is="tabMenu.getIcon(id)"
          v-if="tabMenu.getIcon(id)"
          class="tab-menu-icon"
          aria-hidden="true"
        />
      </template>

      <template #icon="{ id }">
        <component :is="getTabIcon(id)" v-if="getTabIcon(id)" class="tab-icon" />
        <TabLinkIndicator
          :is-linked="tabLinks.getGroupOf(id) !== null"
          :color-index="tabLinks.getGroupOf(id)?.colorIndex"
        />
      </template>

      <template #tab="{ id }">
        <component
          :is="windowTabs.tabs.getTab(id)!.kind.pane"
          v-if="windowTabs.tabs.getTab(id)"
          :state="windowTabs.tabs.getTab(id)!.state"
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
          :tabs="windowTabs.tabs.tabs.value"
          :commands="commands"
          :search="search"
          :doing="commandDeps"
          :get-target="getTarget"
          :plugin-ways="pluginWays"
          :run-command="runCommand"
          :open-current-vault="openCurrentVault"
          :agent="getUserAgent()"
        />
        <div v-else />
      </template>
    </WorkspaceLayout>

    <SlotViewHost :view="statusBarView" />

    <UnsavedChangesPrompt
      :conflicts="fileFlush.conflicts.value"
      :called="getTitle"
      :words="unsaved"
    />

    <CommandPalette
      :commands="commands"
      :search="search"
      :doing="commandDeps"
      :get-target="getTarget"
      :places="{ findFileIcon }"
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
  --ring-width: 0.1875rem;
  content: '';
  position: fixed;
  inset: 0;
  z-index: var(--numen-lift-menu);
  border: var(--ring-width) solid var(--numen-accent);
  border-radius: var(--numen-window-radius);
  pointer-events: none;
}

.starting {
  height: 100vh;
}

.below {
  flex: 1;
  min-height: 0;
}

.tab-menu-icon {
  --size: 0.875rem;
  inline-size: var(--size);
  block-size: var(--size);
  stroke-width: 1.75;
}

/* Lucide draws on a 24 grid, and the stroke is given in those units. */
.tab-icon {
  inline-size: 100%;
  block-size: 100%;
  stroke-width: 1.75;
  opacity: 0.75;
}
</style>
